package providers

const ollamaBaseURL = "http://localhost:11434/v1"

// OllamaProvider wraps OpenAIProvider with Ollama defaults.
// Ollama exposes an OpenAI-compatible API at /v1.
type OllamaProvider struct {
	*OpenAIProvider
}

// NewOllamaProvider creates an Ollama provider (OpenAI-compatible).
func NewOllamaProvider(model, baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = ollamaBaseURL
	}
	if model == "" {
		model = "llama3"
	}
	p := NewOpenAIProvider("", model, baseURL)
	p.name = "ollama"
	return &OllamaProvider{OpenAIProvider: p}
}

func (p *OllamaProvider) Name() string { return "ollama" }
