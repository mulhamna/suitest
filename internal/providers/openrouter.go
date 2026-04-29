package providers

const openRouterBaseURL = "https://openrouter.ai/api/v1"

// OpenRouterProvider wraps OpenAIProvider with OpenRouter defaults.
type OpenRouterProvider struct {
	*OpenAIProvider
}

// NewOpenRouterProvider creates an OpenRouter provider (OpenAI-compatible).
func NewOpenRouterProvider(apiKey, model, baseURL string) *OpenRouterProvider {
	if baseURL == "" {
		baseURL = openRouterBaseURL
	}
	if model == "" {
		model = "mistral/mistral-7b-instruct"
	}
	p := NewOpenAIProvider(apiKey, model, baseURL)
	p.name = "openrouter"
	return &OpenRouterProvider{OpenAIProvider: p}
}

func (p *OpenRouterProvider) Name() string { return "openrouter" }
