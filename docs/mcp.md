# MCP Server

suitest exposes an MCP (Model Context Protocol) server for IDE integration with Claude Code, Cursor, Windsurf, and others.

## Start the server

```bash
suitest serve
suitest serve --port 3100
```

## Configure in Claude Code

Add to your MCP config (`~/.claude/mcp.json` or project `.claude/mcp.json`):

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

Or with a custom port:

```json
{
  "mcpServers": {
    "suitest": {
      "command": "suitest",
      "args": ["serve", "--port", "3100"]
    }
  }
}
```

## Available tools

| Tool | Description |
|---|---|
| `suitest_run` | Run tests on a given path |
| `suitest_plan` | Generate test plan (dry run) |
| `suitest_get_report` | Get the latest test report |
| `suitest_fix` | Apply AI fix to a failing test |
| `suitest_init` | Initialize suitest in a project |

## Tool: suitest_run

```json
{
  "path": "./",
  "mode": "auto",
  "provider": "claude",
  "fix": false,
  "dry_run": false,
  "max_retries": 3,
  "concurrency": 4
}
```

## Tool: suitest_plan

```json
{
  "path": "./",
  "provider": "openrouter",
  "model": "mistral/mistral-7b-instruct"
}
```

## Tool: suitest_get_report

No parameters — returns the last run result as JSON.

## Tool: suitest_fix

```json
{
  "file": "./tests/api_test.go",
  "error": "FAIL: TestGetUser — expected 200, got 404",
  "provider": "claude"
}
```

## Tool: suitest_init

```json
{
  "path": "./"
}
```
