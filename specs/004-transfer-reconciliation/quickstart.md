# Quickstart Validation Guide: Automatic Transfer Reconciliation

**Feature**: `004-transfer-reconciliation`  

---

## Prerequisites

1. Go 1.22+ installed.
2. Local environment configured with `DB_DRIVER=sqlite` and `USE_MOCK_PROVIDERS=true`.
3. Valid `GEMINI_API_KEY` in `.env`.

---

## Validation Scenarios

### Scenario 1: Automated Unit Tests (Reconciliation Engine & Rules)

Run the comprehensive unit test suite covering all 5 reconciliation states (`MATCHED`, `NOT_FOUND_YET`, `AMBIGUOUS_MULTIPLE_MATCHES`, `AMOUNT_MISMATCH`, `ALREADY_USED`) and anti-double spending controls:

```bash
go test -v ./core/usecase/... -run TestPaymentReconciliation
```

**Expected Outcome**: All reconciliation branches pass with 100% assertions satisfied.

---

### Scenario 2: End-to-End WhatsApp Conversational Flow with Alias Transfer

Execute the simulated conversational test against the local server:

1. **Start backend**:
   ```bash
   go run main.go
   ```
2. **Execute Multi-Turn Conversation**:
   - **Step 1**: Send order confirmation to `POST /chat`:
     ```json
     {
       "name": "Matias",
       "phone": "5493519876543",
       "message": "Quiero pedir 1 combo Clasico para retirar",
       "reset": true
     }
     ```
     *Expected Response*: Order #1 created in `PENDING_PAYMENT`. Receipt formatted with Total *$7.000*, Alias `Caneroll.mp`, and Titular `Ivonne Abreu`.
   - **Step 2**: Seed simulated transfer in Mock Provider and tell Roli "Ya transferí":
     ```json
     {
       "name": "Matias",
       "phone": "5493519876543",
       "message": "Listo, ya te transferí los $7000 desde mi cuenta!",
       "reset": false
     }
     ```
     *Expected Response*: Roli detects the payment, confirms receipt, and announces that the order is being prepared. Database order status updates to `PAID` with `external_payment_id` recorded.

---

### Scenario 3: Ambiguity Disambiguation via Operation ID

1. Order with $7.000 total.
2. Two candidate transfers of $7.000 present in mock provider.
3. Customer says "Ya pagué".
4. *Expected Response*: Roli notices multiple matches and prompts for the operation number.
5. Customer replies "El número de operación es 99887766".
6. *Expected Response*: Roli matches the specific transaction #99887766 and transitions order to `PAID`.
