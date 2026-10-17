# Implementation Plan: Roli - Prompt-as-Code DSL & Deterministic Function Security

**Branch**: `003-roli-prompt-dsl` | **Date**: 2026-10-16 | **Spec**: [specs/003-roli-prompt-dsl/spec.md](file:///c:/Users/mdcac/canerollss/canerollss-backend/specs/003-roli-prompt-dsl/spec.md)

**Input**: Feature specification from `specs/003-roli-prompt-dsl/spec.md`

## Summary

Implement the conversational AI agent "Roli" for CaneRollss using the **Prompt-as-Code** pattern with an embedded XML DSL template (`cinnamon_sales_agent.tmpl`) via Go's `//go:embed`. Before every conversation turn, an `AgentPromptBuilder` compiles dynamic session variables (`CurrentTime`, `CustomerName`, `CustomerPhone`, `IsCashRegisterOpen`, `HasSavedAddresses`, `PendingOrderID`) into the system instructions. Furthermore, deterministic function-calling security is enforced at the GenAI protocol level via `ToolConfig.FunctionCallingConfig.AllowedFunctionNames`, completely locking out `crear_pedido_con_link_pago` whenever the cash register is closed. The prompt is imbued with official CaneRollss copy (catalog welcome, receipt format with Alias `Caneroll.mp`, reheating advice, and Córdoba delivery zones).

## Technical Context

**Language/Version**: Go 1.26  
**Primary Dependencies**: `google.golang.org/genai v1.75.0`, `github.com/gofiber/fiber/v2`, `text/template`, `embed`  
**Storage**: MySQL (production) / SQLite (isolated testing) via GORM  
**Testing**: `go test` (unit tests and live GenAI E2E tests)  
**Target Platform**: Linux (Docker) / Windows local dev  
**Project Type**: REST API / Webhook Backend with Hexagonal Architecture  
**Performance Goals**: < 2ms template rendering overhead, zero heap allocation for template parsing  
**Constraints**: Single self-contained Go binary via `//go:embed`; protocol-level function blocking via `ToolConfig`  
**Scale/Scope**: Handles concurrent WhatsApp dialogues partitioned by phone number  

## Constitution Check

- Clean Hexagonal Architecture: Prompt template and builder live in `core/usecase` domain layer.
- Thread Safety: Template parsed once at startup; executed against per-turn buffers without global state mutation.
- Zero Database Leakage: Tool execution strictly filters customer data by `wa_id`.

## Project Structure

### Documentation (this feature)

```text
specs/003-roli-prompt-dsl/
├── spec.md              # Requirements and user stories
├── plan.md              # This implementation plan
├── research.md          # Technical research & decisions
├── data-model.md        # Session context, builder and policy models
├── quickstart.md        # Validation guide
└── contracts/
    ├── agent-context-schema.json
    └── prompt-dsl-contract.md
```

### Source Code Layout

```text
core/
└── usecase/
    ├── prompts/
    │   └── cinnamon_sales_agent.tmpl   # Embedded XML DSL prompt template
    ├── agent_prompt_builder.go         # //go:embed and template renderer
    ├── agent_prompt_builder_test.go    # Unit tests for DSL rendering
    └── whatsapp_agent_usecase.go       # Injects AgentSessionContext & ToolConfig

adapters/
└── input/http/handlers/
    └── whatsapp_handler.go             # Resolves session context before invoking usecase

tests/
└── gemini_live_test.go                 # Live multi-turn integration test with Gemini
```

## Complexity Tracking

No violations. Implementation uses Go standard library `embed` and `text/template` alongside the existing official Google GenAI Go SDK.
