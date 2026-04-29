# suitest

> AI-powered testing agent CLI — provider-agnostic, no lock-in, no subscription.

suitest automatically generates, executes, and debugs tests for your project — frontend, backend, and API. Bring your own AI provider: Claude, OpenAI, OpenRouter, Ollama, or any OpenAI-compatible endpoint.

---

## Install

### Homebrew (macOS / Linux)

```bash
brew tap mulhamna/tap
brew install suitest
```

### Scoop (Windows)

```powershell
scoop bucket add mulhamna https://github.com/mulhamna/scoop-bucket
scoop install suitest
```

### Go

```bash
go install github.com/mulhamna/suitest@latest
```

### Download binary

Grab the latest release from [GitHub Releases](https://github.com/mulhamna/suitest/releases).

---

## Quick start

```bash
# Configure your provider
export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY, OPENROUTER_API_KEY

# Initialize in your project
suitest init

# Run tests
suitest run .
```

---

## Providers

suitest auto-detects your provider from environment variables:

| Env var | Provider |
|---|---|
| `ANTHROPIC_API_KEY` | Claude (Anthropic) |
| `OPENAI_API_KEY` | OpenAI |
| `OPENROUTER_API_KEY` | OpenRouter |
| Ollama on localhost | Ollama (local) |

Override with `--provider` and `--model`:

```bash
suitest run . --provider openrouter --model mistral/mistral-7b-instruct
suitest run . --provider ollama --model llama3
suitest run . --provider openai --base-url https://my-endpoint.com/v1
```

---

## Commands

### `suitest init`

Scaffold `.suitest.yaml` in your project root.

```bash
suitest init
```

### `suitest run [path]`

Run the full agent loop: discover → plan → generate → execute → fix → report.

```bash
suitest run .
suitest run ./src --mode browser
suitest run ./api --mode api --provider claude
suitest run . --dry-run          # show plan only
suitest run . --fix              # auto-apply AI fixes to source
```

| Flag | Default | Description |
|---|---|---|
| `--mode` | auto | `auto`, `browser`, `api`, `unit` |
| `--provider` | auto | `claude`, `openai`, `openrouter`, `ollama` |
| `--model` | provider default | Model name/slug |
| `--base-url` | provider default | Custom OpenAI-compatible endpoint |
| `--fix` | false | Auto-apply AI-suggested fixes |
| `--dry-run` | false | Plan only, no test execution |
| `--output` | terminal | `terminal`, `json`, `markdown` |
| `--max-retries` | 3 | Fix retry attempts per test |
| `--concurrency` | 4 | Parallel test runners |

### `suitest report`

Print or export the last run report.

```bash
suitest report
suitest report --format json > report.json
suitest report --format markdown > REPORT.md
```

### `suitest serve`

Start the MCP server for IDE integration (Claude Code, Cursor, Windsurf).

```bash
suitest serve
suitest serve --port 3100
```

---

## Config

Global config at `~/.suitest/config.yaml`:

```yaml
default_provider: openrouter

providers:
  claude:
    api_key: "${ANTHROPIC_API_KEY}"
    model: claude-sonnet-4-20250514

  openai:
    api_key: "${OPENAI_API_KEY}"
    model: gpt-4o

  openrouter:
    api_key: "${OPENROUTER_API_KEY}"
    model: mistral/mistral-7b-instruct

  ollama:
    base_url: http://localhost:11434
    model: llama3

agent:
  max_retries: 3
  concurrency: 4
  auto_fix: false
```

Project-level override at `.suitest.yaml`:

```yaml
mode: browser
provider: claude
entry_url: http://localhost:3000
test_dir: ./tests
```

---

## MCP server

Add to your MCP config for IDE integration:

```json
{
  "mcpServers": {
    "suitest": {
      "command": "suitest",
      "args": ["serve"]
    }
  }
}
```

Tools exposed: `suitest_run`, `suitest_plan`, `suitest_get_report`, `suitest_fix`, `suitest_init`.

See [docs/mcp.md](docs/mcp.md) for full reference.

---

## Runner detection

suitest auto-detects the test runner from your project:

| Signal | Runner |
|---|---|
| `go.mod` | go test |
| `package.json` with jest/vitest | jest / vitest |
| `requirements.txt` / `pyproject.toml` | pytest |
| `--mode browser` | chromedp (headless Chrome) |
| `--mode api` | HTTP/REST |

---

## Development

```bash
git clone https://github.com/mulhamna/suitest
cd suitest
go mod tidy
go run . run ./testdata/sample-project
```

```bash
go test ./...
go build -o suitest .
```

---

## License

MIT
