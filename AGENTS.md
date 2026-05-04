# suitest

> Open-source AI-powered testing agent CLI — provider-agnostic, works with Claude Code, Codex CLI, OpenRouter, Ollama, or any OpenAI-compatible provider.

---

## What is suitest?

`suitest` is a CLI-first testing agent that automatically generates, executes, and debugs tests for your project — frontend, backend, and API. Unlike SaaS alternatives, you bring your own AI provider. No lock-in, no subscription, no account required.

---

## Project name

**suitest** — `github.com/yourhandle/suitest`

- Binary: `suitest`
- Module: `github.com/yourhandle/suitest`
- MCP server: `suitest-mcp`
- Claude plugin namespace: `suitest`

---

## Stack

| Layer | Choice | Reason |
|---|---|---|
| Language | **Go** | Single binary, fast startup (<10ms), goroutines for concurrency, easy cross-compile |
| CLI framework | **Cobra + Viper** | Industry standard for Go CLIs |
| Browser testing | **chromedp** | Headless Chrome via CDP, no external deps |
| HTTP testing | **net/http + testify** | Built-in, zero overhead |
| MCP server | **Go HTTP SSE** | Native, no Node.js runtime needed |
| Config | **YAML** (`~/.suitest/config.yaml`) | Human-readable, versionable |

---

## Repository structure

```
suitest/
├── CLAUDE.md                  # This file — context for AI agents
├── README.md
├── go.mod
├── go.sum
├── main.go                    # Entrypoint
│
├── cmd/                       # Cobra commands
│   ├── root.go                # Root command, global flags
│   ├── run.go                 # suitest run <path>
│   ├── init.go                # suitest init
│   ├── report.go              # suitest report
│   └── serve.go               # suitest serve (MCP server mode)
│
├── internal/
│   ├── agent/                 # Core AI agent loop
│   │   ├── agent.go           # Main agent orchestration
│   │   ├── planner.go         # Test plan generation from codebase
│   │   ├── executor.go        # Execute → observe → fix loop
│   │   └── prompt/            # Prompt templates
│   │       ├── plan.txt
│   │       ├── fix.txt
│   │       └── summarize.txt
│   │
│   ├── providers/             # AI provider abstraction
│   │   ├── provider.go        # Interface definition
│   │   ├── claude.go          # Anthropic API / Claude Code passthrough
│   │   ├── openai.go          # OpenAI / Codex CLI
│   │   ├── openrouter.go      # OpenRouter (Groq, Mistral, Llama, etc.)
│   │   ├── ollama.go          # Local Ollama
│   │   └── auto.go            # Auto-detect from env/config
│   │
│   ├── runners/               # Test execution engines
│   │   ├── runner.go          # Runner interface
│   │   ├── browser.go         # E2E via chromedp
│   │   ├── api.go             # HTTP/REST API testing
│   │   ├── unit.go            # Unit test generation + go test / pytest / jest
│   │   └── detect.go          # Auto-detect project type
│   │
│   ├── report/                # Output formatting
│   │   ├── terminal.go        # Colored terminal output
│   │   ├── json.go            # JSON report
│   │   └── markdown.go        # Markdown summary
│   │
│   └── config/                # Config management
│       ├── config.go
│       └── defaults.go
│
├── mcp/                       # MCP server implementation
│   ├── server.go              # SSE server entrypoint
│   ├── tools.go               # Tool definitions (run_tests, get_report, etc.)
│   └── handlers.go            # Tool handlers
│
├── plugin/                    # Claude plugin / extension support
│   ├── manifest.json          # Plugin manifest
│   ├── openapi.yaml           # OpenAPI spec for Claude plugin
│   └── handler.go             # Plugin HTTP handler
│
└── docs/
    ├── providers.md           # How to configure each provider
    ├── mcp.md                 # MCP server setup
    └── plugin.md              # Claude plugin setup
```

---

## Core concepts

### Provider interface

All AI providers implement this single interface:

```go
// internal/providers/provider.go
type Provider interface {
    Name() string
    Complete(ctx context.Context, messages []Message, opts CompleteOptions) (string, error)
    Stream(ctx context.Context, messages []Message, opts CompleteOptions) (<-chan string, error)
}
```

Providers are selected via config or `--provider` flag. The `auto` provider detects from environment:
- `ANTHROPIC_API_KEY` → claude
- `OPENAI_API_KEY` → openai
- `OPENROUTER_API_KEY` → openrouter
- Ollama running locally → ollama

### Agent loop

```
1. Discover project (language, framework, existing tests)
2. Generate test plan via LLM
3. For each test in plan:
   a. Generate test code
   b. Execute test
   c. If fail → send error + code to LLM → get fix → retry (max 3x)
4. Aggregate results
5. Output report
```

### Runner detection

`detect.go` inspects the project root to determine runner:

