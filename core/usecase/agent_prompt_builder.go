package usecase

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed prompts/cinnamon_sales_agent.tmpl
var defaultPromptTemplate string

// AgentSessionContext holds the live dynamic context injected on each turn
type AgentSessionContext struct {
	CurrentTime        string
	CustomerName       string
	CustomerPhone      string
	IsCashRegisterOpen bool
	HasSavedAddresses  bool
	PendingOrderID     string
}

// AgentPromptBuilder parses and renders the DSL system prompt
type AgentPromptBuilder struct {
	tmpl *template.Template
}

// NewAgentPromptBuilder compiles the embedded template
func NewAgentPromptBuilder() (*AgentPromptBuilder, error) {
	tmpl, err := template.New("cinnamon_sales_agent").Parse(defaultPromptTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing cinnamon_sales_agent.tmpl: %w", err)
	}
	return &AgentPromptBuilder{tmpl: tmpl}, nil
}

// BuildSystemInstruction renders the prompt with dynamic session variables
func (b *AgentPromptBuilder) BuildSystemInstruction(ctx AgentSessionContext) (string, error) {
	// Sanitize / Fallbacks
	if strings.TrimSpace(ctx.CustomerName) == "" {
		ctx.CustomerName = "Cliente"
	}
	if strings.TrimSpace(ctx.CurrentTime) == "" {
		ctx.CurrentTime = "Horario habitual"
	}

	var buf bytes.Buffer
	if err := b.tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("error rendering cinnamon_sales_agent template: %w", err)
	}

	return buf.String(), nil
}
