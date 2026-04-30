package config

// Default returns a Config populated with sensible defaults.
func Default() *Config {
	return &Config{
		DefaultProvider: "auto",
		Providers: map[string]*ProviderConfig{
			"claude": {
				Model: "claude-sonnet-4-20250514",
			},
			"claude-cli": {
				Model: "claude-sonnet-4-20250514",
			},
			"openai": {
				Model:   "gpt-4o",
				BaseURL: "https://api.openai.com/v1",
			},
			"openrouter": {
				Model:   "mistral/mistral-7b-instruct",
				BaseURL: "https://openrouter.ai/api/v1",
			},
			"ollama": {
				Model:   "llama3",
				BaseURL: "http://localhost:11434/v1",
			},
			"codex-cli": {
				Model: "",
			},
			"gemini-cli": {
				Model: "",
			},
		},
		Agent: AgentConfig{
			MaxRetries:  3,
			Concurrency: 4,
			AutoFix:     false,
		},
		Mode:    "auto",
		Output:  "terminal",
		TestDir: "./tests",
	}
}
