# Phase 1 Data Model: Roli - Prompt-as-Code DSL & Session Context

**Feature**: `003-roli-prompt-dsl`  
**Date**: 2026-10-16  

---

## 1. Context Structs & In-Memory Entities

### `AgentSessionContext`
Represents the dynamic runtime state of a customer conversation before a turn is processed by the AI model.

```go
type AgentSessionContext struct {
    // CurrentTime is the human-readable date and time in Córdoba, Argentina (e.g. "Sábado 17:15")
    CurrentTime string

    // CustomerName is the WhatsApp profile display name or fallback "Cliente"
    CustomerName string

    // CustomerPhone is the international WhatsApp phone number (wa_id, e.g. "5491198765432")
    CustomerPhone string

    // IsCashRegisterOpen indicates whether the bakery's cash register / kitchen is active
    IsCashRegisterOpen bool

    // HasSavedAddresses indicates whether the customer has at least one saved delivery address
    HasSavedAddresses bool

    // PendingOrderID holds the ID of any active pending order awaiting payment (or empty string)
    PendingOrderID string
}
```

#### Field Constraints & Validation Rules:
- `CurrentTime`: Must not be empty. Computed from `time.Now().In(argentinaLocation)` and formatted in Spanish (e.g. "Lunes 18:30").
- `CustomerName`: If empty or whitespace, defaults to `"Cliente"`.
- `CustomerPhone`: Mandatory, sanitized string matching `^\d{10,15}$`.
- `IsCashRegisterOpen`: Evaluated from `cashRepo.GetActive() == nil` check.
- `HasSavedAddresses`: Evaluated from `len(addressRepo.FindByPhone(customerPhone)) > 0`.
- `PendingOrderID`: Evaluated from `orderRepo.FindPendingByCustomer(customer.ID)` or empty string `" "` if none.

---

### `AgentPromptBuilder`
Component responsible for compiling, caching, and rendering the DSL prompt template.

```go
type AgentPromptBuilder struct {
    tmpl *template.Template
}

func NewAgentPromptBuilder() (*AgentPromptBuilder, error)
func (b *AgentPromptBuilder) BuildSystemInstruction(ctx AgentSessionContext) (string, error)
```

#### Invariants:
- The template `cinnamon_sales_agent.tmpl` is parsed once during `NewAgentPromptBuilder()` initialization.
- Thread-safe: `template.Template.Execute` is concurrent-safe when reading immutable parsed trees into independent `bytes.Buffer` instances.

---

## 2. Tool Security Policy

```go
type ToolSecurityPolicy struct {
    AllowedFunctions []string
    Mode             string // AUTO or ANY
}

func ResolveToolSecurityPolicy(isRegisterOpen bool) ToolSecurityPolicy
```

#### State Transition Matrix:

| Condition | `IsCashRegisterOpen` | Allowed Functions |
|---|---|---|
| Register Open | `true` | `["consultar_catalogo", "obtener_direcciones_cliente", "guardar_direccion_cliente", "cotizar_envio", "crear_pedido_con_link_pago", "verificar_estado_pago"]` |
| Register Closed | `false` | `["consultar_catalogo", "verificar_estado_pago"]` |

---

## 3. DSL Template File Structure (`cinnamon_sales_agent.tmpl`)

The DSL document is structured using explicit XML blocks:
- `<agent_profile>`: Name ("Roli"), business ("CaneRolls"), and authentic Córdoba tone guidelines.
- `<live_session_context>`: Dynamic values injected from `AgentSessionContext`.
- `<strict_guardrails>`: 6 non-negotiable business rules (no "topping", zero hallucination, confirmation before creating orders, etc.).
- `<conversation_state_machine>`: 5-step guided flow (Selection -> Delivery -> Address/Quote -> Pre-closure Summary -> Payment & Verification).
- `<examples_of_interaction>`: WhatsApp conversation snippets with CaneRollss official copy (welcome, receipt with Alias `Caneroll.mp`, heating tips).
