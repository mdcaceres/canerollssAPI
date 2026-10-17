# Quickstart Validation Guide: WhatsApp AI-Powered Order Automation

This guide provides runnable validation steps to verify the WhatsApp conversational agent, address management, delivery quotation, and payment link flows end-to-end.

---

## Prerequisites

1. **Go Environment**: Go 1.22+ or 1.26 installed.
2. **Database**: MySQL running locally or via Docker with schema initialized by `connection.go`.
3. **Environment Variables** (in `.env` or system environment):
   ```bash
   PORT=8080
   GEMINI_API_KEY="your-google-ai-api-key"
   WHATSAPP_VERIFY_TOKEN="canerollss_verify_secret"
   WHATSAPP_PHONE_NUMBER_ID="mock_phone_id"
   WHATSAPP_API_TOKEN="mock_token"
   USE_MOCK_PROVIDERS=true
   ```

---

## Scenario 1: Webhook Verification Challenge (Meta Verification)

**Goal**: Verify that Meta's Cloud API can handshake with the backend.

```bash
curl -X GET "http://localhost:8080/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=canerollss_verify_secret&hub.challenge=CHALLENGE_ACCEPTED_123"
```

**Expected Outcome**:
- HTTP Status: `200 OK`
- Response Body: `CHALLENGE_ACCEPTED_123`

---

## Scenario 2: Catalog & Flavor Inquiry (No "topping" jargon)

**Goal**: Ask for menu options and verify the AI presents flavors and prices using Argentine terminology without mentioning "topping".

**Payload**:
```bash
curl -X POST "http://localhost:8080/webhook/whatsapp" \
  -H "Content-Type: application/json" \
  -d '{
    "entry": [{
      "changes": [{
        "value": {
          "messages": [{
            "from": "5491155551234",
            "id": "wamid.msg1",
            "text": { "body": "Hola buenas! Qué sabores tienen y cuánto salen los combos?" },
            "type": "text"
          }]
        }
      }]
    }]
  }'
```

**Expected Outcome**:
- Webhook response: `200 OK` immediately (< 2s).
- Outbound WhatsApp response (logged in console or sent to Meta): Friendly message listing active toppings as "sabores" (Clásico, Dulce de leche, Avellana, Oreo, Pistacho premium) with individual and combo prices. Zero occurrences of the word "topping".

---

## Scenario 3: Multiple Address Registration & Delivery Quote

**Goal**: Save a new address for the client and request a delivery quote.

**Payload**:
```bash
curl -X POST "http://localhost:8080/webhook/whatsapp" \
  -H "Content-Type: application/json" \
  -d '{
    "entry": [{
      "changes": [{
        "value": {
          "messages": [{
            "from": "5491155551234",
            "id": "wamid.msg2",
            "text": { "body": "Guardá mi dirección de Casa: Av. Santa Fe 1234, piso 4B, timbre blanco en CABA. Cuánto me saldría el envío?" },
            "type": "text"
          }]
        }
      }]
    }]
  }'
```

**Expected Outcome**:
- Address persisted in `customer_address` table linked to `customer_phone = '5491155551234'`.
- Agent responds confirming the address was saved and provides the quoted shipping fee (e.g. `$1500.00 ARS` via `MockDeliveryProvider`).

---

## Scenario 4: Multi-Item Order Confirmation with Payment Link

**Goal**: Place an order with multiple items (e.g. 1 combo Nutella + 2 individuales Pistacho), confirm delivery modality, receive itemized breakdown and Mercado Pago payment link.

**Payload**:
```bash
curl -X POST "http://localhost:8080/webhook/whatsapp" \
  -H "Content-Type: application/json" \
  -d '{
    "entry": [{
      "changes": [{
        "value": {
          "messages": [{
            "from": "5491155551234",
            "id": "wamid.msg3",
            "text": { "body": "Quiero pedir 1 combo de Avellana y 2 individuales de Pistacho premium para enviar a mi Casa. Confirmo el pedido!" },
            "type": "text"
          }]
        }
      }]
    }]
  }'
```

**Expected Outcome**:
- Order created in database with:
  - `status = 'PENDING_PAYMENT'`
  - `delivery_type = 'DELIVERY'`
  - `address_id` linked to the saved address
  - 2 line items in `order_item`
  - `total_amount = subtotal_items + delivery_cost`
- WhatsApp reply contains the itemized total and the Mercado Pago checkout link (`https://sandbox.mercadopago.com/...`).

---

## Scenario 5: Payment Confirmation via Webhook

**Goal**: Process Mercado Pago IPN/webhook and verify order transitions to `PAID`.

**Payload**:
```bash
curl -X POST "http://localhost:8080/webhook/mercadopago" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "payment.created",
    "type": "payment",
    "data": { "id": "mock_payment_123" }
  }'
```

**Expected Outcome**:
- HTTP Status: `200 OK`.
- Associated `order` status updated to `PAID`.
- Confirmation message pushed to customer on WhatsApp: *"¡Pago confirmado! Estamos preparando tu pedido."*
