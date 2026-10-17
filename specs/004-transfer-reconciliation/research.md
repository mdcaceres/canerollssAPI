# Research & Decisions: Automatic Transfer Reconciliation (Mercado Pago Alias/CVU)

**Feature**: `004-transfer-reconciliation`  
**Date**: 2026-10-16  
**Status**: Completed  

---

## Decision 1: Mercado Pago Payment Search API for Alias/CVU Transfers

- **Problem**: Transfers made to the store's Alias (`Caneroll.mp`) or CVU arrive in Mercado Pago as peer-to-peer / inter-bank transfers (via Coelsa/DEBIN) without an `external_reference` linking them to a CaneRollss order ID.
- **Decision**: Query the Mercado Pago API endpoint:
  ```http
  GET /v1/payments/search?sort=date_created&criteria=desc&range=date_created&begin_date={ISO8601}&end_date={ISO8601}&status=approved
  ```
  And for direct lookups by operation ID:
  ```http
  GET /v1/payments/{id}
  ```
- **Rationale**:
  - The `/v1/payments/search` endpoint allows filtering by approval date ranges and status.
  - Transactions contain the required audit fields: `id`, `transaction_amount`, `currency_id`, `date_approved`, `operation_type`, `payment_method_id`, and `payer` details (`first_name`, `last_name`, `identification.number`).
- **Alternatives Considered**:
  - *Mercado Pago Reports API (Banking CSV)*: Rejected because generation is asynchronous, scheduled in batches, and has delays of up to hours, making it impossible to provide instantaneous conversational confirmations.
  - *Checkout Pro Links*: Rejected by business definition; CaneRollss operates on direct Alias transfers to avoid payment gateway fees and surcharges.

---

## Decision 2: Multi-Turn Reconciliation Strategy and State Machine

- **Problem**: When a customer says "Ya transferí", we need a deterministic algorithm to match the order without false positives, avoiding double-spending and resolving concurrent orders with equal amounts.
- **Decision**: Implement a 5-outcome classification in `PaymentReconciliationUseCase`:
  1. `MATCHED`:
     - Case A (Unambiguous amount): Exactly 1 unlinked transfer found matching `order.total_amount` within `order.created_at - 15m` to `now`.
     - Case B (Direct ID): Customer provided operation ID/voucher number; retrieved transaction matches `order.total_amount` and is unlinked.
     - Case C (Sender disambiguation): Multiple transfers found, but customer provided sender name or DNI/CUIT matching exactly 1 transfer.
  2. `NOT_FOUND_YET`: No unlinked approved transfers found in the lookback window.
  3. `AMBIGUOUS_MULTIPLE_MATCHES`: Multiple unlinked transfers match the exact amount within the time window and no disambiguating sender info was provided.
  4. `AMOUNT_MISMATCH`: The customer provided an operation ID or sender match, but the transferred amount does not match `order.total_amount`.
  5. `ALREADY_USED`: The operation ID supplied or matched has already been reconciled to another order in the database (anti-fraud / anti-double spending).
- **Rationale**:
  - Guarantees zero false positives.
  - Transparently communicates the situation to the customer via the WhatsApp agent with actionable suggestions.
- **Alternatives Considered**:
  - *Auto-assigning the oldest match*: Rejected because if Customer A ($7.000) and Customer B ($7.000) order simultaneously and Customer B transfers first, Customer A's order could mistakenly be marked as paid.

---

## Decision 3: Clean & Hexagonal Architecture Boundaries

- **Problem**: Avoid leaking Mercado Pago JSON schemas, HTTP client libraries, or webhook structs into the domain or use case layer.
- **Decision**:
  1. **Domain (`core/domain`)**:
     - `PaymentTransaction`: Pure struct with `ID`, `Amount`, `Currency`, `DateApproved`, `OperationType`, `PayerName`, `PayerID`, `Status`.
     - `ReconciliationStatus`: String enum (`MATCHED`, `NOT_FOUND_YET`, `AMBIGUOUS_MULTIPLE_MATCHES`, `AMOUNT_MISMATCH`, `ALREADY_USED`).
     - `ReconciliationResult`: Pure value object containing status, matched transaction, order ID, and explanation message.
  2. **Ports (`ports/output` & `ports/input`)**:
     - `output.PaymentProvider`: Extended with `SearchApprovedTransfers(ctx, filter)` and `GetTransferByID(ctx, id)`.
     - `input.PaymentReconciliationUseCasePort`: Interface defining the reconciliation usecase methods for consumption by both WhatsApp Agent tool and Webhook handler.
  3. **Use Case (`core/usecase`)**:
     - `PaymentReconciliationUseCase`: Orchestrates lookup, anti-double-spending checks via `orderRepo`, order status transition to `PAID`, and cash register updates.
  4. **Adapters (`adapters/output/payment` & `adapters/input/http/handlers`)**:
     - `MercadoPagoProvider`: Translates MP HTTP responses into domain `PaymentTransaction`.
     - `MockPaymentProvider`: Maintains in-memory transfer ledger for testing.
     - `MercadoPagoWebhookHandler`: Ingests HTTP webhook payloads with optional HMAC validation.
- **Rationale**: Strict adherence to Hexagonal / Clean architecture principles, allowing complete testability in memory and zero infrastructure coupling in domain rules.

---

## Decision 4: Deterministic Mocking for Automated Testing & Local Dev

- **Problem**: Testing bank transfers requires either live Argentine bank accounts or an in-memory controllable simulator.
- **Decision**: Equip `MockPaymentProvider` with an in-memory repository of simulated transfers:
  ```go
  func (m *MockPaymentProvider) SeedTransfer(t domain.PaymentTransaction)
  func (m *MockPaymentProvider) ClearTransfers()
  ```
  When `USE_MOCK_PROVIDERS=true`, tests and local simulations can inject incoming transfers and test all 5 reconciliation branches without external dependencies.
- **Rationale**: Ensures fast, deterministic unit, contract, and E2E testing without external network flakiness.

---

## Decision 5: WhatsApp Agent Integration & Prompt Copy

- **Problem**: When a customer confirms an order or states they paid, the agent must present the official CaneRollss receipt and execute the reconciliation tool seamlessly.
- **Decision**:
  - Remove all Checkout Pro links from `cinnamon_sales_agent.tmpl` and `whatsapp_agent_usecase.go`.
  - The receipt format strictly outputs:
    ```text
    ✨ ¡Gracias por tu pedido, {Nombre}! ✨

    🧾 Pedido: #{ID} ({Resumen de items})
    🛵 Envío: {Modalidad} ({Dirección})

    Total: *${Monto}*

    📲 Alias: Caneroll.mp
    👤 A nombre de: Ivonne Abreu

    Una vez realizada la transferencia, avisame "ya pagué" o mandame el comprobante 🤎
    ```
  - In `whatsapp_agent_usecase.go`, the tool `verificar_estado_pago` calls `PaymentReconciliationUseCase.ReconcileCustomerOrder` and returns structured feedback for Roli to inform the customer.
