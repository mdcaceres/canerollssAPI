# External Ports & Adapters Contracts

Este documento define los contratos de interfaces en Go para la integración de servicios de entrega a domicilio y pasarelas de pago, implementando el patrón Ports & Adapters (Arquitectura Hexagonal).

---

## 1. Logistics Port: `DeliveryProvider`

### Go Interface (`ports/output/delivery_provider.go`)

```go
package output

import (
	"canerollss/core/domain"
	"context"
)

type DeliveryQuote struct {
	EstimatedFee float64 `json:"estimated_fee"`
	Currency     string  `json:"currency"`
	DurationMin  int     `json:"duration_min"`
	QuoteID      string  `json:"quote_id"`
}

type DeliveryConfirmation struct {
	ExternalDeliveryID string `json:"external_delivery_id"`
	TrackingURL        string `json:"tracking_url"`
	Status             string `json:"status"`
}

type DeliveryProvider interface {
	QuoteDelivery(ctx context.Context, address *domain.CustomerAddress) (*DeliveryQuote, error)
	CreateDelivery(ctx context.Context, order *domain.Order, address *domain.CustomerAddress) (*DeliveryConfirmation, error)
}
```

### Implementaciones:
1. **`MockDeliveryProvider` (`adapters/output/delivery/mock_delivery_provider.go`)**:
   - Activa por defecto en desarrollo y tests.
   - Retorna un costo fijo o configurable razonable (ej. `$1500.00` ARS, duración estimada 25 min, quote ID `mock-quote-xyz`).
   - Simula creación de envío retornando `external_delivery_id: "mock-uber-12345"` y `tracking_url: "https://tracking.uber.com/mock-uber-12345"`.
2. **`UberDirectProvider` (`adapters/output/delivery/uber_direct_provider.go`)**:
   - Esqueleto desacoplado que consume la API REST de Uber Direct v1 vía `net/http`.
   - Endpoint de cotización: `POST https://api.uber.com/v1/customers/{customer_id}/delivery_quotes`
   - Endpoint de creación: `POST https://api.uber.com/v1/customers/{customer_id}/deliveries`
   - Manejo de Bearer Token OAuth2 / API Key mediante variables de entorno `UBER_DIRECT_CUSTOMER_ID` y `UBER_DIRECT_CLIENT_SECRET`.

---

## 2. Payment Gateway Port: `PaymentProvider`

### Go Interface (`ports/output/payment_provider.go`)

```go
package output

import (
	"canerollss/core/domain"
	"context"
)

type PaymentPreferenceResult struct {
	PreferenceID string `json:"preference_id"`
	PaymentLink  string `json:"payment_link"`
}

type PaymentStatusResult struct {
	ExternalPaymentID string `json:"external_payment_id"`
	Status            string `json:"status"` // "approved", "pending", "rejected"
	IsPaid            bool   `json:"is_paid"`
}

type PaymentProvider interface {
	GeneratePaymentLink(ctx context.Context, order *domain.Order) (*PaymentPreferenceResult, error)
	VerifyPaymentStatus(ctx context.Context, externalPaymentID string) (*PaymentStatusResult, error)
}
```

### Implementaciones:
1. **`MockPaymentProvider` (`adapters/output/payment/mock_payment_provider.go`)**:
   - Activa por defecto en desarrollo y tests.
   - Genera enlaces de pago simulados: `https://sandbox.mercadopago.com/checkout/v1/redirect?pref_id=mock-pref-123`.
   - Permite verificar pagos retornando aprobado si el pref_id contiene ciertos prefijos o para cualquier ID simulado.
2. **`MercadoPagoProvider` (`adapters/output/payment/mercadopago_provider.go`)**:
   - Esqueleto desacoplado que consume la API REST de Mercado Pago Checkout Pro vía `net/http`.
   - Endpoint de preferencias: `POST https://api.mercadopago.com/checkout/preferences`
   - Endpoint de consulta de pago: `GET https://api.mercadopago.com/v1/payments/{id}`
   - Header: `Authorization: Bearer MP_ACCESS_TOKEN`
   - Integración con webhook en `POST /webhook/mercadopago` que procesa notificaciones de tipo `payment` o `topic=payment`.
