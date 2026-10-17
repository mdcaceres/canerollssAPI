# Research & Architecture Decisions: WhatsApp AI-Powered Order Automation

This document captures the technical evaluations, architectural decisions, and rejected alternatives for the WhatsApp AI cinnamon roll sales agent.

---

## Decision 1: Asynchronous WhatsApp Cloud API Webhook Handling

- **Decision**: Implement `POST /webhook/whatsapp` returning `HTTP 200` immediately (< 2 seconds), ignoring Meta `statuses` events (`sent`, `delivered`, `read`), and delegating customer message processing to an asynchronous goroutine.
- **Rationale**: Meta WhatsApp Cloud API enforces strict delivery acknowledgement timeouts (typically 3-5 seconds). AI agent reasoning, function execution, and external logistics/payment quotes can take between 2 to 5 seconds. Failing to acknowledge within Meta's window causes repeated webhook retries and duplicate user responses.
- **Alternatives Considered**:
  - *Synchronous processing in the HTTP handler*: Rejected because Gemini Flash tool executions and network roundtrips exceed Meta's retry SLA.
  - *Full message queue infrastructure (RabbitMQ / Kafka / Redis Streams)*: Rejected as over-engineering for the current single-instance monolithic backend. In-process goroutines with memory buffers fulfill all throughput requirements cleanly.

---

## Decision 2: In-Memory Message Debouncing & Session State

- **Decision**: Implement `SessionManager` in memory using `sync.RWMutex`, a message accumulation buffer `[]string`, and a `time.Timer` with a configurable 4-second delay per `wa_id`.
- **Rationale**: WhatsApp users frequently send thoughts across multiple fragmented messages in quick succession (e.g., "Hola" -> "quiero pedir" -> "1 combo Nutella" -> "para delivery a mi casa"). Resetting the 4s timer on each arrival consolidates the messages into a single prompt for Gemini, preventing fragmented AI replies and reducing API token costs by ~70%.
- **Alternatives Considered**:
  - *No debouncing (immediate reaction per message)*: Rejected due to disjointed AI answers and hallucinated assumptions from incomplete messages.
  - *Redis / Distributed Key-Value store*: Rejected to keep the application self-contained. The existing architecture runs on a single Go backend instance. Ephemeral conversation memory is acceptable to lose on server restart.

---

## Decision 3: Official Gemini Go SDK & Model

- **Decision**: Use `google.golang.org/genai` with model `gemini-2.5-flash`.
- **Rationale**: `gemini-2.5-flash` provides sub-second first-token latency, native tool calling (Function Calling), rich Spanish reasoning (supporting Argentine rioplatense slang and natural tone), and direct Go struct mapping via the official Google Gen AI SDK.
- **Alternatives Considered**:
  - *Third-party OpenAI or Anthropic SDKs*: Rejected to adhere to the project's Google ecosystem alignment.
  - *Generic REST client calling Gemini API directly*: Rejected because `google.golang.org/genai` provides typed tool definitions, history management, and clean SDK upgrades.

---

## Decision 4: 1:N Customer Addresses & Verified Phone Injection

- **Decision**: Create a dedicated `CustomerAddress` entity associated with `Customer` by `customer_phone` (`wa_id`). Never request or accept the customer's phone number as text input in chat; always inject `wa_id` securely from the WhatsApp webhook payload.
- **Rationale**: WhatsApp provides cryptographically verified sender IDs (`wa_id`). Allowing users to type their phone number invites typos, formatting mismatches (country codes, prefixes), and security impersonation. The 1:N address model enables recurring customers to maintain aliases like "Casa", "Trabajo", and "Depto".
- **Alternatives Considered**:
  - *Storing a single text address column in `Customer`*: Rejected because it prevents recurring customers from choosing between home and work without re-entering the entire address.
  - *Prompting user: "Ingresá tu teléfono"*: Rejected because it harms user experience and degrades conversion rates.

---

## Decision 5: Hexagonal Architecture (Ports & Adapters) for Uber Direct and Mercado Pago

- **Decision**: Define domain output interfaces `DeliveryProvider` (`QuoteDelivery`, `CreateDelivery`) and `PaymentProvider` (`GeneratePaymentLink`, `VerifyPaymentStatus`) in `ports/output/`. Implement active `MockDeliveryProvider` and `MockPaymentProvider` for development/testing, and standalone HTTP adapter skeletons for `UberDirectProvider` and `MercadoPagoProvider`.
- **Rationale**: Enables end-to-end testing, local development, and CI/CD validation without requiring real Uber Direct credentials or Mercado Pago test accounts. When live API keys are provided in `.env`, the adapter implementation can be switched with zero changes to domain use cases.
- **Alternatives Considered**:
  - *Direct coupling with third-party HTTP endpoints inside use cases*: Rejected because it violates Clean Architecture principles, hinders unit testing, and couples business logic to external API schemas.

---

## Decision 6: Cash Register Integrity with Unpaid Orders

- **Decision**: The order creation tool persists orders with `status = PENDING_PAYMENT`. `CashRegister` metrics and cash counts remain untouched until the order transitions to `PAID` or `COMPLETED`.
- **Rationale**: Customers may abandon the checkout process after receiving the payment link. Distorting the active cash register with pending or unpaid orders would lead to discrepancies during daily and monthly financial closings (`MonthlyClosing`).
- **Alternatives Considered**:
  - *Only saving the order upon payment webhook*: Rejected because the payment preference requires an existing `order_id` as an external reference (`external_reference`), and customers need their order ID upfront.
