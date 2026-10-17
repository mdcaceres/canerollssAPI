# External HTTP Contract: Mercado Pago API & Webhook

**Feature**: `004-transfer-reconciliation`  

---

## 1. Mercado Pago Payments Search (`GET /v1/payments/search`)

### Request
```http
GET https://api.mercadopago.com/v1/payments/search?sort=date_created&criteria=desc&range=date_created&begin_date=2026-10-16T22:00:00Z&end_date=2026-10-16T22:30:00Z&status=approved HTTP/1.1
Authorization: Bearer MP_ACCESS_TOKEN
Accept: application/json
```

### Response (`200 OK`)
```json
{
  "paging": {
    "total": 1,
    "limit": 30,
    "offset": 0
  },
  "results": [
    {
      "id": 12345678901,
      "date_created": "2026-10-16T22:15:30.000-04:00",
      "date_approved": "2026-10-16T22:15:32.000-04:00",
      "operation_type": "money_transfer",
      "payment_type_id": "bank_transfer",
      "status": "approved",
      "status_detail": "accredited",
      "currency_id": "ARS",
      "transaction_amount": 8500.00,
      "payer": {
        "id": "987654321",
        "email": "juan.perez@example.com",
        "first_name": "Juan",
        "last_name": "Pérez",
        "identification": {
          "type": "DNI",
          "number": "35123456"
        }
      }
    }
  ]
}
```

---

## 2. Mercado Pago Webhook Ingestion (`POST /webhook/mercadopago`)

### Request
```http
POST /webhook/mercadopago HTTP/1.1
Host: api.canerollss.com
Content-Type: application/json
x-signature: ts=1729124400,v1=abcdef0123456789...

{
  "action": "payment.created",
  "api_version": "v1",
  "data": {
    "id": "12345678901"
  },
  "date_created": "2026-10-16T22:15:30Z",
  "id": 10987654321,
  "live_mode": true,
  "type": "payment",
  "user_id": "1122334455"
}
```

### Response
- `200 OK`: Acknowledged and scheduled for reconciliation.
- `400 Bad Request`: Malformed body or invalid HMAC signature.
