package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/input"
	"canerollss/ports/output"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type PaymentReconciliationUseCase struct {
	orderRepo   output.OrderRepository
	cashRepo    output.CashRegisterRepository
	paymentPort output.PaymentProvider
}

func NewPaymentReconciliationUseCase(
	orderRepo output.OrderRepository,
	cashRepo output.CashRegisterRepository,
	paymentPort output.PaymentProvider,
) *PaymentReconciliationUseCase {
	return &PaymentReconciliationUseCase{
		orderRepo:   orderRepo,
		cashRepo:    cashRepo,
		paymentPort: paymentPort,
	}
}

// ReconcileCustomerOrder evaluates whether an incoming bank/CVU transfer matches a specific customer pending order.
func (u *PaymentReconciliationUseCase) ReconcileCustomerOrder(
	ctx context.Context,
	orderID uint,
	opts input.DisambiguationOptions,
) (*domain.ReconciliationResult, error) {
	order, err := u.orderRepo.GetByID(orderID)
	if err != nil || order == nil {
		return nil, errors.New("pedido no encontrado")
	}

	if order.Status == domain.OrderStatusPaid || order.Status == domain.OrderStatusCompleted {
		return &domain.ReconciliationResult{
			Status:  domain.ReconciliationStatusMatched,
			OrderID: order.ID,
			Message: fmt.Sprintf("¡Tu pedido #%d ya está confirmado y pagado! Estamos preparando tus roles de canela 🤎", order.ID),
		}, nil
	}

	if order.Status != domain.OrderStatusPendingPayment {
		return &domain.ReconciliationResult{
			Status:  domain.ReconciliationStatusNotFoundYet,
			OrderID: order.ID,
			Message: fmt.Sprintf("El pedido #%d se encuentra en estado %s y no está pendiente de pago.", order.ID, order.Status),
		}, nil
	}

	// 1. If customer supplied a specific operation ID / voucher reference
	trimmedOpID := strings.TrimSpace(opts.OperationID)
	if trimmedOpID != "" {
		tx, err := u.paymentPort.GetTransferByID(ctx, trimmedOpID)
		if err != nil || tx == nil || !strings.EqualFold(tx.Status, "approved") {
			return &domain.ReconciliationResult{
				Status:  domain.ReconciliationStatusNotFoundYet,
				OrderID: order.ID,
				Message: fmt.Sprintf("No encontramos ninguna transferencia aprobada con el número de operación %s. Por favor revisá los dígitos del comprobante.", trimmedOpID),
			}, nil
		}

		// Anti double-spending check
		if existing, _ := u.orderRepo.GetByExternalPaymentID(tx.ID); existing != nil && existing.ID != order.ID {
			return &domain.ReconciliationResult{
				Status:  domain.ReconciliationStatusAlreadyUsed,
				OrderID: order.ID,
				Message: fmt.Sprintf("El comprobante de operación %s ya fue utilizado previamente para el pedido #%d.", tx.ID, existing.ID),
			}, nil
		}

		// Amount match check
		if tx.Amount != order.TotalAmount {
			return &domain.ReconciliationResult{
				Status:  domain.ReconciliationStatusAmountMismatch,
				OrderID: order.ID,
				Message: fmt.Sprintf("El monto recibido en la operación %s es de $%.2f pero el total de tu pedido es de $%.2f.", tx.ID, tx.Amount, order.TotalAmount),
			}, nil
		}

		// Successful match
		_ = u.markOrderPaid(order, tx)
		return &domain.ReconciliationResult{
			Status:             domain.ReconciliationStatusMatched,
			OrderID:            order.ID,
			MatchedTransaction: tx,
			CandidateCount:     1,
			Message:            fmt.Sprintf("¡Pago de $%.2f recibido con éxito! Tu pedido #%d ya está confirmado y entra en producción 🤎🍥", tx.Amount, order.ID),
		}, nil
	}

	// 2. Query approved transfers within the lookback window
	fromTime := order.CreatedAt.Add(-15 * time.Minute)
	if fromTime.IsZero() {
		fromTime = time.Now().Add(-15 * time.Minute)
	}
	toTime := time.Now().Add(5 * time.Minute)

	filter := output.TransferSearchFilter{
		FromDate: fromTime,
		ToDate:   toTime,
		Status:   "approved",
	}

	transfers, err := u.paymentPort.SearchApprovedTransfers(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error consultando transferencias: %w", err)
	}

	// Filter out already consumed transfers and currency mismatch
	var availableTransfers []domain.PaymentTransaction
	for _, tx := range transfers {
		if tx.Currency != "" && !strings.EqualFold(tx.Currency, "ARS") {
			continue
		}
		if existing, _ := u.orderRepo.GetByExternalPaymentID(tx.ID); existing != nil && existing.ID != order.ID {
			continue // Already consumed by another order
		}
		availableTransfers = append(availableTransfers, tx)
	}

	// Filter candidates matching amount
	var amountCandidates []domain.PaymentTransaction
	for _, tx := range availableTransfers {
		if tx.Amount == order.TotalAmount {
			amountCandidates = append(amountCandidates, tx)
		}
	}

	// If customer specified sender name or ID, apply filter
	filterSenderName := strings.TrimSpace(strings.ToLower(opts.PayerName))
	filterSenderID := strings.TrimSpace(opts.PayerID)
	if filterSenderName != "" || filterSenderID != "" {
		var senderCandidates []domain.PaymentTransaction
		for _, tx := range amountCandidates {
			nameMatch := filterSenderName == "" || strings.Contains(strings.ToLower(tx.PayerName), filterSenderName)
			idMatch := filterSenderID == "" || strings.Contains(tx.PayerIdentification, filterSenderID)
			if nameMatch && idMatch {
				senderCandidates = append(senderCandidates, tx)
			}
		}
		if len(senderCandidates) == 1 {
			amountCandidates = senderCandidates
		}
	}

	switch len(amountCandidates) {
	case 1:
		matched := amountCandidates[0]
		_ = u.markOrderPaid(order, &matched)
		return &domain.ReconciliationResult{
			Status:             domain.ReconciliationStatusMatched,
			OrderID:            order.ID,
			MatchedTransaction: &matched,
			CandidateCount:     1,
			Message:            fmt.Sprintf("¡Transferencia de $%.2f identificada con éxito! Tu pedido #%d ha sido confirmado 🤎🍥", matched.Amount, order.ID),
		}, nil

	case 0:
		// Check if any transfer was received with an amount mismatch
		if len(availableTransfers) > 0 {
			var mismatchedAmounts []string
			for _, t := range availableTransfers {
				mismatchedAmounts = append(mismatchedAmounts, fmt.Sprintf("$%.2f", t.Amount))
			}
			return &domain.ReconciliationResult{
				Status:  domain.ReconciliationStatusAmountMismatch,
				OrderID: order.ID,
				Message: fmt.Sprintf("Detectamos transferencias recientes por (%s), pero el monto exacto de tu pedido es de $%.2f. Si realizaste el pago por otro importe o desde otra cuenta, por favor pasame el comprobante o número de operación.", strings.Join(mismatchedAmounts, ", "), order.TotalAmount),
			}, nil
		}

		return &domain.ReconciliationResult{
			Status:  domain.ReconciliationStatusNotFoundYet,
			OrderID: order.ID,
			Message: fmt.Sprintf("Aún no visualizamos la transferencia de $%.2f en nuestra cuenta de Mercado Pago. Las transferencias entre bancos o billeteras a veces demoran 1 o 2 minutos. Aguardá un instante y volvé a decirme 'ya pagué' o pasame el comprobante.", order.TotalAmount),
		}, nil

	default:
		// Multiple unassigned matches for the exact amount
		return &domain.ReconciliationResult{
			Status:         domain.ReconciliationStatusAmbiguousMultipleMatches,
			OrderID:        order.ID,
			CandidateCount: len(amountCandidates),
			Message:        fmt.Sprintf("Detectamos más de una transferencia por $%.2f en este momento. Para asegurarnos de acreditar la tuya, ¿podrías indicarme el número de operación de tu comprobante o el nombre del titular de la cuenta emisora?", order.TotalAmount),
		}, nil
	}
}

