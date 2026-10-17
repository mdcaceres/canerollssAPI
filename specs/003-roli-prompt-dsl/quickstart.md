# Quickstart Validation Guide: Roli Prompt-as-Code DSL & Tool Security

**Feature**: `003-roli-prompt-dsl`  
**Date**: 2026-10-16  

---

## 1. Prerequisites
- Go 1.26 installed.
- Valid Google Gemini API Key configured in `.env` (`GEMINI_API_KEY`).
- `USE_MOCK_PROVIDERS=true` for local isolated validation.

---

## 2. Test Execution

### A. Unit Testing: Template Rendering & Context Injection
Run unit tests verifying that `AgentPromptBuilder` compiles the template and renders dynamic values without error:

```powershell
go test -v ./core/usecase/... -run TestAgentPromptBuilder
```

**Expected Outcome**:
- Template compiles cleanly via `//go:embed`.
- Substituted fields (`CurrentTime`, `CustomerName`, `IsCashRegisterOpen`, etc.) match expected XML tags.

---

### B. Unit Testing: Deterministic Function Gating
Verify that closing the cash register disables `crear_pedido_con_link_pago` in `AllowedFunctionNames`:

```powershell
go test -v ./core/usecase/... -run TestToolSecurityGating
```

**Expected Outcome**:
- When `IsCashRegisterOpen == false`, `AllowedFunctionNames` only contains `["consultar_catalogo", "verificar_estado_pago"]`.

---

### C. Live Integration Testing: End-to-End Persona Verification
Execute the real GenAI multi-turn interaction test against Gemini with Roli's persona:

```powershell
go test -v ./tests/... -run TestLiveGeminiIntegration
```

**Expected Outcome**:
1. Turn 1 (Catalog): Roli responds with the CaneRollss welcome copy ("¡Te compartimos nuestro menú! 🤎🍥"), listing flavors with bullet points and without using the word "topping".
2. Turn 2 (Delivery & Quote): Roli quotes the delivery address, provides the receipt breakdown, and asks for confirmation.
3. Turn 3 (Order & Link): Roli issues the order with the Mercado Pago checkout link and bank transfer Alias `Caneroll.mp` (Titular: Ivonne Abreu).
