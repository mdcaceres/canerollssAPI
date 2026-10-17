# Implementation Plan: WhatsApp AI-Powered Order Automation & Transactional Flow

**Branch**: `002-whatsapp-ai-ordering` | **Date**: 2026-10-16 | **Spec**: [specs/002-whatsapp-ai-ordering/spec.md](file:///c:/Users/mdcac/canerollss/canerollss-backend/specs/002-whatsapp-ai-ordering/spec.md)

**Input**: Feature specification from `specs/002-whatsapp-ai-ordering/spec.md`

---

## Summary

Automate cinnamon roll order-taking via WhatsApp Cloud API using Google Gemini 2.5 Flash (`google.golang.org/genai`). The system handles multi-message debouncing (4-second buffer), catalog queries using Argentine flavor terminology ("sabores"/"roles", strictly avoiding "toppings"), 1:N customer delivery address management, pre-order delivery fee quotation via a `DeliveryProvider` logistics port (Uber Direct API / Mock), and order creation in `PENDING_PAYMENT` status with digital payment preference generation via a `PaymentProvider` port (Mercado Pago API / Mock). Customers never type their phone number; it is automatically bound from the WhatsApp session (`wa_id`).

---

## Technical Context

**Language/Version**: Go 1.22+ (go.mod configured for Go 1.26)

**Primary Dependencies**:
- Web Framework: `github.com/gofiber/fiber/v2`
- ORM: `gorm.io/gorm` with MySQL driver `gorm.io/driver/mysql`
- AI SDK: `google.golang.org/genai` (model: `gemini-2.5-flash`)
- Environment & Utils: `github.com/joho/godotenv`, `github.com/google/uuid`

**Storage**: MySQL with GORM auto-migrations (`customer_address`, `order`, `order_item`, `customer`, `cash_register`, `topping`)

**Testing**: Go standard testing (`testing` package) with Mock adapters for unit and integration checks

**Target Platform**: Linux server / Containerized cloud deployment / Local Windows dev environment

**Project Type**: REST Web Service with WhatsApp Webhook and AI Function Calling orchestration

**Performance Goals**:
- Webhook response latency < 2 seconds (immediate HTTP 200)
- Debounce window: 4 seconds for message batching
- AI agent turn response latency: 2-4 seconds end-to-end

**Constraints**:
- Must not fabricate prices or flavors (must strictly invoke live catalog tools)
- Must not ask the user to type their phone number (must extract and persist `wa_id`)
- Must not distort `CashRegister` totals with unpaid orders (must keep `PENDING_PAYMENT` segregated until paid)
- Must avoid the word "topping" in customer-facing text

**Scale/Scope**: ~50 concurrent WhatsApp conversations per instance, memory footprint < 150MB

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / Gate | Status | Evidence / Notes |
|------------------|--------|------------------|
| **I. Hexagonal Architecture / Clean Separation** | PASSED | Domain logic (`core/usecase`), domain models (`core/domain`), interfaces (`ports/output`), and infrastructure (`adapters/output`, `adapters/input`) remain strictly decoupled. |
| **II. Reusability of Existing Business Logic** | PASSED | Reuses `OrderUseCase.CreateOrder`, `ToppingRepository`, `CustomerRepository`, and `CashRegisterRepository` without duplicating core logic. |
| **III. Testability & Decoupling** | PASSED | External logistics (Uber Direct) and payment gateways (Mercado Pago) are abstracted behind `DeliveryProvider` and `PaymentProvider` with fully functional Mock adapters. |
| **IV. Data Integrity & Security** | PASSED | Customer identity is cryptographically tied to WhatsApp `wa_id`; no manual phone entry or identity spoofing is possible. |
| **V. Zero Fabrication Rule** | PASSED | Agent is bound to live Function Declarations and strict system prompts preventing hallucinated products or prices. |

---

## Project Structure

### Documentation (this feature)

```text
specs/002-whatsapp-ai-ordering/
├── plan.md              # This file (/speckit-plan output)
├── research.md          # Phase 0 output (/speckit-plan output)
├── data-model.md        # Phase 1 output (/speckit-plan output)
├── quickstart.md        # Phase 1 output (/speckit-plan output)
├── contracts/           # Phase 1 output (/speckit-plan output)
│   ├── gemini-tools.json    # FunctionDeclarations for gemini-2.5-flash
│   └── external-ports.md    # Go interface contracts for Delivery & Payment
└── tasks.md             # Phase 2 output (/speckit-tasks command)
```

### Source Code (repository root)

```text
canerollss-backend/
├── core/
│   ├── domain/
│   │   ├── customer_address.go      # CustomerAddress entity (1:N with customer)
│   │   ├── order.go                 # Order entity extended with status, delivery_type, payment_link, etc.
│   │   ├── customer.go              # Customer entity
│   │   ├── topping.go               # Topping (flavors) entity
│   │   └── cash_register.go         # Cash register entity
│   └── usecase/
│       ├── address_usecase.go       # Address management use case
│       ├── order_usecase.go         # Order lifecycle & creation logic
│       └── whatsapp_agent_usecase.go# Gemini Flash agent runner, tool executor, conversation memory
├── ports/
│   └── output/
│       ├── customer_address_repository.go # Repository port for CustomerAddress
│       ├── delivery_provider.go           # Logistics port (QuoteDelivery, CreateDelivery)
│       ├── payment_provider.go            # Payment port (GeneratePaymentLink, VerifyPaymentStatus)
│       ├── order_repository.go            # Order repository port
│       ├── customer_repository.go         # Customer repository port
│       └── topping_repository.go          # Topping repository port
├── adapters/
│   ├── output/
│   │   ├── datasource/
│   │   │   └── connection.go              # AutoMigrate for CustomerAddress and Order
│   │   ├── repository/
│   │   │   └── customer_address_repository.go # GORM implementation for CustomerAddress
│   │   ├── delivery/
│   │   │   ├── mock_delivery_provider.go      # Active Mock logistics provider
│   │   │   └── uber_direct_provider.go        # HTTP skeleton for Uber Direct API
│   │   ├── payment/
│   │   │   ├── mock_payment_provider.go       # Active Mock payment provider
│   │   │   └── mercadopago_provider.go        # HTTP skeleton for Mercado Pago API
│   │   └── whatsapp/
│   │       └── whatsapp_client.go             # Meta Graph API outbound message sender
│   └── input/
│       └── http/
│           ├── handlers/
│           │   ├── whatsapp_handler.go        # GET (verification) and POST (webhook) for WhatsApp
│           │   └── mercadopago_handler.go     # POST webhook for Mercado Pago payment updates
│           ├── routes/
│           │   └── webhook_routes.go          # Route registration in Fiber
│           └── session/
│               └── session_manager.go         # 4-second Debounce timer and in-memory session buffer
└── app/
    └── app.go                                 # Dependency injection & service initialization
```

---

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| In-memory `SessionManager` with mutex | WhatsApp users break thoughts into bursts of 3-4 messages within seconds. | Processing each message individually produces fragmented AI responses, doubles token costs, and creates broken UX. |
| Separate `CustomerAddress` table | Customers have multiple recurring locations (home, work, partner's place). | Storing a single address in `Customer` forces re-typing on every different delivery. |
| External Ports (`DeliveryProvider`, `PaymentProvider`) with Mocks | Allows development, unit testing, and CI verification without requiring live Uber Direct or Mercado Pago production keys. | Direct HTTP calls inside use case violates Hexagonal Architecture and breaks testability. |
