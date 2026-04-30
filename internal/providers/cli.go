package providers

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CLIProvider implements Provider by shelling out to an authenticated local AI CLI.
type CLIProvider struct {
	name    string
	command string
	args    []string
	model   string
}

// NewCLIProvider creates a provider backed by a local CLI binary.
func NewCLIProvider(name, command string, baseArgs []string, model string) *CLIProvider {
	return &CLIProvider{
		name:    name,
		command: command,
		args:    append([]string{}, baseArgs...),
		model:   model,
	}
}

func (p *CLIProvider) Name() string { return p.name }

func (p *CLIProvider) Complete(ctx context.Context, messages []Message, opts CompleteOptions) (string, error) {
	prompt := buildCLIPrompt(messages)
	args := append([]string{}, p.args...)

	switch p.name {
	case "claude-cli":
		if p.model != "" {
			args = append(args, "--model", p.model)
		}
		args = append(args, prompt)
	case "codex-cli":
		if p.model != "" {
			args = append(args, "--model", p.model)
		}
		args = append(args, prompt)
	case "gemini-cli":
		if p.model != "" {
			args = append(args, "--model", p.model)
		}
		args = append(args, prompt)
	default:
		return "", fmt.Errorf("unsupported CLI provider: %s", p.name)
	}

	cmd := exec.CommandContext(ctx, p.command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s failed: %s", p.command, msg)
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", fmt.Errorf("%s returned empty output", p.command)
	}
	return result, nil
}

func (p *CLIProvider) Stream(ctx context.Context, messages []Message, opts CompleteOptions) (<-chan string, error) {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		result, err := p.Complete(ctx, messages, opts)
		if err != nil {
			return
		}
		select {
		case ch <- result:
		case <-ctx.Done():
		}
	}()
	return ch, nil
}

func buildCLIPrompt(messages []Message) string {
	var b strings.Builder
	for _, m := range messages {
		role := strings.ToUpper(strings.TrimSpace(m.Role))
		if role == "" {
			role = "USER"
		}
		b.WriteString(role)
		b.WriteString(":\n")
		b.WriteString(strings.TrimSpace(m.Content))
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}
