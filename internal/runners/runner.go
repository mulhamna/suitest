package runners

import "context"

// RunResult holds the result of executing a single test.
type RunResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
	Runtime string `json:"runtime,omitempty"`
}

// Runner is the interface all test execution engines implement.
type Runner interface {
	// Name returns the runner identifier (e.g., "go", "jest", "pytest").
	Name() string

	// Run executes the tests at the given path and returns results.
	Run(ctx context.Context, path string, testCode string) ([]RunResult, error)

	// RunFile executes a specific test file and returns results.
	RunFile(ctx context.Context, path string) ([]RunResult, error)
}
