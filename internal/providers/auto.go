package providers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/mulhamna/suitest/internal/config"
)

// New creates a Provider based on the configuration.
// It resolves the "auto" provider by inspecting environment variables.
func New(cfg *config.Config) (Provider, error) {
	providerName := cfg.DefaultProvider
	if providerName == "" || providerName == "auto" {
		providerName = detectProvider()
	}

	pc := cfg.GetProviderConfig(providerName)

	// Allow config-level model override
	if cfg.Model != "" {
		pc.Model = cfg.Model
	}

	switch providerName {
	case "claude-cli":
		if _, err := exec.LookPath("claude"); err != nil {
			return nil, fmt.Errorf("claude-cli provider requires Claude Code CLI installed and authenticated")
		}
		return NewCLIProvider("claude-cli", "claude", []string{"--print", "--permission-mode", "bypassPermissions"}, pc.Model), nil

	case "codex-cli":
		if _, err := exec.LookPath("codex"); err != nil {
			return nil, fmt.Errorf("codex-cli provider requires Codex CLI installed and authenticated")
		}
		return NewCLIProvider("codex-cli", "codex", []string{"exec"}, pc.Model), nil

	case "gemini-cli":
		if _, err := exec.LookPath("gemini"); err != nil {
			return nil, fmt.Errorf("gemini-cli provider requires Gemini CLI installed and authenticated")
		}
		return NewCLIProvider("gemini-cli", "gemini", []string{"--prompt"}, pc.Model), nil

	case "claude":
		apiKey := pc.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("claude provider requires ANTHROPIC_API_KEY or providers.claude.api_key in config")
		}
		return NewClaudeProvider(apiKey, pc.Model, pc.BaseURL), nil

	case "openai":
		apiKey := pc.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("openai provider requires OPENAI_API_KEY or providers.openai.api_key in config")
		}
		return NewOpenAIProvider(apiKey, pc.Model, pc.BaseURL), nil

	case "openrouter":
		apiKey := pc.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENROUTER_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("openrouter provider requires OPENROUTER_API_KEY or providers.openrouter.api_key in config")
		}
		return NewOpenRouterProvider(apiKey, pc.Model, pc.BaseURL), nil

	case "ollama":
		baseURL := pc.BaseURL
		if baseURL == "" {
			baseURL = ollamaBaseURL
		}
		return NewOllamaProvider(pc.Model, baseURL), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: claude, claude-cli, openai, openrouter, ollama, codex-cli, gemini-cli)", providerName)
	}
}

// detectProvider inspects environment variables to pick a provider automatically.
func detectProvider() string {
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return "claude"
	}
	if hasCommand("claude") {
		return "claude-cli"
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return "openai"
	}
	if os.Getenv("OPENROUTER_API_KEY") != "" {
		return "openrouter"
	}
	if hasCommand("codex") {
		return "codex-cli"
	}
	if hasCommand("gemini") {
		return "gemini-cli"
	}
	if isOllamaRunning() {
		return "ollama"
	}
	return "claude-cli" // fallback if an authenticated local CLI is available, else init will fail clearly
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// isOllamaRunning checks if a local Ollama instance is reachable.
func isOllamaRunning() bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:11434/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
