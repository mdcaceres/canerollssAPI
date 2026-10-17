# Tasks: Roli - Prompt-as-Code DSL & Deterministic Function Security

**Feature**: `003-roli-prompt-dsl`  
**Spec**: [specs/003-roli-prompt-dsl/spec.md](file:///c:/Users/mdcac/canerollss/canerollss-backend/specs/003-roli-prompt-dsl/spec.md)  
**Plan**: [specs/003-roli-prompt-dsl/plan.md](file:///c:/Users/mdcac/canerollss/canerollss-backend/specs/003-roli-prompt-dsl/plan.md)  

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, directory structure, and foundational prompt types

- [X] T001 Create prompt template directory at `core/usecase/prompts/` and initialize template file `core/usecase/prompts/cinnamon_sales_agent.tmpl`
- [X] T002 [P] Define `AgentSessionContext` struct with fields `CurrentTime`, `CustomerName`, `CustomerPhone`, `IsCashRegisterOpen`, `HasSavedAddresses`, and `PendingOrderID` in `core/usecase/agent_prompt_builder.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core template compilation and embedding infrastructure that MUST be complete before user stories

**⚠️ CRITICAL**: All user story tasks depend on this phase

- [X] T003 Implement `AgentPromptBuilder` in `core/usecase/agent_prompt_builder.go` using `//go:embed prompts/cinnamon_sales_agent.tmpl` and Go `text/template` compilation
- [X] T004 [P] Implement unit tests for `AgentPromptBuilder` in `core/usecase/agent_prompt_builder_test.go` verifying template compilation and execution with mock session context

**Checkpoint**: Template builder operational and covered by unit tests

---

## Phase 3: User Story 1 - Declarative Prompt-as-Code Rendering with Live Session Context (Priority: P1) 🎯 MVP

**Goal**: Roli dynamically renders live session variables (time in Córdoba, customer name/wa_id, register status, address existence) into system instructions on every turn.

**Independent Test**: Provide mock session contexts to `AgentPromptBuilder.BuildSystemInstruction` and verify XML tag substitutions and fallback handling.

### Tests for User Story 1
- [X] T005 [P] [US1] Write unit tests in `core/usecase/agent_prompt_builder_test.go` verifying fallback of empty `CustomerName` to `"Cliente"` and proper tag injection in `<live_session_context>`

### Implementation for User Story 1
- [X] T006 [US1] Write complete XML DSL prompt in `core/usecase/prompts/cinnamon_sales_agent.tmpl` with blocks `<agent_profile>`, `<live_session_context>`, `<strict_guardrails>`, `<conversation_state_machine>`, and `<examples_of_interaction>`
- [X] T007 [US1] Update `WhatsAppAgentUseCase` in `core/usecase/whatsapp_agent_usecase.go` to accept `AgentPromptBuilder` and invoke `BuildSystemInstruction` on every message turn instead of legacy static prompt
- [X] T008 [US1] Update `adapters/input/http/handlers/whatsapp_handler.go` to assemble `AgentSessionContext` (computing Argentina local time, address count via `addressUseCase`, cash register status via `cashRepo`, and pending order via `orderRepo`) before calling `agentUseCase.ProcessMessage`

**Checkpoint**: User Story 1 is fully functional and independently testable; prompts reflect live customer and operational state.

---

## Phase 4: User Story 2 - Deterministic Function Gating via ToolConfig (Priority: P2)

**Goal**: Block the LLM from calling `crear_pedido_con_link_pago` at the protocol level via Google GenAI `ToolConfig.FunctionCallingConfig.AllowedFunctionNames` when `IsCashRegisterOpen` is false.

**Independent Test**: Call `ProcessMessage` with `IsCashRegisterOpen = false` and verify that `AllowedFunctionNames` only contains `["consultar_catalogo", "verificar_estado_pago"]`.

### Tests for User Story 2
- [X] T009 [P] [US2] Write unit tests in `core/usecase/whatsapp_agent_usecase_test.go` asserting that `ToolConfig` restricts `AllowedFunctionNames` to `consultar_catalogo` and `verificar_estado_pago` when register is closed, and includes all 6 tools when open

### Implementation for User Story 2
- [X] T010 [US2] Implement `resolveToolConfig(isRegisterOpen bool) *genai.ToolConfig` in `core/usecase/whatsapp_agent_usecase.go` enforcing function gating based on register state
- [X] T011 [US2] Attach dynamic `ToolConfig` to `genai.GenerateContentConfig` in `core/usecase/whatsapp_agent_usecase.go` for all model invocations

**Checkpoint**: User Stories 1 AND 2 are functional; order creation is safely gated when kitchen/register is closed.

---

## Phase 5: User Story 3 - CaneRollss Official Communication Identity & Copy Templates (Priority: P3)

**Goal**: Align Roli's responses with authentic CaneRollss templates: menu welcome, visual receipt with Alias `Caneroll.mp` / `Ivonne Abreu`, delivery zones (Córdoba Capital), and microwave warming tips.

**Independent Test**: Execute multi-turn conversation and assert presence of brand copies, receipt format, and Instagram tagging recommendations.

### Implementation for User Story 3
- [X] T012 [US3] Add official CaneRollss welcome copy (`"¡Te compartimos nuestro menú! 🤎🍥"`), flavor list format, and limited batch disclaimer into `core/usecase/prompts/cinnamon_sales_agent.tmpl`
- [X] T013 [US3] Add official visual receipt layout (Pedido, Envío, Total, Alias `Caneroll.mp`, Titular `Ivonne Abreu`) and delivery coverage rules (Córdoba Capital, entregas viernes a domingo) into `core/usecase/prompts/cinnamon_sales_agent.tmpl`
- [X] T014 [US3] Add post-sale microwave warming instructions (5-10 seconds in microwave, oven-safe container) and Instagram tagging copy into `core/usecase/prompts/cinnamon_sales_agent.tmpl`
- [X] T015 [US3] Update offline fallback handler in `core/usecase/whatsapp_agent_usecase.go` to match the exact CaneRollss receipt and catalog format

**Checkpoint**: All three user stories functional with full CaneRollss brand identity.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: End-to-end live verification and test suite validation

- [X] T016 [P] Update `tests/gemini_live_test.go` to execute a 3-turn live verification with Roli (Catalog inquiry -> Delivery quote with address -> Order confirmation with receipt and payment link)
- [X] T017 Run complete test suite (`go test ./...`) across all packages to guarantee zero regressions
- [X] T018 Execute validation scenarios from `specs/003-roli-prompt-dsl/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies
- **Phase 1 (Setup)**: Can start immediately.
- **Phase 2 (Foundational)**: Depends on Phase 1 (T001, T002). Blocks all User Stories.
- **Phase 3 (User Story 1 - P1)**: Depends on Phase 2. Delivers core Prompt-as-Code MVP.
- **Phase 4 (User Story 2 - P2)**: Depends on Phase 2. Can run parallel to or after US1.
- **Phase 5 (User Story 3 - P3)**: Depends on US1 (enhances DSL template with official copies).
- **Phase 6 (Polish)**: Depends on completion of all user stories.

### Parallel Execution Opportunities
- `T002` (Setup context struct) can run in parallel with `T001`.
- `T004` (Unit test for builder) can run in parallel with template writing.
- `T005` (US1 test) and `T009` (US2 test) can be developed in parallel.
- `T016` (Live test update) can be developed independently of handler updates.

---

## Implementation Strategy (MVP First)

1. Complete Phase 1 & Phase 2 (Setup & Foundational Builder).
2. Complete Phase 3 (User Story 1): Embed DSL template and verify dynamic rendering with `AgentSessionContext`.
3. **VALIDATE MVP**: Confirm that Roli greets with dynamic time and customer name.
4. Complete Phase 4 (User Story 2): Enforce protocol-level tool restriction when register is closed.
5. Complete Phase 5 (User Story 3): Inject official CaneRollss copies, receipt format, and warming advice.
6. Complete Phase 6: Run full test suite and live validation.
