package providers

import "context"

// Role constants for message roles.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message is a single message in a conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompleteOptions controls how the provider generates a completion.
type CompleteOptions struct {
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	StopWords   []string `json:"stop,omitempty"`
}

// Provider is the interface all AI providers must implement.
type Provider interface {
	// Name returns the provider's identifier (e.g., "claude", "openai").
	Name() string

	// Complete sends messages and returns the full response string.
	Complete(ctx context.Context, messages []Message, opts CompleteOptions) (string, error)

	// Stream sends messages and returns a channel that streams response tokens.
	Stream(ctx context.Context, messages []Message, opts CompleteOptions) (<-chan string, error)
}
