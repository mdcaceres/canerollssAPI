# Feature Specification: Automatic Transfer Reconciliation (Mercado Pago Alias/CVU)

**Feature Branch**: `004-transfer-reconciliation`

**Created**: 2026-10-16

**Status**: Draft

**Input**: User description: "Diseña e implementa el módulo de conciliación automática de transferencias bancarias y de billeteras virtuales recibidas en nuestro Alias/CVU de Mercado Pago (Caneroll.mp, titular: Ivonne Abreu), integrado al flujo conversacional del agente de WhatsApp y respetando estrictamente nuestra Arquitectura Hexagonal (Ports & Adapters) y Clean Architecture en Go."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Order Confirmation with Direct Transfer Instructions (Priority: P1)

As a CaneRollss customer on WhatsApp confirming my order, I want to receive an exact order summary with the official payment instructions for direct bank/wallet transfer (total amount, Alias `Caneroll.mp`, and account holder `Ivonne Abreu`), with my order safely stored in `PENDING_PAYMENT` status so that I can immediately transfer the money from my banking app.

**Why this priority**: Direct transfer to Alias `Caneroll.mp` is the primary and only customer payment method used in CaneRollss (avoiding checkout commissions and card surcharges). Without accurate instructions and a registered pending order, no transfers can be attributed or fulfilled.

**Independent Test**: Can be tested by creating an order via customer dialogue, verifying the order is registered in `PENDING_PAYMENT` with matching total, and that the customer receives the official payment details with Alias and account holder name.

**Acceptance Scenarios**:

1. **Given** a customer with confirmed flavors, quantities, and delivery option (pickup or delivery fee calculated), **When** the order creation process is completed, **Then** the order is stored with status `PENDING_PAYMENT`, exact total amount (items + delivery fee), and the customer receives the official CaneRollss receipt showing the total, Alias `Caneroll.mp`, and account holder `Ivonne Abreu`.
2. **Given** an order created in `PENDING_PAYMENT`, **When** payment instructions are delivered, **Then** no third-party checkout URLs or external card links are presented to the customer as mandatory payment methods.

---

### User Story 2 - On-Demand Conversational Transfer Reconciliation (Priority: P1)

As a customer who just transferred money to Alias `Caneroll.mp`, I want to inform the WhatsApp agent ("Ya transferí", "Listo, mandé la plata desde la cuenta de Juan Pérez" or providing the transfer operation number) and have the system immediately verify the incoming transfer with Mercado Pago, automatically marking my order as `PAID` without requiring manual staff intervention.

**Why this priority**: Eliminates human delays in checking banking apps, provides instant gratification to the customer, and speeds up bakery production and dispatch.

**Independent Test**: Can be tested by simulating an order in `PENDING_PAYMENT` for $8.500, simulating an incoming approved transfer in Mercado Pago for $8.500 within the recent time window, sending "Ya transferí", and verifying the order status transitions to `PAID` with external payment ID recorded.

**Acceptance Scenarios**:

1. **Exact Match (Single Candidate)**: **Given** an active order in `PENDING_PAYMENT` for amount $X created within the active register window, **When** the customer states they have paid and exactly one unassigned approved incoming transfer of amount $X exists within the matching time window, **Then** the order is marked `PAID`, the transfer ID is permanently linked to the order to prevent reuse, and the agent confirms payment receipt and informs order preparation.
2. **Match by Operation ID / Reference**: **Given** an active order in `PENDING_PAYMENT` and multiple or ambiguous candidates, **When** the customer provides their transfer receipt number or operation ID, **Then** the system retrieves that specific transaction, validates that the amount matches the order total and has not been used, marks the order `PAID`, and confirms completion.
3. **Payer Name / CUIT Match**: **Given** an active order in `PENDING_PAYMENT` where the customer specifies the sender account holder name or national ID (CUIT/DNI), **When** the system matches an approved transfer with that sender metadata and matching amount, **Then** the order is reconciled to `PAID`.

