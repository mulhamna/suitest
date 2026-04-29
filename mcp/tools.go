package mcp

// ToolDefinition defines an MCP tool.
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// GetToolDefinitions returns all tool definitions exposed by suitest MCP.
func GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "suitest_run",
			Description: "Run AI-powered tests on a project path. Generates, executes, and debugs tests automatically.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Project root path to run tests on",
					},
					"mode": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"auto", "browser", "api", "unit"},
						"description": "Test mode",
						"default":     "auto",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI provider to use (claude, openai, openrouter, ollama)",
					},
					"fix": map[string]interface{}{
						"type":        "boolean",
						"description": "Auto-apply AI-suggested fixes to source files",
						"default":     false,
					},
					"dry_run": map[string]interface{}{
						"type":        "boolean",
						"description": "Show test plan only without executing",
						"default":     false,
					},
					"max_retries": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum fix retry attempts per test",
						"default":     3,
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "suitest_plan",
			Description: "Generate a test plan for a project without executing tests (dry run).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Project root path",
					},
					"mode": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"auto", "browser", "api", "unit"},
						"description": "Test mode",
						"default":     "auto",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "suitest_get_report",
			Description: "Get the latest test report from the most recent suitest run.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"format": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"json", "markdown", "terminal"},
						"description": "Report format",
						"default":     "json",
					},
				},
			},
		},
		{
			Name:        "suitest_fix",
			Description: "Apply an AI-generated fix to a failing test file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file": map[string]interface{}{
						"type":        "string",
						"description": "Path to the test file to fix",
					},
					"error": map[string]interface{}{
						"type":        "string",
						"description": "Error message or failure details",
					},
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "AI provider to use",
					},
				},
				"required": []string{"file", "error"},
			},
		},
		{
			Name:        "suitest_init",
			Description: "Initialize suitest in a project by creating .suitest.yaml.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Project root path",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}
