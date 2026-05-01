# suitest

> AI-powered testing agent CLI — provider-agnostic, no lock-in, no subscription.

suitest automatically generates, executes, and debugs tests for your project — frontend, backend, and API. Bring your own AI provider: Claude, OpenAI, OpenRouter, Ollama, Claude Code CLI, Codex CLI, Gemini CLI, or any OpenAI-compatible endpoint.

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
# or use a logged-in CLI provider:
# suitest run . --provider claude-cli
# suitest run . --provider codex-cli
# suitest run . --provider gemini-cli

# Initialize in your project
suitest init

# Save a reusable target
suitest target create frontend-checkout \
  --type frontend \
  --path . \
  --url http://localhost:3000 \
  --expectation "User opens homepage, logs in, and reaches the dashboard"

# Map scenarios once
suitest scenario map frontend-checkout

# Run the saved target
suitest run frontend-checkout
```

---

## Providers

suitest auto-detects your provider from environment variables or available logged-in CLIs:

| Env var | Provider |
|---|---|
| `ANTHROPIC_API_KEY` | Claude (Anthropic API) |
| Claude Code CLI available | Claude Code CLI |
| `OPENAI_API_KEY` | OpenAI |
| `OPENROUTER_API_KEY` | OpenRouter |
| Codex CLI available | Codex CLI |
| Gemini CLI available | Gemini CLI |
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

Run the first-time onboarding flow. `init` now asks for storage, operator mode, FE/BE target type, URL or curl seed, and expected success flow, then saves a reusable target.

```bash
suitest init
```

### `suitest run [path]`

Run the full agent loop. You can run an ad-hoc path or a saved target.

```bash
suitest run .
suitest run ./src --mode browser
suitest run ./api --mode api --provider claude
suitest run frontend-checkout
suitest run frontend-checkout --scenario-set smoke
suitest run . --dry-run          # show plan only
suitest run . --fix              # auto-apply AI fixes to source
```

| Flag | Default | Description |
|---|---|---|
| `--mode` | auto | `auto`, `browser`, `api`, `unit` |
| `--provider` | auto | `claude`, `claude-cli`, `codex-cli`, `gemini-cli`, `openai`, `openrouter`, `ollama` |
| `--model` | provider default | Model name/slug |
| `--base-url` | provider default | Custom OpenAI-compatible endpoint |
| `--fix` | false | Auto-apply AI-suggested fixes |
| `--dry-run` | false | Plan only, no test execution |
| `--output` | terminal | `terminal`, `json`, `markdown` |
| `--max-retries` | 3 | Fix retry attempts per test |
| `--concurrency` | 4 | Parallel test runners |
| `--target` | empty | Saved target name to run |
| `--scenario-set` | target default | Saved scenario set for target runs |
| `--yes` | false | Skip confirmation for saved target runs |

### `suitest target`

Create and reuse explicit test targets so QA can rerun the same thing tomorrow without re-entering URL or curl input.

```bash
suitest target create frontend-checkout --type frontend --path . --url http://localhost:3000 --expectation "User logs in successfully"
suitest target list
```

### `suitest scenario`

Map and persist scenario sets separately from run results. Generated scenarios can stay as drafts until QA approves them.

```bash
suitest scenario map frontend-checkout
suitest scenario approve frontend-checkout default
suitest scenario list frontend-checkout
```

### `suitest tui`

Open a lightweight interactive terminal view for browsing saved targets, scenario sets, and run history.

```bash
suitest tui
```

### `suitest settings`

View or update global operating preferences separately from targets and run results.

```bash
suitest settings
suitest settings set storage.driver json
suitest settings set operator.mode native
```

### `suitest report`

Print or export the last run report.

```bash
suitest report
suitest report history
suitest report show 20260501-153000
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

storage:
  driver: json
  # path: ~/.suitest/suitest.db   # optional when using sqlite

operator:
  mode: native

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

Storage behavior:
- `json`: targets/scenarios are stored as local files under `~/.suitest/`, and the latest run report is saved to `~/.suitest/last-report.json`
- `sqlite`: targets, scenario sets, and the latest run report are stored in `~/.suitest/suitest.db` unless `storage.path` overrides it

Project-level override at `.suitest.yaml`:

```yaml
mode: browser
provider: claude-cli
entry_url: http://localhost:3000
test_dir: ./tests
```

## No API key mode

If a user is already logged into a local coding CLI, suitest can use that auth instead of direct API keys:

```bash
suitest run . --provider claude-cli
suitest run . --provider codex-cli
suitest run . --provider gemini-cli
```

This is useful when users already work through Claude Code, Codex CLI, or Gemini CLI and do not want to manage separate API keys.

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

Release smoke test for release-please.
