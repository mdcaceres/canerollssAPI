# Go Ports Contract: Payment Reconciliation

**Feature**: `004-transfer-reconciliation`  
**Package**: `ports/output` & `ports/input`  

---

## 1. Output Port (`ports/output/payment_provider.go`)

Defines the contract that any payment infrastructure adapter (Mercado Pago, Mock, etc.) must satisfy.

```go
package output

import (
    "canerollss/core/domain"
    "context"
    "time"
)

// TransferSearchFilter parameters for searching incoming transfers
type TransferSearchFilter struct {
    FromDate time.Time
    ToDate   time.Time
    Status   string // typically "approved"
}

type PaymentProvider interface {
    // SearchApprovedTransfers searches incoming peer-to-peer / bank transfers in the window
    SearchApprovedTransfers(ctx context.Context, filter TransferSearchFilter) ([]domain.PaymentTransaction, error)

    // GetTransferByID fetches a specific transfer by its unique operation ID
    GetTransferByID(ctx context.Context, operationID string) (*domain.PaymentTransaction, error)

    // Legacy/compatibility methods
    GeneratePaymentLink(ctx context.Context, order *domain.Order) (*PaymentPreferenceResult, error)
    VerifyPaymentStatus(ctx context.Context, externalPaymentID string) (*PaymentStatusResult, error)
}
```

---

## 2. Input Port (`ports/input/payment_reconciliation_port.go`)

Defines the use case contract callable by driving adapters (WhatsApp Agent tool and HTTP Webhook).

```go
package input

import (
    "canerollss/core/domain"
    "context"
)

type DisambiguationOptions struct {
    OperationID string // Customer-provided operation or voucher ID
    PayerName   string // Customer-provided sender account holder name
    PayerID     string // Customer-provided sender DNI/CUIT
}

type PaymentReconciliationUseCasePort interface {
    // ReconcileCustomerOrder evaluates payment for a customer's pending order
    ReconcileCustomerOrder(ctx context.Context, orderID uint, opts DisambiguationOptions) (*domain.ReconciliationResult, error)

    // ReconcileIncomingTransfer processes an incoming transfer event (e.g. from webhook)
    ReconcileIncomingTransfer(ctx context.Context, operationID string) (*domain.ReconciliationResult, error)
}
```