---

### User Story 3 - Ambiguity Resolution and Discrepancy Handling (Priority: P2)

As a customer or store administrator, I want the system to gracefully handle cases where a payment cannot be uniquely reconciled (multiple transfers with the same amount, transfer not yet credited, partial amount, or previously consumed transfer) so that double-spending is impossible and the customer receives clear, actionable guidance.

**Why this priority**: Protects store finances against fraudulent reuse of payment receipts and prevents assigning the wrong customer's payment when two simultaneous orders share the same price.

**Independent Test**: Can be tested by simulating two distinct pending orders with identical amounts ($7.000) and only one incoming transfer, verifying the agent pauses automated matching and asks for the operation ID or payer name.

**Acceptance Scenarios**:

1. **Multiple Ambiguous Matches**: **Given** a pending order of $7.000 and two or more unassigned incoming transfers of $7.000 in the recent window, **When** the customer asks to verify payment without an operation ID, **Then** the system detects ambiguity, does NOT mark the order as paid, and asks the customer for the operation number or payer name from their receipt.
2. **Transfer Not Found Yet**: **Given** a pending order and no matching approved transfer in Mercado Pago, **When** the customer checks payment, **Then** the system informs that the transfer has not appeared yet in the account, advising that inter-bank transfers may take a couple of minutes and inviting them to retry shortly.
3. **Amount Mismatch**: **Given** an incoming transfer attributed to the customer whose amount is lower or higher than the pending order total, **When** evaluated, **Then** the system rejects automatic completion and notifies the discrepancy so the customer or human staff can resolve it.
4. **Anti-Double Spending (Already Reconciled)**: **Given** a transfer ID already associated with a completed order, **When** another verification attempts to use the same transfer ID, **Then** the system rejects the transaction as already consumed.

---

### User Story 4 - Reactive Webhook Ingestion & Notification (Priority: P3)

As a store operator, I want incoming Mercado Pago payment webhooks (`payment.created` / `payment.updated`) to trigger automatic reconciliation in real-time so that if a customer transfers before or right after confirming, the order is verified proactively.

**Why this priority**: Improves responsiveness by processing payments as soon as Mercado Pago pushes the notification, without depending solely on customer polling.

**Independent Test**: Can be tested by receiving a webhook event with a valid signature and payload, looking up the transaction, matching it to the corresponding pending order, and updating order status.

**Acceptance Scenarios**:

1. **Webhook Notification Ingestion**: **Given** a valid webhook event received from the payment provider with authentic signature, **When** processed, **Then** the system fetches the full transaction details from the provider.
2. **Proactive Order Reconcile**: **Given** an unassigned approved transfer received via webhook that uniquely matches an active pending order, **When** reconciled, **Then** the order is marked `PAID` and an outbound WhatsApp confirmation is dispatched.

---

### Edge Cases