| Signal | Runner |
|---|---|
| `package.json` with jest/vitest | `unit` (jest) |
| `go.mod` | `unit` (go test) |
| `requirements.txt` / `pyproject.toml` | `unit` (pytest) |
| `--mode browser` flag | `browser` (chromedp) |
| `--mode api` flag | `api` |
| `.suitest.yaml` `mode:` field | override all above |

---

## CLI commands

### `suitest init`

Scaffolds `.suitest.yaml` in project root by inspecting the codebase.

```bash
suitest init
# → .suitest.yaml created
# → guides user to set provider in ~/.suitest/config.yaml
```

### `suitest run [path]`

Main command. Runs the full agent loop.

```bash
suitest run .
suitest run ./src --mode browser --provider openrouter
suitest run ./api --mode api --model mistral/mistral-7b
suitest run . --dry-run          # Show plan only, no execution
suitest run . --fix              # Auto-apply fixes to source files
```

Flags:

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

Start the MCP server (for IDE integration).

```bash
suitest serve
suitest serve --port 3100
```

---

## Config file

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
    # Any OpenAI-compatible endpoint works:
    # base_url: https://openrouter.ai/api/v1

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
model: claude-sonnet-4-20250514
entry_url: http://localhost:3000   # for browser mode
test_dir: ./tests
```

---

## MCP server

The MCP server exposes suitest capabilities to AI IDEs (Claude Code, Cursor, Windsurf, etc.).

### Start

```bash
suitest serve --port 3100
```

Or add to MCP config:

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

### Tools exposed

| Tool | Description |
|---|---|
| `suitest_run` | Run tests on a given path with options |
| `suitest_plan` | Generate test plan without executing |
| `suitest_get_report` | Get the latest test report |
| `suitest_fix` | Apply AI fix to a failing test file |
| `suitest_init` | Initialize suitest in a project |

### Tool schema example — `suitest_run`

```json
{
  "name": "suitest_run",
  "description": "Run AI-powered tests on a project path",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": { "type": "string", "description": "Project root path" },
      "mode": { "type": "string", "enum": ["auto", "browser", "api", "unit"] },
      "provider": { "type": "string" },
      "fix": { "type": "boolean", "default": false }
    },
    "required": ["path"]
  }
}
```

---

## Claude plugin

The Claude plugin exposes suitest as an action inside Claude.ai conversations.

### Files

```
plugin/
├── manifest.json       # Plugin metadata + auth
├── openapi.yaml        # API spec Claude reads
└── handler.go          # HTTP endpoints
```

### manifest.json

```json
{
  "schema_version": "v1",
  "name_for_human": "Suitest",
  "name_for_model": "suitest",
  "description_for_human": "Run AI-powered tests on your codebase from Claude.",
  "description_for_model": "Use suitest to generate and run tests on a software project. You can run tests, get reports, and apply fixes.",
  "auth": { "type": "none" },
  "api": {
    "type": "openapi",
    "url": "http://localhost:3100/openapi.yaml"
  }
}
```

### Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/run` | Run tests |
| `GET` | `/report` | Get latest report |
| `POST` | `/plan` | Get test plan (dry run) |
| `POST` | `/fix` | Apply fix to file |

---

## Provider implementation guide

To add a new provider, create `internal/providers/myprovider.go`:

```go
package providers

type MyProvider struct {
    apiKey  string
    model   string
    baseURL string
}

func (p *MyProvider) Name() string { return "myprovider" }

func (p *MyProvider) Complete(ctx context.Context, messages []Message, opts CompleteOptions) (string, error) {
    // Call your provider's API here
    // Must return the full response string
}

func (p *MyProvider) Stream(ctx context.Context, messages []Message, opts CompleteOptions) (<-chan string, error) {
    // Return a channel that streams tokens
}
```

Register in `auto.go`:

```go
case "myprovider":
    return &MyProvider{
        apiKey:  cfg.APIKey,
        model:   cfg.Model,
        baseURL: cfg.BaseURL,
    }, nil
```

Any OpenAI-compatible API works out of the box via the `openai` provider with `--base-url`.

---

## Development

```bash
git clone https://github.com/yourhandle/suitest
cd suitest
go mod tidy
go run . run ./testdata/sample-project
```

### Run tests

```bash
go test ./...
```

### Build binary

```bash
go build -o suitest .

# Cross-compile
GOOS=darwin  GOARCH=arm64 go build -o dist/suitest-darwin-arm64 .
GOOS=linux   GOARCH=amd64 go build -o dist/suitest-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o dist/suitest-windows-amd64.exe .
```

### Build + release (GoReleaser)

```bash
goreleaser release --snapshot --clean
```

---

## AI agent guidance (for Claude Code, Cursor, etc.)

When working in this repo as an AI agent:

- **Provider interface is the contract** — never break `internal/providers/provider.go` interface without updating all implementations
- **Add tests for every new runner** — use `testdata/` fixtures, never real network calls in unit tests
- **Config changes need backward compat** — old `config.yaml` must still parse
- **MCP tools must match OpenAPI spec** — keep `plugin/openapi.yaml` and `mcp/tools.go` in sync
- **No hardcoded API keys ever** — all secrets via env vars or config file with `${}` interpolation
- **Prompts live in `internal/agent/prompt/`** — edit `.txt` files, not Go strings

