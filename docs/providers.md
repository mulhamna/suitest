# Provider Configuration

suitest is provider-agnostic. Configure your preferred AI provider in `~/.suitest/config.yaml`.

## Auto-detection

If `--provider` is not set, suitest detects from environment:

| Env var | Provider |
|---|---|
| `ANTHROPIC_API_KEY` | claude |
| `OPENAI_API_KEY` | openai |
| `OPENROUTER_API_KEY` | openrouter |
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

## Adding a new provider

See [Provider implementation guide](../CLAUDE.md#provider-implementation-guide) in CLAUDE.md.
