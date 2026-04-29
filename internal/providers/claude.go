package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	claudeDefaultBaseURL = "https://api.anthropic.com"
	claudeAPIVersion     = "2023-06-01"
	claudeDefaultModel   = "claude-sonnet-4-20250514"
)

// ClaudeProvider implements Provider for the Anthropic Claude API.
type ClaudeProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewClaudeProvider creates a Claude provider.
func NewClaudeProvider(apiKey, model, baseURL string) *ClaudeProvider {
	if baseURL == "" {
		baseURL = claudeDefaultBaseURL
	}
	if model == "" {
		model = claudeDefaultModel
	}
	return &ClaudeProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *ClaudeProvider) Name() string { return "claude" }

func (p *ClaudeProvider) Complete(ctx context.Context, messages []Message, opts CompleteOptions) (string, error) {
	// Extract system message if present
	var system string
	var convMsgs []claudeMessage
	for _, m := range messages {
		if m.Role == RoleSystem {
			system = m.Content
		} else {
			convMsgs = append(convMsgs, claudeMessage{Role: m.Role, Content: m.Content})
		}
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	reqBody := claudeRequest{
		Model:     p.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  convMsgs,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result claudeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error [%s]: %s", result.Error.Type, result.Error.Message)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return result.Content[0].Text, nil
}

func (p *ClaudeProvider) Stream(ctx context.Context, messages []Message, opts CompleteOptions) (<-chan string, error) {
	// For streaming with Claude, we use a non-streaming call and send full response at once.
	// Full SSE streaming can be added as an enhancement.
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