// ReconcileIncomingTransfer reconciles an incoming transfer event (e.g. from webhook notification).
func (u *PaymentReconciliationUseCase) ReconcileIncomingTransfer(
	ctx context.Context,
	operationID string,
) (*domain.ReconciliationResult, error) {
	tx, err := u.paymentPort.GetTransferByID(ctx, operationID)
	if err != nil || tx == nil || !strings.EqualFold(tx.Status, "approved") {
		return &domain.ReconciliationResult{
			Status:  domain.ReconciliationStatusNotFoundYet,
			Message: "Transferencia no encontrada o aún no aprobada.",
		}, nil
	}

	// Check if already used
	if existing, _ := u.orderRepo.GetByExternalPaymentID(tx.ID); existing != nil {
		return &domain.ReconciliationResult{
			Status:  domain.ReconciliationStatusAlreadyUsed,
			OrderID: existing.ID,
			Message: fmt.Sprintf("La operación %s ya fue asociada al pedido #%d.", tx.ID, existing.ID),
		}, nil
	}

	// Find open register
	activeReg, err := u.cashRepo.GetActive()
	if err != nil || activeReg == nil {
		return &domain.ReconciliationResult{
			Status:  domain.ReconciliationStatusNotFoundYet,
			Message: "No hay una caja registradora abierta para conciliar pedidos.",
		}, nil
	}

	// Look for candidate pending orders in current register
	orders, err := u.orderRepo.GetByRegisterID(activeReg.ID)
	if err != nil {
		return nil, err
	}

	var matchingOrders []*domain.Order
	for i := range orders {
		if orders[i].Status == domain.OrderStatusPendingPayment && orders[i].TotalAmount == tx.Amount {
			matchingOrders = append(matchingOrders, &orders[i])
		}
	}

	if len(matchingOrders) == 1 {
		matchedOrder := matchingOrders[0]
		_ = u.markOrderPaid(matchedOrder, tx)
		return &domain.ReconciliationResult{
			Status:             domain.ReconciliationStatusMatched,
			OrderID:            matchedOrder.ID,
			MatchedTransaction: tx,
			CandidateCount:     1,
			Message:            fmt.Sprintf("Pedido #%d conciliado automáticamente vía webhook por $%.2f.", matchedOrder.ID, tx.Amount),
		}, nil
	}

	if len(matchingOrders) > 1 {
		return &domain.ReconciliationResult{
			Status:         domain.ReconciliationStatusAmbiguousMultipleMatches,
			CandidateCount: len(matchingOrders),
			Message:        fmt.Sprintf("Se encontraron múltiples pedidos pendientes con el monto $%.2f. Esperando desambiguación.", tx.Amount),
		}, nil
	}

	return &domain.ReconciliationResult{
		Status:  domain.ReconciliationStatusNotFoundYet,
		Message: fmt.Sprintf("No se encontró ningún pedido pendiente de pago por $%.2f en la caja activa.", tx.Amount),
	}, nil
}

func (u *PaymentReconciliationUseCase) markOrderPaid(order *domain.Order, tx *domain.PaymentTransaction) error {
	order.Status = domain.OrderStatusPaid
	order.ExternalPaymentID = tx.ID
	order.PaymentMethod = "TRANSFER_ALIAS"
	now := time.Now()
	order.PaidAt = &now
	return u.orderRepo.Update(order)
}
