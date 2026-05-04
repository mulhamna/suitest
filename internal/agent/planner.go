package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/mulhamna/suitest/internal/providers"
)

// TestPlan describes a single test case to be generated and executed.
type TestPlan struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	Target      string `json:"target,omitempty"` // file or URL to test
	TestCode    string `json:"test_code,omitempty"`
}

// Planner generates test plans from project discovery using an LLM.
type Planner struct {
	provider providers.Provider
}

// NewPlanner creates a Planner.
func NewPlanner(p providers.Provider) *Planner {
	return &Planner{provider: p}
}

// Plan generates a list of test plans for the discovered project.
func (p *Planner) Plan(ctx context.Context, discovery *ProjectDiscovery, mode string) ([]TestPlan, error) {
	prompt, err := buildPlanPrompt(discovery, mode)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	messages := []providers.Message{
		{
			Role:    providers.RoleSystem,
			Content: "You are an expert software testing agent. Your job is to analyze codebases and generate comprehensive test plans. Always respond with valid JSON.",
		},
		{
			Role:    providers.RoleUser,
			Content: prompt,
		},
	}

	response, err := p.provider.Complete(ctx, messages, providers.CompleteOptions{
		MaxTokens:   4096,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	plans, err := parsePlanResponse(response, mode)
	if err != nil {
		// Return a minimal fallback plan
		return fallbackPlan(discovery, mode), nil
	}

	return plans, nil
}

func buildPlanPrompt(discovery *ProjectDiscovery, mode string) (string, error) {
	tmplContent, err := loadPromptTemplate("plan.txt")
	if err != nil {
		return buildFallbackPlanPrompt(discovery, mode), nil
	}

	tmpl, err := template.New("plan").Parse(tmplContent)
	if err != nil {
		return buildFallbackPlanPrompt(discovery, mode), nil
	}

	data := map[string]interface{}{
		"Language":    discovery.Language,
		"Mode":        mode,
		"Files":       strings.Join(discovery.Files, "\n"),
		"Summary":     discovery.Summary,
		"FileCount":   len(discovery.Files),
		"EntryURL":    discovery.EntryURL,
		"SeedCurl":    discovery.SeedCurl,
		"TargetType":  discovery.TargetType,
		"Expectation": discovery.Expectation,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return buildFallbackPlanPrompt(discovery, mode), nil
	}

	return buf.String(), nil
}

func buildFallbackPlanPrompt(discovery *ProjectDiscovery, mode string) string {
	fileList := strings.Join(discovery.Files, "\n")
	return fmt.Sprintf(`You are analyzing a %s project to generate a test plan.

Project summary: %s
Test mode: %s
Target type: %s
Entry URL: %s
Sample curl: %s
Expected success flow: %s
Source files:
%s

Generate a comprehensive test plan as a JSON array. Each test case should have:
- name: short descriptive name
- description: what this test verifies
- mode: "%s"
- target: the file or function to test

Respond ONLY with a JSON array of test cases. Example:
[
  {
    "name": "TestUserCreation",
    "description": "Verify that a user can be created with valid input",
    "mode": "%s",
    "target": "user.go"
  }
]

Generate 3-8 test cases appropriate for this project.`, discovery.Language, discovery.Summary, mode, discovery.TargetType, discovery.EntryURL, discovery.SeedCurl, discovery.Expectation, fileList, mode, mode)
}

func parsePlanResponse(response, mode string) ([]TestPlan, error) {
	// Extract JSON from response (it might be wrapped in markdown code blocks)
	jsonStr := extractJSON(response)

	var plans []TestPlan
	if err := json.Unmarshal([]byte(jsonStr), &plans); err != nil {
		return nil, fmt.Errorf("parse JSON plans: %w", err)
	}

	// Set mode if not set
	for i := range plans {
		if plans[i].Mode == "" {
			plans[i].Mode = mode
		}
	}

	return plans, nil
}

func fallbackPlan(discovery *ProjectDiscovery, mode string) []TestPlan {
	return []TestPlan{
		{
			Name:        "BasicFunctionalityTest",
			Description: "Verify basic functionality of the project",
			Mode:        mode,
			Target:      discovery.Path,
		},
	}
}

// extractJSON finds and returns the first JSON array or object in a string.
func extractJSON(s string) string {
	// Strip markdown code blocks
	s = strings.ReplaceAll(s, "```json", "")
	s = strings.ReplaceAll(s, "```", "")
	s = strings.TrimSpace(s)

	// Find the first [ or {
	start := -1
	for i, c := range s {
		if c == '[' || c == '{' {
			start = i
			break
		}
	}
	if start == -1 {
		return s
	}

	// Find matching close bracket
	opener := rune(s[start])
	var closer rune
	if opener == '[' {
		closer = ']'
	} else {
		closer = '}'
	}

	depth := 0
	inStr := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := rune(s[i])
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inStr {
			escaped = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		if c == opener {
			depth++
		} else if c == closer {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return s[start:]
}

// loadPromptTemplate loads a prompt template from the embedded prompt directory.
func loadPromptTemplate(name string) (string, error) {
	// Find the prompt directory relative to this file at runtime
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not determine source file path")
	}
	dir := filepath.Dir(filename)
	promptPath := filepath.Join(dir, "prompt", name)

	data, err := os.ReadFile(promptPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
