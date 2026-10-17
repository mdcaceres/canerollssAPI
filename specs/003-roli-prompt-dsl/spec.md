# Feature Specification: Roli - Prompt-as-Code DSL & Deterministic Function Security

**Feature Branch**: `003-roli-prompt-dsl`  
**Created**: 2026-10-16  
**Status**: Draft  
**Input**: User description: "Fase del Agente IA Conversacional (WhatsAppAgentUseCase) para nuestro backend en Go, aplicando el patrón de Prompt-as-Code (DSL Declarativo con Plantillas) inyectado mediante //go:embed y configurado con seguridad determinística (ToolConfig / AllowedFunctionNames). La agente se llamará Roli y estará hiper-especializada en la venta y toma de pedidos de CaneRolls en Córdoba, Argentina..."

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Declarative Prompt-as-Code Rendering with Live Session Context (Priority: P1)

As a CaneRolls customer communicating with the bakery on WhatsApp, I want the AI sales agent ("Roli") to converse with an authentic, friendly Córdoba (Argentine) tone, know my name, understand whether I already have delivery addresses saved, and acknowledge the current time, so that the interaction feels personalized, natural, and locally grounded.

**Why this priority**:
Prompt-as-Code decouples the agent's persona and business instructions from procedural Go code. Rendering live contextual variables (customer name, WhatsApp ID, current time, address presence, pending order status) on every turn prevents hallucinations and grounds the model before it generates any token.

**Independent Test**:
Can be tested independently by providing mock `AgentSessionContext` structs with different values (e.g., existing addresses vs. first-time customer) and verifying that the rendered system instruction contains the substituted XML tags.

**Acceptance Scenarios**:
1. **Given** a customer named "Sofía" messaging on Saturday at 17:15, **When** the agent processes her message, **Then** the rendered system instruction contains `<customer_name>Sofía</customer_name>`, `<current_time>Sábado 17:15</current_time>`, and `<agent_profile><name>Roli</name>...`.
2. **Given** a customer with no prior saved addresses (`HasSavedAddresses: false`), **When** asking for delivery, **Then** Roli follows `<conversation_state_machine>` Step 3 to ask for street, number, and apartment details before quoting.

---

### User Story 2 - Deterministic Function Gating via ToolConfig (Priority: P2)

As a bakery business owner, I want the system to deterministically disable order creation when the cash register is closed, so that customers cannot create orders or receive payment links when the kitchen/oven is not taking orders, regardless of how the customer prompts the LLM.

**Why this priority**:
Relying solely on LLM prompt adherence to prevent order creation when the cash register is closed is insecure (prompt injection / jailbreak risk). Enforcing `AllowedFunctionNames` in the model configuration guarantees at the protocol level that the model *cannot* invoke `crear_pedido_con_link_pago`.

**Independent Test**:
Can be tested independently by setting `IsCashRegisterOpen: false` and asserting that `ToolConfig.FunctionCallingConfig.AllowedFunctionNames` excludes `crear_pedido_con_link_pago`, verifying that even if the customer writes "Quiero confirmar mi pedido ya", the order creation tool cannot be executed.

**Acceptance Scenarios**:
1. **Given** `IsCashRegisterOpen` is `false`, **When** the model config is constructed, **Then** `AllowedFunctionNames` contains only `consultar_catalogo` and `verificar_estado_pago`.
2. **Given** `IsCashRegisterOpen` is `false` and a customer attempts to place an order, **Then** Roli informs the customer that the oven/cash register is closed and invites them to check back for the next baking batch.
3. **Given** `IsCashRegisterOpen` is `true`, **When** the model config is constructed, **Then** `AllowedFunctionNames` includes all tools: `consultar_catalogo`, `obtener_direcciones_cliente`, `guardar_direccion_cliente`, `cotizar_envio`, `crear_pedido_con_link_pago`, and `verificar_estado_pago`.

---

### User Story 3 - CaneRollss Official Communication Identity & Copy Templates (Priority: P3)

As a CaneRolls customer, I want to receive menu listings, order breakdowns, and post-delivery warming instructions in the exact visual structure, emoji style, and official bakery tone (including bank transfer Alias and Instagram tags), so that the brand experience is cohesive and professional.

**Why this priority**:
Replicating real-world bakery communication templates (warm welcome, cupos limitados por tanda, visual receipt breakdown with Alias `Caneroll.mp`, and microwave warming tips) builds brand trust and customer loyalty.

**Independent Test**:
Can be tested independently by triggering menu inquiries, order summaries, and delivery notifications, and verifying that the outputs adhere to the formatting guidelines in `<examples_of_interaction>`.

