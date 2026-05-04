package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mulhamna/suitest/internal/providers"
	"github.com/mulhamna/suitest/internal/runners"
)

// Config holds agent configuration.
type Config struct {
	Provider    providers.Provider
	Path        string
	Mode        string
	MaxRetries  int
	Concurrency int
	AutoFix     bool
	DryRun      bool
	TargetName  string
	TargetType  string
	EntryURL    string
	SeedCurl    string
	Expectation string
	Plans       []TestPlan
}

// TestResult holds the result for a single planned test.
type TestResult struct {
	Plan    TestPlan            `json:"plan"`
	Results []runners.RunResult `json:"results"`
	Passed  bool                `json:"passed"`
	Retries int                 `json:"retries"`
}

// RunResult is the overall result of an agent run.
type RunResult struct {
	RunID      string       `json:"run_id"`
	StartedAt  time.Time    `json:"started_at"`
	FinishedAt time.Time    `json:"finished_at"`
	Path       string       `json:"path"`
	Mode       string       `json:"mode"`
	Provider   string       `json:"provider"`
	Tests      []TestResult `json:"tests"`
	TotalTests int          `json:"total_tests"`
	Passed     int          `json:"passed"`
	Failed     int          `json:"failed"`
	DryRun     bool         `json:"dry_run"`
}

// Agent orchestrates the full test generation and execution loop.
type Agent struct {
	service *RunnerService
}

// New creates a new Agent.
func New(cfg Config) *Agent {
	return &Agent{service: NewRunnerService(cfg, nil)}
}

// Run executes the full agent loop and returns a RunResult.
func (a *Agent) Run(ctx context.Context) (*RunResult, error) {
	return a.service.Run(ctx)
}

// buildRunner creates the appropriate runner based on mode.
func buildRunner(mode, path string) runners.Runner {
	switch mode {
	case "browser":
		return runners.NewBrowserRunner(true)
	case "api":
		return runners.NewAPIRunner()
	default:
		return runners.NewUnitRunner(path)
	}
}

// ProjectDiscovery holds information about the discovered project.
type ProjectDiscovery struct {
	Path        string
	Language    string
	Framework   string
	Files       []string
	Summary     string
	TargetName  string
	TargetType  string
	EntryURL    string
	SeedCurl    string
	Expectation string
}

// DiscoverProject walks the project directory to understand its structure.
func DiscoverProject(root string) (*ProjectDiscovery, error) {
	d := &ProjectDiscovery{Path: root}

	// Walk directory for key files
	var sourceFiles []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == "__pycache__" || name == ".suitest" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(info.Name())
		switch ext {
		case ".go", ".js", ".ts", ".py", ".jsx", ".tsx":
			rel, _ := filepath.Rel(root, path)
			sourceFiles = append(sourceFiles, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Limit to first 50 files for context
	if len(sourceFiles) > 50 {
		sourceFiles = sourceFiles[:50]
	}
	d.Files = sourceFiles

	// Detect language
	if fileExistsAt(filepath.Join(root, "go.mod")) {
		d.Language = "Go"
	} else if fileExistsAt(filepath.Join(root, "package.json")) {
		d.Language = "JavaScript/TypeScript"
	} else if fileExistsAt(filepath.Join(root, "requirements.txt")) ||
		fileExistsAt(filepath.Join(root, "pyproject.toml")) {
		d.Language = "Python"
	} else {
		d.Language = "Unknown"
	}

	d.Summary = fmt.Sprintf("%s project with %d source files", d.Language, len(sourceFiles))
	return d, nil
}

func fileExistsAt(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
