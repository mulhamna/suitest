package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/mulhamna/suitest/internal/providers"
	"github.com/mulhamna/suitest/internal/runners"
)

// Executor handles the generate → execute → fix loop for a single test.
type Executor struct {
	provider   providers.Provider
	runner     runners.Runner
	maxRetries int
	autoFix    bool
}

// NewExecutor creates an Executor.
func NewExecutor(p providers.Provider, r runners.Runner, maxRetries int, autoFix bool) *Executor {
	return &Executor{
		provider:   p,
		runner:     r,
		maxRetries: maxRetries,
		autoFix:    autoFix,
	}
}

// Execute runs the generate → execute → fix loop for a TestPlan.
func (e *Executor) Execute(ctx context.Context, plan TestPlan, discovery *ProjectDiscovery) TestResult {
	tr := TestResult{Plan: plan}

	// Step 1: Generate test code
	testCode, err := e.generateTestCode(ctx, plan, discovery)
	if err != nil {
		tr.Results = []runners.RunResult{{
			Name:   plan.Name,
			Passed: false,
			Error:  fmt.Sprintf("test generation failed: %v", err),
		}}
		return tr
	}
	plan.TestCode = testCode

	// Step 2: Execute and optionally fix
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		tr.Retries = attempt

		runResults, err := e.runner.Run(ctx, plan.Target, plan.TestCode)
		if err != nil {
			tr.Results = []runners.RunResult{{
				Name:   plan.Name,
				Passed: false,
				Error:  fmt.Sprintf("runner error: %v", err),
			}}
			if attempt < e.maxRetries {
				// Try to fix the runner error
				fixed, fixErr := e.fixTestCode(ctx, plan, discovery, err.Error())
				if fixErr == nil {
					plan.TestCode = fixed
					continue
				}
			}
			break
		}

		tr.Results = runResults
		allPassed := true
		for _, r := range runResults {
			if !r.Passed {
				allPassed = false
				break
			}
		}

		if allPassed || attempt == e.maxRetries {
			tr.Passed = allPassed
			break
		}

		// Collect failures for the fix prompt
		var failureDetails strings.Builder
		for _, r := range runResults {
			if !r.Passed {
				failureDetails.WriteString(fmt.Sprintf("Test: %s\nError: %s\nOutput: %s\n\n", r.Name, r.Error, r.Output))
			}
		}

		fmt.Printf("  [%s] attempt %d failed, requesting fix...\n", plan.Name, attempt+1)
		fixed, fixErr := e.fixTestCode(ctx, plan, discovery, failureDetails.String())
		if fixErr != nil {
			fmt.Printf("  [%s] fix request failed: %v\n", plan.Name, fixErr)
			tr.Passed = false
			break
		}
		plan.TestCode = fixed
	}

	return tr
}

// generateTestCode asks the LLM to write test code for the plan.
func (e *Executor) generateTestCode(ctx context.Context, plan TestPlan, discovery *ProjectDiscovery) (string, error) {
	prompt := fmt.Sprintf(`Generate test code for the following test case.

Test name: %s
Description: %s
Mode: %s
Project language: %s
Target: %s

Project files:
%s

Requirements:
- Write complete, runnable test code
- Use the appropriate testing framework for %s
- For Go: use the testing package and write valid *_test.go file
- For JS/TS: write jest/vitest compatible tests
- For Python: write pytest compatible tests
- For API: return JSON array of APITestCase objects with name, method, url, expect fields
- For Browser: return JSON array of BrowserTestCase objects with name, url, actions fields
- Make tests that actually verify meaningful behavior
- Include proper imports and package declarations

Respond with ONLY the test code (no explanation, no markdown fences).`,
		plan.Name, plan.Description, plan.Mode, discovery.Language, plan.Target,
		strings.Join(discovery.Files, "\n"), discovery.Language)

	messages := []providers.Message{
		{
			Role:    providers.RoleSystem,
			Content: "You are an expert software testing agent. Generate complete, runnable test code. Respond with code only.",
		},
		{
			Role:    providers.RoleUser,
			Content: prompt,
		},
	}

	response, err := e.provider.Complete(ctx, messages, providers.CompleteOptions{
		MaxTokens:   4096,
		Temperature: 0.1,
	})
	if err != nil {
		return "", err
	}

	// Strip markdown fences if present
	code := stripMarkdownFences(response)
	return code, nil
}

// fixTestCode asks the LLM to fix failing test code.
func (e *Executor) fixTestCode(ctx context.Context, plan TestPlan, discovery *ProjectDiscovery, failures string) (string, error) {
	prompt := fmt.Sprintf(`Fix the following test code that has failures.

Test name: %s
Mode: %s
Project language: %s

Current test code:
%s

Failure details:
%s

Fix the test code to resolve all failures.
Respond with ONLY the fixed test code (no explanation, no markdown fences).`,
		plan.Name, plan.Mode, discovery.Language, plan.TestCode, failures)

	messages := []providers.Message{
		{
			Role:    providers.RoleSystem,
			Content: "You are an expert software testing agent. Fix failing test code. Respond with code only.",
		},
		{
			Role:    providers.RoleUser,
			Content: prompt,
		},
	}

	response, err := e.provider.Complete(ctx, messages, providers.CompleteOptions{
		MaxTokens:   4096,
		Temperature: 0.1,
	})
	if err != nil {
		return "", err
	}

	return stripMarkdownFences(response), nil
}

func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}

	// Remove opening fence
	if strings.HasPrefix(lines[0], "```") {
		lines = lines[1:]
	}

	// Remove closing fence
	if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}
