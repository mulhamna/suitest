package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mulhamna/suitest/internal/agent"
	"github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/internal/providers"
	"github.com/mulhamna/suitest/internal/report"
	"github.com/mulhamna/suitest/internal/runners"
)

// Handlers implements the MCP tool handlers.
type Handlers struct {
	cfg *config.Config
}

// NewHandlers creates Handlers.
func NewHandlers(cfg *config.Config) *Handlers {
	return &Handlers{cfg: cfg}
}

// ToolCallParams is the params for tools/call.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// HandleToolCall dispatches a tool call.
func (h *Handlers) HandleToolCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var tcp ToolCallParams
	if err := json.Unmarshal(params, &tcp); err != nil {
		return nil, fmt.Errorf("parse tool call params: %w", err)
	}

	switch tcp.Name {
	case "suitest_run":
		return h.handleRun(ctx, tcp.Arguments)
	case "suitest_plan":
		return h.handlePlan(ctx, tcp.Arguments)
	case "suitest_get_report":
		return h.handleGetReport(ctx, tcp.Arguments)
	case "suitest_fix":
		return h.handleFix(ctx, tcp.Arguments)
	case "suitest_init":
		return h.handleInit(ctx, tcp.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", tcp.Name)
	}
}

type runArgs struct {
	Path       string `json:"path"`
	Mode       string `json:"mode"`
	Provider   string `json:"provider"`
	Fix        bool   `json:"fix"`
	DryRun     bool   `json:"dry_run"`
	MaxRetries int    `json:"max_retries"`
}

func (h *Handlers) handleRun(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var a runArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(a.Path)
	if err != nil {
		return nil, err
	}

	cfg := h.cfg
	if a.Provider != "" {
		cfg.DefaultProvider = a.Provider
	}

	provider, err := providers.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("init provider: %w", err)
	}

	mode := a.Mode
	if mode == "" || mode == "auto" {
		mode, _ = runners.Detect(absPath)
	}

	maxRetries := a.MaxRetries
	if maxRetries == 0 {
		maxRetries = cfg.Agent.MaxRetries
	}

	ag := agent.New(agent.Config{
		Provider:    provider,
		Path:        absPath,
		Mode:        mode,
		MaxRetries:  maxRetries,
		Concurrency: cfg.Agent.Concurrency,
		AutoFix:     a.Fix,
		DryRun:      a.DryRun,
	})

	result, err := ag.Run(ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type planArgs struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

func (h *Handlers) handlePlan(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var a planArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}
	a.Mode = "auto"
	rawArgs, _ := json.Marshal(runArgs{Path: a.Path, Mode: a.Mode, DryRun: true})
	return h.handleRun(ctx, rawArgs)
}

type getReportArgs struct {
	Format string `json:"format"`
}

func (h *Handlers) handleGetReport(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var a getReportArgs
	json.Unmarshal(args, &a)
	if a.Format == "" {
		a.Format = "json"
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(home, ".suitest", "last-report.json"))
	if err != nil {
		return nil, fmt.Errorf("no report found: %w", err)
	}

	var result agent.RunResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	switch a.Format {
	case "markdown":
		var buf bytes.Buffer
		r := report.NewMarkdownReporter()
		r.Write(&buf, &result)
		return map[string]string{"content": buf.String(), "format": "markdown"}, nil
	default:
		return result, nil
	}
}

type fixArgs struct {
	File     string `json:"file"`
	Error    string `json:"error"`
	Provider string `json:"provider"`
}

func (h *Handlers) handleFix(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var a fixArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(a.File)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", a.File, err)
	}

	cfg := h.cfg
	if a.Provider != "" {
		cfg.DefaultProvider = a.Provider
	}

	provider, err := providers.New(cfg)
	if err != nil {
		return nil, err
	}

	msgs := []providers.Message{
		{
			Role:    providers.RoleSystem,
			Content: "You are an expert debugging assistant. Fix the failing test code. Return ONLY the fixed code.",
		},
		{
			Role: providers.RoleUser,
			Content: fmt.Sprintf("Fix this test file:\n\n%s\n\nError:\n%s\n\nReturn only the fixed code.",
				string(data), a.Error),
		},
	}

	fixed, err := provider.Complete(ctx, msgs, providers.CompleteOptions{MaxTokens: 4096})
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"file":  a.File,
		"fixed": fixed,
	}, nil
}

type initArgs struct {
	Path string `json:"path"`
}

func (h *Handlers) handleInit(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var a initArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(a.Path)
	if err != nil {
		return nil, err
	}

	mode, _ := runners.Detect(absPath)

	configPath := filepath.Join(absPath, ".suitest.yaml")
	content := fmt.Sprintf("mode: %s\nprovider: auto\ntest_dir: ./tests\n", mode)

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return nil, err
	}

	return map[string]string{
		"config":  configPath,
		"mode":    mode,
		"message": "suitest initialized successfully",
	}, nil
}
