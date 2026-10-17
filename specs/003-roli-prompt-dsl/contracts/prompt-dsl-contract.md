# Contract: Prompt-as-Code DSL Specification

**Feature**: `003-roli-prompt-dsl`  
**File Path**: `core/usecase/prompts/cinnamon_sales_agent.tmpl`  

---

## 1. Top-Level XML Tag Contract

The template MUST define and preserve the following XML blocks:

```xml
<agent_profile>
  <name>Roli</name>
  <business>CaneRolls (Roles de canela artesanales y recién horneados en Córdoba)</business>
  <tone>Español rioplatense de Córdoba (Argentina)...</tone>
</agent_profile>

<live_session_context>
  <current_time>{{.CurrentTime}}</current_time>
  <customer_name>{{.CustomerName}}</customer_name>
  <customer_phone>{{.CustomerPhone}}</customer_phone>
  <cash_register_open>{{.IsCashRegisterOpen}}</cash_register_open>
  <has_saved_addresses>{{.HasSavedAddresses}}</has_saved_addresses>
  <active_pending_order_id>{{.PendingOrderID}}</active_pending_order_id>
</live_session_context>

<strict_guardrails>
  <!-- 6 Golden Rules -->
</strict_guardrails>

<conversation_state_machine>
  <!-- 5 Sequential Steps -->
</conversation_state_machine>

<examples_of_interaction>
  <!-- WhatsApp multi-turn dialogues with official CaneRollss copies -->
</examples_of_interaction>
```

---

## 2. Injected Values Substitution Contract

| Variable Tag | Data Source | Sample Rendered Output |
|---|---|---|
| `{{.CurrentTime}}` | Argentina Timezone (UTC-3) | `<current_time>Sábado 17:15</current_time>` |
| `{{.CustomerName}}` | WhatsApp Profile / "Cliente" | `<customer_name>Sofía</customer_name>` |
| `{{.CustomerPhone}}` | `msg.From` (`wa_id`) | `<customer_phone>5491198765432</customer_phone>` |
| `{{.IsCashRegisterOpen}}` | CashRegister Repository status | `<cash_register_open>true</cash_register_open>` |
| `{{.HasSavedAddresses}}` | Address Repository count > 0 | `<has_saved_addresses>false</has_saved_addresses>` |
| `{{.PendingOrderID}}` | Pending Order ID string | `<active_pending_order_id>14</active_pending_order_id>` |
