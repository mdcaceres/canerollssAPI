package payment

import (
	"bytes"
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type MercadoPagoProvider struct {
	accessToken string
	baseURL     string
	httpClient  *http.Client
}

func NewMercadoPagoProvider() *MercadoPagoProvider {
	baseURL := os.Getenv("MP_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.mercadopago.com"
	}
	return &MercadoPagoProvider{
		accessToken: os.Getenv("MP_ACCESS_TOKEN"),
		baseURL:     baseURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

type mpPreferenceItem struct {
	Title     string  `json:"title"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type mpPreferenceRequest struct {
	Items             []mpPreferenceItem `json:"items"`
	ExternalReference string             `json:"external_reference"`
}

type mpPreferenceResponse struct {
	ID       string `json:"id"`
	InitPoint string `json:"init_point"`
	SandboxInitPoint string `json:"sandbox_init_point"`
}

func (p *MercadoPagoProvider) GeneratePaymentLink(ctx context.Context, order *domain.Order) (*output.PaymentPreferenceResult, error) {
	if p.accessToken == "" {
		// Fallback to mock link if credentials are not configured
		prefID := fmt.Sprintf("mp-pref-%d", order.ID)
		return &output.PaymentPreferenceResult{
			PreferenceID: prefID,
			PaymentLink:  fmt.Sprintf("https://sandbox.mercadopago.com/checkout/v1/redirect?pref_id=%s", prefID),
		}, nil
	}

	var items []mpPreferenceItem
	for _, it := range order.Items {
		name := "Rol de canela"
		if it.Topping.Name != "" {
			name = it.Topping.Name
		}
		items = append(items, mpPreferenceItem{
			Title:     name,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
		})
	}
	if order.DeliveryCost > 0 {
		items = append(items, mpPreferenceItem{
			Title:     "Costo de envío a domicilio",
			Quantity:  1,
			UnitPrice: order.DeliveryCost,
		})
	}

	body, _ := json.Marshal(mpPreferenceRequest{
		Items:             items,
		ExternalReference: fmt.Sprintf("%d", order.ID),
	})

	url := fmt.Sprintf("%s/checkout/preferences", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mercadopago preference error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mercadopago returned status %d", resp.StatusCode)
	}

	var res mpPreferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	link := res.InitPoint
	if link == "" {
		link = res.SandboxInitPoint
	}

	return &output.PaymentPreferenceResult{
		PreferenceID: res.ID,
		PaymentLink:  link,
	}, nil
}

type mpPaymentResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

func (p *MercadoPagoProvider) VerifyPaymentStatus(ctx context.Context, externalPaymentID string) (*output.PaymentStatusResult, error) {
	if p.accessToken == "" {
		return &output.PaymentStatusResult{
			ExternalPaymentID: externalPaymentID,
			Status:            "approved",
			IsPaid:            true,
		}, nil
	}

	url := fmt.Sprintf("%s/v1/payments/%s", p.baseURL, externalPaymentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mercadopago payment inquiry failed: %d", resp.StatusCode)
	}

	var res mpPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &output.PaymentStatusResult{
		ExternalPaymentID: externalPaymentID,
		Status:            res.Status,
		IsPaid:            res.Status == "approved",
	}, nil
}

type mpPaymentItem struct {
	ID                int64   `json:"id"`
	DateApproved      string  `json:"date_approved"`
	OperationType     string  `json:"operation_type"`
	Status            string  `json:"status"`
	CurrencyID        string  `json:"currency_id"`
	TransactionAmount float64 `json:"transaction_amount"`
	Payer             struct {
		FirstName      string `json:"first_name"`
		LastName       string `json:"last_name"`
		Identification struct {
			Type   string `json:"type"`
			Number string `json:"number"`
		} `json:"identification"`
	} `json:"payer"`
}

type mpSearchResponse struct {
	Results []mpPaymentItem `json:"results"`
}

func (p *MercadoPagoProvider) toDomainTransaction(item mpPaymentItem) domain.PaymentTransaction {
	dateApp, _ := time.Parse(time.RFC3339, item.DateApproved)
	payerName := strings.TrimSpace(item.Payer.FirstName + " " + item.Payer.LastName)

	return domain.PaymentTransaction{
		ID:                  fmt.Sprintf("%d", item.ID),
		Amount:              item.TransactionAmount,
		Currency:            item.CurrencyID,
		DateApproved:        dateApp,
		OperationType:       item.OperationType,
		PayerName:           payerName,
		PayerIdentification: item.Payer.Identification.Number,
		Status:              item.Status,
	}
}

func (p *MercadoPagoProvider) SearchApprovedTransfers(ctx context.Context, filter output.TransferSearchFilter) ([]domain.PaymentTransaction, error) {
	if p.accessToken == "" {
		return nil, nil
	}

	status := filter.Status
	if status == "" {
		status = "approved"
	}

	url := fmt.Sprintf("%s/v1/payments/search?sort=date_created&criteria=desc&status=%s", p.baseURL, status)
	if !filter.FromDate.IsZero() && !filter.ToDate.IsZero() {
		url += fmt.Sprintf("&range=date_created&begin_date=%s&end_date=%s", filter.FromDate.Format(time.RFC3339), filter.ToDate.Format(time.RFC3339))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mercadopago payment search failed: status %d", resp.StatusCode)
	}

	var searchRes mpSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return nil, err
	}

	var transactions []domain.PaymentTransaction
	for _, it := range searchRes.Results {
		transactions = append(transactions, p.toDomainTransaction(it))
	}
	return transactions, nil
}

func (p *MercadoPagoProvider) GetTransferByID(ctx context.Context, operationID string) (*domain.PaymentTransaction, error) {
	if p.accessToken == "" {
		return nil, nil
	}

	url := fmt.Sprintf("%s/v1/payments/%s", p.baseURL, operationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.accessToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mercadopago get payment failed: status %d", resp.StatusCode)
	}

	var item mpPaymentItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}

	tx := p.toDomainTransaction(item)
	return &tx, nil
}