---

## Roadmap

- [x] Architecture design
- [ ] Provider abstraction layer
- [ ] Go project runner (go test)
- [ ] Node.js runner (jest/vitest)
- [ ] Python runner (pytest)
- [ ] API runner (HTTP)
- [ ] Browser runner (chromedp)
- [ ] MCP server
- [ ] Claude plugin
- [ ] GitHub Actions integration
- [ ] Watch mode (`suitest watch`)
- [ ] PR comment reporter


<claude-mem-context>
# Memory Context

# [suitest] recent context, 2026-05-01 5:33pm GMT+7

Legend: 🎯session 🔴bugfix 🟣feature 🔄refactor ✅change 🔵discovery ⚖️decision
Format: ID TIME TYPE TITLE
Fetch details: get_observations([IDs]) | Search: mem-search skill

Stats: 39 obs (12,526t read) | 515,083t work | 98% savings

### May 1, 2026
614 4:23p 🔵 suitest — Full File Structure Confirmed
615 " 🔵 suitest — Core Agent Loop + Provider Architecture Fully Mapped
617 4:24p 🔵 suitest — Config System, Runner Implementations, MCP Server, and Report Layer Fully Mapped
622 4:41p ⚖️ suitest: UX Flow Architecture — New User vs Existing User Split
623 4:46p ⚖️ suitest: Implementation Plan — CLI Flow Refactor for UX Redesign
625 " 🔵 suitest: Current Config Defaults + CLI Capabilities Confirmed
629 4:47p 🔵 suitest: go.mod — No SQLite Driver, chromedp is Only Browser Dep
630 4:49p 🟣 suitest: catalog Package — Persisted Target + ScenarioSet Layer Implemented
631 " 🟣 suitest: Agent + Config Extended for Target-Aware Testing
632 " 🟣 suitest: New CLI Commands — target, scenario, settings + cmd/helpers.go
638 " 🟣 suitest: scenario + settings Commands Implemented
644 4:50p 🔄 suitest: cmd/run.go Refactored — Saved Target Routing + Helper Extraction
645 4:52p 🟣 suitest: runSavedTarget() Fully Implemented — Complete Existing-User Run Path
646 " 🔵 suitest: go test Fails with "operation not permitted" on Default GOCACHE
647 " ✅ suitest: README + init scaffold + config merge updated for new commands
657 4:53p ⚖️ suitest: Phase 2 Plan — Interactive Init Onboarding Wizard
659 4:56p 🟣 suitest: cmd/init.go Rewritten as Interactive 5-Step Onboarding Wizard
660 5:00p 🔴 suitest: cmd/init.go — fmt.Println → fmt.Print for Banner
661 " 🔵 suitest: Build Clean — Zero Test Files Across All Packages
663 5:01p ✅ suitest: Interactive Init Onboarding — All 5 Steps Complete
664 5:02p ⚖️ suitest: SQLite Storage Layer — New Implementation Plan Started
666 " 🔵 suitest: Persistence Layer — All File-Based via ~/.suitest/
668 5:03p 🔵 suitest: MCP Handlers — Direct Filesystem Access for Reports, No Catalog Integration
670 5:05p 🟣 suitest: internal/storage — Dual-Driver Storage Abstraction Layer Created
671 " 🔄 suitest: All Persistence Touchpoints Routed Through storage Package
674 " 🔴 suitest: Import Cycle — storage → agent → storage Fixed via LoadLastReportData()
677 5:06p 🔵 suitest: Build Clean Post-Refactor — Integration Test Blocked by No Network Access
680 " 🟣 suitest: SQLite Storage Verified End-to-End — Target Written and Confirmed in DB
682 " ✅ suitest: README + settings command Updated for SQLite Storage Config
684 5:10p 🟣 suitest: SQLite Storage Feature — All 5 Plan Steps Complete
685 5:11p ✅ suitest: Post-SQLite Polish Phase Started
688 " 🔄 suitest: cmd/run.go — Viper Bindings Removed, runSavedTarget + writeRunOutput Extracted
689 " ✅ suitest: .gitignore Updated — .gocache/ and .suitest/ Excluded
690 " 🔵 suitest: cmd/settings.go — Input Validation Confirmed for All Settable Keys
691 5:14p 🟣 suitest: Storage Validation Hardened — ValidateSQLiteConfig + expandAndPrepareDBPath Added
693 5:15p ✅ suitest: Polish + Hardening Phase Complete — All 4 Steps Done
694 5:21p ⚖️ suitest: Run History Feature — Implementation Plan Defined
695 5:22p 🔵 suitest: Current Report + Storage Architecture Confirmed for Run History Planning
697 " 🟣 suitest: Run History Model — RunID Field + SQLite runs Table Added

Access 515k tokens of past work via get_observations([IDs]) or mem-search skill.
</claude-mem-context>