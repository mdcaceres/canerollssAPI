package usecase

import (
	"strings"
	"testing"
)

func TestNewAgentPromptBuilder(t *testing.T) {
	builder, err := NewAgentPromptBuilder()
	if err != nil {
		t.Fatalf("expected NewAgentPromptBuilder to succeed, got error: %v", err)
	}
	if builder == nil {
		t.Fatal("expected builder to be non-nil")
	}
}

func TestAgentPromptBuilder_BuildSystemInstruction_FullContext(t *testing.T) {
	builder, err := NewAgentPromptBuilder()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	ctx := AgentSessionContext{
		CurrentTime:        "Sábado 17:15",
		CustomerName:       "Sofía",
		CustomerPhone:      "5493519876543",
		IsCashRegisterOpen: true,
		HasSavedAddresses:  true,
		PendingOrderID:     "105",
	}

	rendered, err := builder.BuildSystemInstruction(ctx)
	if err != nil {
		t.Fatalf("unexpected rendering error: %v", err)
	}

	// Verify Top-Level XML structure and persona
	expectedSubstrings := []string{
		"<agent_profile>",
		"<name>Roli</name>",
		"CaneRolls",
		"<live_session_context>",
		"<current_time>Sábado 17:15</current_time>",
		"<customer_name>Sofía</customer_name>",
		"<customer_phone>5493519876543</customer_phone>",
		"<cash_register_open>true</cash_register_open>",
		"<has_saved_addresses>true</has_saved_addresses>",
		"<active_pending_order_id>105</active_pending_order_id>",
		"<strict_guardrails>",
		"<conversation_state_machine>",
		"<examples_of_interaction>",
		"Caneroll.mp",
		"Ivonne Abreu",
		"¡Te compartimos nuestro menú! 🤎🍥",
		"5-10 segundos en microondas",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(rendered, sub) {
			t.Errorf("rendered prompt missing expected string: %q", sub)
		}
	}
}

func TestAgentPromptBuilder_BuildSystemInstruction_Fallbacks(t *testing.T) {
	builder, err := NewAgentPromptBuilder()
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Empty customer name and current time
	ctx := AgentSessionContext{
		CurrentTime:        "",
		CustomerName:       "",
		CustomerPhone:      "5491122334455",
		IsCashRegisterOpen: false,
		HasSavedAddresses:  false,
		PendingOrderID:     "",
	}

	rendered, err := builder.BuildSystemInstruction(ctx)
	if err != nil {
		t.Fatalf("unexpected rendering error: %v", err)
	}

	if !strings.Contains(rendered, "<customer_name>Cliente</customer_name>") {
		t.Errorf("expected customer name fallback to 'Cliente', got:\n%s", rendered)
	}

	if !strings.Contains(rendered, "<current_time>Horario habitual</current_time>") {
		t.Errorf("expected current time fallback to 'Horario habitual', got:\n%s", rendered)
	}

	if !strings.Contains(rendered, "<cash_register_open>false</cash_register_open>") {
		t.Errorf("expected cash_register_open false, got:\n%s", rendered)
	}

	if !strings.Contains(rendered, "<has_saved_addresses>false</has_saved_addresses>") {
		t.Errorf("expected has_saved_addresses false, got:\n%s", rendered)
	}
}
