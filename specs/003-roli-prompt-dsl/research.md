# Phase 0 Research: Roli - Prompt-as-Code DSL & ToolConfig Security

**Feature**: `003-roli-prompt-dsl`  
**Date**: 2026-10-16  

---

## 1. Technical Decisions & Research

### Decision 1: Prompt-as-Code Architecture via Go `embed` and `text/template`
- **Context**: The existing `WhatsAppAgentUseCase` had hardcoded prompt strings directly in Go source code (`getSystemInstruction()`), which mixed business persona, guardrails, and dynamic rules with application logic, making it difficult to maintain and audit.
- **Decision**: Store the declarative prompt template in `core/usecase/prompts/cinnamon_sales_agent.tmpl` and embed it at compile-time using Go's `//go:embed`. Render it dynamically per turn using standard library `text/template`.
- **Rationale**:
  - Zero external dependencies (`embed` and `text/template` are standard Go libraries).
  - Fast execution (<1ms) since the template is parsed once at initialization and executed with a lightweight `bytes.Buffer`.
  - Self-contained binary (no external disk paths needed in production Docker or standalone binaries).
- **Alternatives Considered**:
  - *Reading from filesystem at runtime*: Requires file path resolution and risks runtime failure if file is missing in container.
  - *Hardcoded multiline Go strings*: Lacks syntax highlighting and clean separation of persona/guardrails.

---

### Decision 2: Injected Dynamic Session Context (`AgentSessionContext`)
- **Context**: The agent needs real-time awareness of:
  - Current day and hour in Córdoba, Argentina (UTC-3).
  - Customer name and WhatsApp phone (`wa_id`).
  - Cash register status (`IsCashRegisterOpen`).
  - Whether customer already has saved delivery addresses (`HasSavedAddresses`).
  - Whether a pending order exists awaiting payment (`PendingOrderID`).
- **Decision**: Define an explicit struct `AgentSessionContext` in `core/usecase/agent_prompt_builder.go`. Prior to calling the GenAI model, `WhatsAppAgentUseCase` gathers these values and executes the template to produce the active `system_instruction`.
- **Rationale**:
  - LLMs hallucinate less when grounded with explicit factual tags (`<live_session_context>`).
  - The model does not need to execute unnecessary tool calls just to check if addresses exist or what time it is.
- **Alternatives Considered**:
  - *Having tools for time and register status*: Causes redundant tool roundtrips, increasing latency and API cost.

---

### Decision 3: Deterministic Function Gating with GenAI `ToolConfig`
- **Context**: When the bakery's cash register or oven is closed, customers must not be able to create orders or receive payment links. Relying solely on the LLM's prompt following is vulnerable to jailbreaks or prompt injection.
- **Decision**: Configure Google GenAI SDK's `ToolConfig` dynamically on each turn:
  ```go
  if !sessionCtx.IsCashRegisterOpen {
      config.ToolConfig = &genai.ToolConfig{
          FunctionCallingConfig: &genai.FunctionCallingConfig{
              Mode: genai.FunctionCallingModeAny, // or Auto with allowed list
              AllowedFunctionNames: []string{
                  "consultar_catalogo",
                  "verificar_estado_pago",
              },
          },
      }
  }
  ```
- **Rationale**:
  - Enforced at the protocol/API level by Google's inference engine: the model cannot physically output a tool call for `crear_pedido_con_link_pago`.
  - Allows catalog browsing and payment verification for previously issued orders even when new order intake is paused.
- **Alternatives Considered**:
  - *Returning an error from the Go tool execution*: The model still attempts to call the tool and burns an API roundtrip; protocol-level gating prevents the attempt entirely.

---

### Decision 4: CaneRollss Identity & WhatsApp Copy Integration
- **Context**: The bakery has established, high-converting WhatsApp message styles (warm tone, emojis, limited batch warning, visual receipt with Alias `Caneroll.mp`, and reheating tips).
- **Decision**: Embed these exact templates and copy patterns into the DSL `<examples_of_interaction>` and `<strict_guardrails>` sections.
- **Rationale**:
  - Standardizes brand voice: warm Córdoba rioplatense with natural voseo (*"contame"*, *"te armo"*, *"fijate"*, *"che"*).
  - Prevents the technical term "topping" from ever reaching customers (enforcing "sabores", "roles" or "variedades").
  - Seamlessly guides customers through the 5-step conversation state machine.
