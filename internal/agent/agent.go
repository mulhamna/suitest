package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mulhamna/suitest/internal/providers"
	"github.com/mulhamna/suitest/internal/runners"
	"github.com/mulhamna/suitest/internal/storage"
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
	cfg     Config
	planner *Planner
	runner  runners.Runner
}

// New creates a new Agent.
func New(cfg Config) *Agent {
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 4
	}

	return &Agent{
		cfg:     cfg,
		planner: NewPlanner(cfg.Provider),
	}
}

// Run executes the full agent loop and returns a RunResult.
func (a *Agent) Run(ctx context.Context) (*RunResult, error) {
	result := &RunResult{
		RunID:     time.Now().Format("20060102-150405"),
		StartedAt: time.Now(),
		Path:      a.cfg.Path,
		Mode:      a.cfg.Mode,
		Provider:  a.cfg.Provider.Name(),
		DryRun:    a.cfg.DryRun,
	}

	// Step 1: Discover project
	fmt.Printf("\nDiscovering project at %s...\n", a.cfg.Path)
	discovery, err := DiscoverProject(a.cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("project discovery failed: %w", err)
	}
	discovery.TargetName = a.cfg.TargetName
	discovery.TargetType = a.cfg.TargetType
	discovery.EntryURL = a.cfg.EntryURL
	discovery.SeedCurl = a.cfg.SeedCurl
	discovery.Expectation = a.cfg.Expectation
	if discovery.Expectation != "" {
		discovery.Summary = fmt.Sprintf("%s | expected flow: %s", discovery.Summary, discovery.Expectation)
	}
	if discovery.EntryURL != "" {
		discovery.Summary = fmt.Sprintf("%s | url: %s", discovery.Summary, discovery.EntryURL)
	}
	fmt.Printf("Project summary: %s\n", discovery.Summary)

	// Step 2: Generate test plan
	plans := a.cfg.Plans
	if len(plans) == 0 {
		fmt.Println("\nGenerating test plan via LLM...")
		plans, err = a.planner.Plan(ctx, discovery, a.cfg.Mode)
		if err != nil {
			return nil, fmt.Errorf("test planning failed: %w", err)
		}
	} else {
		fmt.Println("\nUsing saved scenario set...")
	}
	fmt.Printf("Generated %d test cases\n", len(plans))

	if a.cfg.DryRun {
		fmt.Println("\nDry run — test plan:")
		for i, p := range plans {
			fmt.Printf("  %d. %s\n", i+1, p.Name)
			fmt.Printf("     %s\n", p.Description)
		}
		result.FinishedAt = time.Now()
		result.TotalTests = len(plans)
		return result, nil
	}

	// Step 3: Initialize runner
	a.runner = buildRunner(a.cfg.Mode, a.cfg.Path)

	// Step 4: Execute tests concurrently
	fmt.Printf("\nExecuting %d tests (concurrency: %d)...\n", len(plans), a.cfg.Concurrency)

	testResults := make([]TestResult, len(plans))
	sem := make(chan struct{}, a.cfg.Concurrency)
	var wg sync.WaitGroup

	for i, plan := range plans {
		wg.Add(1)
		go func(idx int, p TestPlan) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			executor := NewExecutor(a.cfg.Provider, a.runner, a.cfg.MaxRetries, a.cfg.AutoFix)
			tr := executor.Execute(ctx, p, discovery)
			testResults[idx] = tr
		}(i, plan)
	}

	wg.Wait()

	// Aggregate results
	result.Tests = testResults
	result.TotalTests = len(testResults)
	for _, tr := range testResults {
		if tr.Passed {
			result.Passed++
		} else {
			result.Failed++
		}
	}

	result.FinishedAt = time.Now()

	// Save report
	data, _ := json.Marshal(result)
	storage.SaveReport(data)

	return result, nil
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
