# Provider Configuration

suitest is provider-agnostic. Configure your preferred AI provider in `~/.suitest/config.yaml`.

## Auto-detection

If `--provider` is not set, suitest detects from environment and available local CLIs:

| Env var | Provider |
|---|---|
| `ANTHROPIC_API_KEY` | claude |
| Claude Code CLI available | claude-cli |
| `OPENAI_API_KEY` | openai |
| `OPENROUTER_API_KEY` | openrouter |
| Codex CLI available | codex-cli |
| Gemini CLI available | gemini-cli |
| Ollama on localhost:11434 | ollama |

## Claude (Anthropic)

```yaml
providers:
  claude:
    api_key: "${ANTHROPIC_API_KEY}"
    model: claude-sonnet-4-20250514
```

```bash
suitest run . --provider claude
```

## Claude Code CLI

Use an existing Claude Code login instead of an API key.

```yaml
providers:
  claude-cli:
    model: claude-sonnet-4-20250514
```

```bash
suitest run . --provider claude-cli
```

## Codex CLI

Use an existing Codex CLI login instead of an API key.

```yaml
providers:
  codex-cli: {}
```

```bash
suitest run . --provider codex-cli
```

## Gemini CLI

Use an existing Gemini CLI login instead of an API key.

```yaml
providers:
  gemini-cli: {}
```

```bash
suitest run . --provider gemini-cli
```

## OpenAI

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: gpt-4o
```

```bash
suitest run . --provider openai
```

## OpenRouter

Supports Groq, Mistral, Llama, and 200+ models.

```yaml
providers:
  openrouter:
    api_key: "${OPENROUTER_API_KEY}"
    model: mistral/mistral-7b-instruct
```

```bash
suitest run . --provider openrouter --model meta-llama/llama-3-70b-instruct
```

## Ollama (local)

Run models locally with no API key required.

```bash
ollama pull llama3
```

```yaml
providers:
  ollama:
    base_url: http://localhost:11434
    model: llama3
```

```bash
suitest run . --provider ollama --model llama3
```

## Custom OpenAI-compatible endpoint

Any OpenAI-compatible API works via `--base-url`:

```bash
suitest run . --provider openai --base-url https://my-endpoint.com/v1 --model my-model
```

## Notes on CLI-backed providers

CLI-backed providers shell out to the local authenticated tool in non-interactive mode.

Current behavior:
- `claude-cli` uses `claude --print --permission-mode bypassPermissions`
- `codex-cli` uses `codex exec`
- `gemini-cli` uses `gemini --prompt`

Tradeoffs:
- simpler onboarding for users without API keys
- depends on the local CLI being installed and logged in
- output shape can be less predictable than raw API integrations

## Adding a new provider

See [Provider implementation guide](../CLAUDE.md#provider-implementation-guide) in CLAUDE.md.
