# Claude Plugin

suitest can be used as a Claude plugin, exposing test-running capabilities directly inside Claude.ai conversations.

## Requirements

- suitest binary installed and in PATH
- MCP server running locally (`suitest serve`)

## Setup

1. Start the suitest server:

```bash
suitest serve --port 3100
```

2. The plugin manifest is available at:

```
http://localhost:3100/manifest.json
```

3. The OpenAPI spec is at:

```
http://localhost:3100/openapi.yaml
```

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/run` | Run tests on a project path |
| `POST` | `/plan` | Generate test plan (dry run) |
| `GET` | `/report` | Get latest test report |
| `POST` | `/fix` | Apply AI fix to a failing test file |
| `POST` | `/init` | Initialize suitest in a project |

## Example: Run tests via plugin

```json
POST /run
{
  "path": "/home/user/myproject",
  "mode": "auto",
  "provider": "claude",
  "fix": false
}
```

Response:

```json
{
  "total_tests": 5,
  "passed": 4,
  "failed": 1,
  "duration_ms": 12340,
  "results": [...]
}
```

## Example: Get report as Markdown

```
GET /report?format=markdown
```
