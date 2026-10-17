# Implementation Plan: Automatic Transfer Reconciliation (Mercado Pago Alias/CVU)

**Branch**: `004-transfer-reconciliation` | **Date**: 2026-10-16 | **Spec**: [specs/004-transfer-reconciliation/spec.md](spec.md)

**Input**: Feature specification from `/specs/004-transfer-reconciliation/spec.md`

## Summary

Implement the automatic bank/CVU transfer reconciliation module for payments received on the store's Mercado Pago Alias (`Caneroll.mp`, account holder: `Ivonne Abreu`). The solution adheres strictly to Clean & Hexagonal Architecture by defining pure domain entities (`PaymentTransaction`, `ReconciliationResult`), ports (`output.PaymentProvider` and `input.PaymentReconciliationUseCasePort`), a dedicated business usecase (`PaymentReconciliationUseCase`), and infrastructure adapters (HTTP `MercadoPagoProvider` and in-memory `MockPaymentProvider`). The module integrates directly into the WhatsApp Agent's Prompt-as-Code DSL and provides an updated webhook handler for asynchronous notifications.

## Technical Context

**Language/Version**: Go 1.22+  
**Primary Dependencies**: `google.golang.org/genai` (Agent DSL), `github.com/gofiber/fiber/v2` (Webhooks/HTTP), `gorm.io/gorm` (Database ORM)  
**Storage**: SQLite (`DB_DRIVER=sqlite` for local dev/testing) and MySQL (production Docker) via GORM  
**Testing**: Go standard testing package (`testing`, `httptest`), isolated SQLite databases per test suite  
**Target Platform**: Linux / Windows Go backend server  
**Project Type**: Web service / REST API + WhatsApp Webhook + Conversational AI Agent  
**Performance Goals**: On-demand payment verification response in < 3 seconds; 0 false positive reconciliations  
**Constraints**: Zero double-spending; pure domain layer free of vendor JSON tags; multi-model AI fallback resilience  
**Scale/Scope**: ~100 daily orders, instantaneous multi-turn conversational checking  

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Layer Decoupling**: Domain (`core/domain`) has NO dependencies on `adapters/` or external SDKs. (PASS)
- **Dependency Inversion**: Use cases depend only on interfaces in `ports/output` or other use cases. (PASS)
- **Test-First / Testability**: All business logic testable with `MockPaymentProvider` in memory without internet. (PASS)
- **Law of Demeter**: No use case directly accesses another use case's private struct members. (PASS)

## Project Structure

### Documentation (this feature)

```text
specs/004-transfer-reconciliation/
├── spec.md              # Feature specification
├── plan.md              # Implementation plan (this file)
├── research.md          # Phase 0 decisions & research
├── data-model.md        # Phase 1 domain entities & state machines
├── quickstart.md        # Phase 1 validation scenarios
└── contracts/           # Phase 1 interface contracts
    ├── payment_provider_port.md
    ├── mercadopago_api_contract.md
    └── whatsapp_tool_contract.md
```

### Source Code (repository root)

```text
core/
├── domain/
│   ├── order.go                       # Updated PaymentMethod & PaidAt fields
│   └── payment_transaction.go         # Pure PaymentTransaction & ReconciliationResult structs
└── usecase/
    ├── payment_reconciliation_usecase.go      # Core reconciliation business logic & anti-double-spending
    ├── payment_reconciliation_usecase_test.go # Comprehensive unit test suite
    ├── whatsapp_agent_usecase.go              # Agent tool hookup & receipt formatting
    └── prompts/
        └── cinnamon_sales_agent.tmpl          # Updated Alias Caneroll.mp prompt & verification instructions

ports/
├── output/
│   └── payment_provider.go            # SearchApprovedTransfers & GetTransferByID contracts
└── input/
    └── payment_reconciliation_port.go # Input port interface for driving adapters

adapters/
├── output/
│   └── payment/
│       ├── mercadopago_provider.go    # HTTP client for /v1/payments/search and /v1/payments/{id}
│       └── mock_payment_provider.go   # Controllable in-memory transfer ledger
└── input/
    └── http/
        ├── handlers/
        │   └── mercadopago_handler.go # Webhook adapter delegating to reconciliation port
        └── routes/
            └── webhook_routes.go

tests/
├── e2e_test.go                        # Updated E2E flow testing alias transfer confirmation
└── reconciliation_test.go             # Integration tests for reconciliation scenarios
```

## Architecture & Design Decisions

1. **Pure Domain Isolation**:
   - `domain.PaymentTransaction` represents the accredited transfer, completely isolated from Mercado Pago's JSON response structures.
   - `domain.ReconciliationResult` provides a deterministic state (`MATCHED`, `NOT_FOUND_YET`, `AMBIGUOUS_MULTIPLE_MATCHES`, `AMOUNT_MISMATCH`, `ALREADY_USED`).
2. **Double-Spending Prevention**:
   - Every matched order persists `external_payment_id`.
   - The reconciliation engine queries `orderRepo` to ensure no active order has previously claimed that transaction ID.
3. **Disambiguation Heuristics**:
   - If multiple candidates exist with the same amount in the time window, the system pauses auto-approval and asks the customer for the operation ID, payer name, or DNI.
4. **Agent Integration**:
   - Replaces payment link output in prompt DSL with the official receipt format: Total, Alias `Caneroll.mp`, Titular `Ivonne Abreu`.
   - Enables Roli to call `verificar_estado_pago` to confirm transfers on-demand.
