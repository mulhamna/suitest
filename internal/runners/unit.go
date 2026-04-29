package runners

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// UnitRunner runs unit tests using the detected language toolchain.
type UnitRunner struct {
	subtype string // "go", "jest", "vitest", "pytest"
	root    string
}

// NewUnitRunner creates a unit test runner for the given project root.
func NewUnitRunner(root string) *UnitRunner {
	return &UnitRunner{
		subtype: DetectSubtype(root),
		root:    root,
	}
}

func (r *UnitRunner) Name() string {
	return "unit/" + r.subtype
}

func (r *UnitRunner) Run(ctx context.Context, path string, testCode string) ([]RunResult, error) {
	switch r.subtype {
	case "go":
		return r.runGo(ctx, path, testCode)
	case "jest":
		return r.runJest(ctx, path, testCode)
	case "vitest":
		return r.runVitest(ctx, path, testCode)
	case "pytest":
		return r.runPytest(ctx, path, testCode)
	default:
		return r.runGo(ctx, path, testCode)
	}
}

func (r *UnitRunner) RunFile(ctx context.Context, path string) ([]RunResult, error) {
	return r.Run(ctx, path, "")
}

// runGo runs go test on the given path.
func (r *UnitRunner) runGo(ctx context.Context, path string, testCode string) ([]RunResult, error) {
	// If testCode is provided, write it to a temp file
	if testCode != "" {
		tmpFile := filepath.Join(path, "suitest_generated_test.go")
		if err := os.WriteFile(tmpFile, []byte(testCode), 0644); err != nil {
			return nil, fmt.Errorf("write test file: %w", err)
		}
		defer os.Remove(tmpFile)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-v", "-json", "./...")
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	elapsed := time.Since(start)

	results := parseGoTestJSON(stdout.String(), elapsed)

	if runErr != nil && len(results) == 0 {
		return []RunResult{{
			Name:    "go test",
			Passed:  false,
			Output:  stdout.String(),
			Error:   stderr.String(),
			Runtime: elapsed.String(),
		}}, nil
	}

	return results, nil
}

type goTestEvent struct {
	Action  string  `json:"Action"`
	Test    string  `json:"Test"`
	Package string  `json:"Package"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

func parseGoTestJSON(output string, totalElapsed time.Duration) []RunResult {
	testOutputs := make(map[string][]string)
	testResults := make(map[string]bool)
	var order []string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Test == "" {
			continue
		}
		key := event.Package + "/" + event.Test
		switch event.Action {
		case "run":
			if _, exists := testResults[key]; !exists {
				order = append(order, key)
			}
		case "output":
			testOutputs[key] = append(testOutputs[key], event.Output)
		case "pass":
			testResults[key] = true
		case "fail":
			testResults[key] = false
		}
	}

	var results []RunResult
	for _, key := range order {
		passed, exists := testResults[key]
		if !exists {
			continue
		}
		parts := strings.SplitN(key, "/", 2)
		name := key
		if len(parts) == 2 {
			name = parts[1]
		}
		outputLines := strings.Join(testOutputs[key], "")
		results = append(results, RunResult{
			Name:    name,
			Passed:  passed,
			Output:  outputLines,
			Runtime: totalElapsed.String(),
		})
	}

	return results
}

// runJest runs jest on the given path.
func (r *UnitRunner) runJest(ctx context.Context, path string, testCode string) ([]RunResult, error) {
	if testCode != "" {
		tmpFile := filepath.Join(path, "suitest.generated.test.js")
		if err := os.WriteFile(tmpFile, []byte(testCode), 0644); err != nil {
			return nil, fmt.Errorf("write test file: %w", err)
		}
		defer os.Remove(tmpFile)
	}

	start := time.Now()

	// Try npx jest, then jest directly
	var cmd *exec.Cmd
	if _, err := exec.LookPath("npx"); err == nil {
		cmd = exec.CommandContext(ctx, "npx", "jest", "--json", "--no-coverage")
	} else {
		cmd = exec.CommandContext(ctx, "jest", "--json", "--no-coverage")
	}
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Run()
	elapsed := time.Since(start)

	return parseJestJSON(stdout.String(), elapsed), nil
}

type jestTestResult struct {
	TestResults []struct {
		TestFilePath string `json:"testFilePath"`
		Status       string `json:"status"`
		AssertionResults []struct {
			Title    string   `json:"title"`
			Status   string   `json:"status"`
			FailureMessages []string `json:"failureMessages"`
		} `json:"assertionResults"`
	} `json:"testResults"`
}

func parseJestJSON(output, _ interface{}) []RunResult {
	// output is string, elapsed is time.Duration
	outStr, ok := output.(string)
	if !ok {
		return nil
	}

	var result jestTestResult
	if err := json.Unmarshal([]byte(outStr), &result); err != nil {
		return []RunResult{{
			Name:   "jest",
			Passed: false,
			Output: outStr,
		}}
	}

	var results []RunResult
	for _, tr := range result.TestResults {
		for _, ar := range tr.AssertionResults {
			r := RunResult{
				Name:   ar.Title,
				Passed: ar.Status == "passed",
				Output: strings.Join(ar.FailureMessages, "\n"),
			}
			results = append(results, r)
		}
	}
	return results
}

// runVitest runs vitest on the given path.
func (r *UnitRunner) runVitest(ctx context.Context, path string, testCode string) ([]RunResult, error) {
	if testCode != "" {
		tmpFile := filepath.Join(path, "suitest.generated.test.ts")
		if err := os.WriteFile(tmpFile, []byte(testCode), 0644); err != nil {
			return nil, fmt.Errorf("write test file: %w", err)
		}
		defer os.Remove(tmpFile)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "npx", "vitest", "run", "--reporter=json")
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Run()
	elapsed := time.Since(start)

	return parseJestJSON(stdout.String(), elapsed), nil
}

// runPytest runs pytest on the given path.
func (r *UnitRunner) runPytest(ctx context.Context, path string, testCode string) ([]RunResult, error) {
	if testCode != "" {
		tmpFile := filepath.Join(path, "test_suitest_generated.py")
		if err := os.WriteFile(tmpFile, []byte(testCode), 0644); err != nil {
			return nil, fmt.Errorf("write test file: %w", err)
		}
		defer os.Remove(tmpFile)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "python", "-m", "pytest", "-v", "--tb=short", "--json-report", "--json-report-file=-")
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Run()
	elapsed := time.Since(start)

	return parsePytestJSON(stdout.String(), stderr.String(), elapsed), nil
}

type pytestReport struct {
	Tests []struct {
		NodeID   string `json:"nodeid"`
		Outcome  string `json:"outcome"`
		Longrepr string `json:"longrepr,omitempty"`
	} `json:"tests"`
}

func parsePytestJSON(stdout, stderr string, elapsed time.Duration) []RunResult {
	var report pytestReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		// Fallback: parse plain text output
		return []RunResult{{
			Name:    "pytest",
			Passed:  !strings.Contains(stdout+stderr, "FAILED") && !strings.Contains(stdout+stderr, "ERROR"),
			Output:  stdout,
			Error:   stderr,
			Runtime: elapsed.String(),
		}}
	}

	var results []RunResult
	for _, t := range report.Tests {
		results = append(results, RunResult{
			Name:    t.NodeID,
			Passed:  t.Outcome == "passed",
			Output:  t.Longrepr,
			Runtime: elapsed.String(),
		})
	}
	return results
}