**Acceptance Scenarios**:
1. **Given** a customer asking for flavors, **When** Roli responds, **Then** the message uses CaneRollss greeting copy ("¡Te compartimos nuestro menú! 🤎🍥"), mentions handmade/freshly baked rolls, formats varieties with bullet points, and notes limited batch spots.
2. **Given** a customer ready to confirm an order, **When** Roli presents the pre-checkout summary, **Then** the message formats the receipt as:
   ```text
   ✨ ¡Gracias por tu pedido! ✨
   🧾 Pedido: [Detalle]
   🛵 Envío: [Costo o Retiro]
   Total: $[Monto]
   📲 Alias: Caneroll.mp  
   👤 A nombre de: Ivonne Abreu
   ```
3. **Given** delivery dispatch or delivery questions, **When** Roli provides guidance, **Then** it specifies delivery coverage in Córdoba Capital, delivery days (viernes, sábado y domingo), and warming recommendations (5-10 seconds in microwave, oven-safe container).

---

### Edge Cases

- **Missing Customer Name**: If WhatsApp profile name is blank or missing, `CustomerName` defaults to "Cliente" in `AgentSessionContext`.
- **Template Rendering Error**: If template parsing or execution encounters an unexpected error, the system must log the error and fall back gracefully to a safe built-in system prompt.
- **Pending Order Resumed**: If `PendingOrderID` is not empty and customer sends payment confirmation ("ya pagué", "ya transferí"), Roli directly triggers payment verification.
- **Empty or Inactive Catalog**: If `consultar_catalogo` returns no active items, Roli informs the customer that all batches are currently sold out.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST store the declarative agent instructions in a dedicated template file located at `core/usecase/prompts/cinnamon_sales_agent.tmpl`.
- **FR-002**: The system MUST embed the template file into the Go binary at compile-time using the Go `//go:embed` directive.
- **FR-003**: The system MUST provide an `AgentPromptBuilder` (or builder component) in `core/usecase` capable of rendering the template using Go standard library `text/template`.
- **FR-004**: The system MUST accept an `AgentSessionContext` struct containing:
  - `CurrentTime` (string, e.g. "Sábado 17:15")
  - `CustomerName` (string)
  - `CustomerPhone` (string)
  - `IsCashRegisterOpen` (bool)
  - `HasSavedAddresses` (bool)
  - `PendingOrderID` (string)
- **FR-005**: The system MUST evaluate `IsCashRegisterOpen` before each turn and apply a `ToolConfig` restriction to the GenAI client config:
  - If `false`, `AllowedFunctionNames` MUST be restricted to `["consultar_catalogo", "verificar_estado_pago"]`.
  - If `true`, all registered functions MUST be allowed.
- **FR-006**: The system MUST strictly enforce that the agent's name is "Roli" and persona is grounded in Córdoba, Argentina.
- **FR-007**: The system MUST prohibit the model from using technical terms ("topping" or "toppings") when speaking to customers, substituting with "sabores", "variedades" or "roles".
- **FR-008**: The system MUST require explicit user confirmation before executing `crear_pedido_con_link_pago`.
- **FR-009**: The system MUST incorporate official CaneRollss copy templates:
  - Catalog welcome: *"¡Te compartimos nuestro menú! 🤎🍥"*
  - Official checkout receipt: Alias `Caneroll.mp`, Titular `Ivonne Abreu`
  - Warming instructions: 5-10 seconds in microwave
  - Delivery coverage: Córdoba Capital, delivery days viernes a domingo.
- **FR-010**: The system MUST use `Role: "user"` for function response parts fed back to Gemini API to maintain protocol compatibility with Google GenAI.
- **FR-011**: The system MUST support configurable model selection via `GEMINI_MODEL`, defaulting to `gemini-3.1-flash-lite` with automatic fallback.

### Key Entities

- **AgentSessionContext**: Represents the dynamic session state of an ongoing conversation (customer metadata, temporal context, business register status, address existence, pending order state).
- **AgentPromptBuilder**: Responsible for compiling, caching, and rendering the DSL prompt template with live session variables into a ready-to-use system instruction.
- **ToolSecurityPolicy**: Encapsulates dynamic function calling restrictions (`ToolConfig`) depending on business constraints (e.g., register open vs. closed).

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: System instruction rendering from embedded template executes in under 2 milliseconds with zero heap allocation overhead for template compilation (parsed once at startup).
- **SC-002**: 100% of attempts to call `crear_pedido_con_link_pago` when `IsCashRegisterOpen == false` are rejected at the protocol level by `ToolConfig`.
- **SC-003**: 100% of customer catalog inquiries receive responses using the "Roli" persona without mentioning the internal term "topping".
- **SC-004**: Order breakdown summaries match the official bakery receipt format including Alias `Caneroll.mp` and account owner `Ivonne Abreu`.

---

## Assumptions

- The WhatsApp Cloud API provides the sender's phone number as the customer identifier (`wa_id`).
- Profile names may be empty if the customer does not have a public WhatsApp display name; fallback to "Cliente" is acceptable.
- Current server time formatted in Argentina timezone (UTC-3) accurately reflects the local business time in Córdoba.