- **Time Skew & Banking Delay**: Transfers initiated via DEBIN or inter-bank Coelsa network may have a credit timestamp 2 to 5 minutes earlier or later than the WhatsApp conversation. The matching window must allow a configurable lookback margin (default: 15 minutes before order creation up to current time).
- **Payment Split across Multiple Transfers**: A customer sending two separate transfers to cover one order (e.g., $4.000 + $3.000 for a $7.000 combo) is flagged for manual review rather than auto-closed.
- **Cash Register Closed**: If an incoming payment matches an order whose cash register was already closed, the order must still record the payment but flag a cash register variance for admin auditing.
- **Provider API Downtime / Rate Limits**: If the payment provider API returns transient 5xx or rate limit errors, reconciliation fails gracefully with a friendly message asking the customer to check back in a moment without cancelling the pending order.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST store orders confirmed in WhatsApp in status `PENDING_PAYMENT` with total order amount, customer reference, and active cash register association.
- **FR-002**: System MUST format payment instructions clearly displaying Total Amount, Alias `Caneroll.mp`, and Account Holder `Ivonne Abreu`.
- **FR-003**: System MUST provide an automated capability to search for approved incoming transfers in the payment provider within a configurable time window (default lookback: 15 minutes prior to order creation).
- **FR-004**: System MUST verify that incoming transfer currency matches the local currency (ARS) and status is approved.
- **FR-005**: System MUST perform exact amount matching between the pending order total and candidate incoming transfers.
- **FR-006**: System MUST prevent double-spending by verifying that a transfer operation ID has never been linked to another order in the database.
- **FR-007**: System MUST classify reconciliation evaluations into deterministic outcomes: `MATCHED`, `NOT_FOUND_YET`, `AMBIGUOUS_MULTIPLE_MATCHES`, `AMOUNT_MISMATCH`, and `ALREADY_USED`.
- **FR-008**: System MUST support disambiguation when the customer supplies the operation ID, transfer reference, payer name, or national ID (CUIT/DNI).
- **FR-009**: System MUST update order status to `PAID` and record `external_payment_id` upon successful match.
- **FR-010**: System MUST expose a conversational tool for the WhatsApp agent allowing on-demand payment verification with optional payer reference and operation ID arguments.
- **FR-011**: System MUST maintain conversational context and guide the user with warm, rioplatense messaging tailored to each reconciliation outcome.
- **FR-012**: System MUST support incoming webhook notifications with signature verification for secure asynchronous processing.
- **FR-013**: System MUST provide an in-memory/mock implementation of the payment search and retrieval contracts for local testing and isolated automated test suites.
- **FR-014**: System MUST adhere strictly to Hexagonal and Clean Architecture, keeping domain entities and business rules isolated from third-party vendor SDKs or specific JSON formats.

### Key Entities

- **PaymentTransaction**: Pure domain representation of an incoming transfer (Operation ID, Amount, Currency, Approval Date, Operation Type, Payer Name, Payer Identification/DNI/CUIT, Status).
- **ReconciliationResult**: Pure domain value object representing the matching outcome (Status: `MATCHED`, `NOT_FOUND_YET`, `AMBIGUOUS_MULTIPLE_MATCHES`, `AMOUNT_MISMATCH`, `ALREADY_USED`, MatchedTransaction, DiscrepancyMessage).
- **Order**: Core business entity representing a customer order, tracking its lifecycle status (`PENDING_PAYMENT`, `PAID`, `CANCELLED`), total amount, customer information, external payment ID, and payment method (`TRANSFER_ALIAS`).
- **CashRegister**: Daily operational entity tracking active sessions and sales volume.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 90% of unambiguous customer transfers to Alias `Caneroll.mp` are verified and transitioned to `PAID` in under 3 seconds upon the customer messaging "Ya transferí".
- **SC-002**: 0% false-positive matches (zero double-spent transfers and zero incorrect attribution of funds between concurrent orders of equal amount).
- **SC-003**: 100% of payment reconciliation business rules are testable using mock providers without network calls to external payment APIs.
- **SC-004**: Customers experiencing ambiguous or uncredited transfers receive immediate actionable instructions in under 2 seconds.
- **SC-005**: Complete end-to-end order flow from WhatsApp order confirmation to transfer verification executes successfully in automated test suites.

## Assumptions

- The business operates under a single verified Mercado Pago account with Alias `Caneroll.mp` and holder `Ivonne Abreu`.
- Mercado Pago API `/v1/payments/search` with appropriate OAuth `MP_ACCESS_TOKEN` provides access to incoming peer-to-peer / inter-bank transfer transactions (`operation_type: bank_transfer`, `money_transfer`, or similar standard MP types).
- Customers initiate transfers from any Argentine banking entity (CBU) or virtual wallet (CVU/Alias) into `Caneroll.mp`.
- All monetary transactions and order totals are expressed in Argentine Pesos (ARS).
- For local development and test runs, `USE_MOCK_PROVIDERS=true` simulates bank transfers without requiring production credentials or live banking movements.
