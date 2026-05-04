package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/internal/events"
	"github.com/mulhamna/suitest/internal/providers"
	"github.com/mulhamna/suitest/internal/runners"
)

// RunnerService executes suitest runs with optional structured event emission.
type RunnerService struct {
	cfg     Config
	emitter events.Emitter
	planner *Planner
	runner  runners.Runner
}

// NewRunnerService creates a reusable execution service for CLI and web callers.
func NewRunnerService(cfg Config, emitter events.Emitter) *RunnerService {
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 4
	}
	if emitter == nil {
		emitter = events.NopEmitter{}
	}

	return &RunnerService{
		cfg:     cfg,
		emitter: emitter,
		planner: NewPlanner(cfg.Provider),
	}
}

// Run executes the full agent loop and returns a RunResult.
func (s *RunnerService) Run(ctx context.Context) (*RunResult, error) {
	result := &RunResult{
		StartedAt: time.Now(),
		Path:      s.cfg.Path,
		Mode:      s.cfg.Mode,
		Provider:  s.cfg.Provider.Name(),
		DryRun:    s.cfg.DryRun,
	}

	s.emit("discovering", fmt.Sprintf("Discovering project at %s", s.cfg.Path), map[string]any{"path": s.cfg.Path})
	discovery, err := discoverProject(s.cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("project discovery failed: %w", err)
	}
	s.emit("project_discovered", discovery.Summary, map[string]any{
		"language": discovery.Language,
		"summary":  discovery.Summary,
		"files":    len(discovery.Files),
	})

	s.emit("planning", "Generating test plan via LLM", map[string]any{"mode": s.cfg.Mode})
	plans, err := s.planner.Plan(ctx, discovery, s.cfg.Mode)
	if err != nil {
		return nil, fmt.Errorf("test planning failed: %w", err)
	}
	s.emit("plan_ready", fmt.Sprintf("Generated %d test cases", len(plans)), map[string]any{"count": len(plans)})

	if s.cfg.DryRun {
		result.FinishedAt = time.Now()
		result.TotalTests = len(plans)
		s.emit("completed", "Dry run complete", map[string]any{"total_tests": len(plans), "dry_run": true})
		return result, nil
	}

	s.runner = buildRunner(s.cfg.Mode, s.cfg.Path)
	s.emit("executing", fmt.Sprintf("Executing %d tests", len(plans)), map[string]any{
		"count":       len(plans),
		"concurrency": s.cfg.Concurrency,
	})

	testResults := make([]TestResult, len(plans))
	sem := make(chan struct{}, s.cfg.Concurrency)
	var wg sync.WaitGroup

	for i, plan := range plans {
		wg.Add(1)
		go func(idx int, p TestPlan) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			s.emit("test_started", fmt.Sprintf("Starting %s", p.Name), map[string]any{
				"index": idx + 1,
				"total": len(plans),
				"name":  p.Name,
			})

			executor := NewExecutor(s.cfg.Provider, s.runner, s.cfg.MaxRetries, s.cfg.AutoFix)
			tr := executor.Execute(ctx, p, discovery)
			testResults[idx] = tr

			status := "failed"
			if tr.Passed {
				status = "passed"
			}
			s.emit("test_finished", fmt.Sprintf("%s %s", p.Name, status), map[string]any{
				"index":   idx + 1,
				"total":   len(plans),
				"name":    p.Name,
				"passed":  tr.Passed,
				"retries": tr.Retries,
			})
		}(i, plan)
	}

	wg.Wait()

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
	data, _ := json.Marshal(result)
	config.SaveReport(data)

	s.emit("completed", "Run finished", map[string]any{
		"total_tests": result.TotalTests,
		"passed":      result.Passed,
		"failed":      result.Failed,
		"duration_ms": result.FinishedAt.Sub(result.StartedAt).Milliseconds(),
	})

	return result, nil
}

func (s *RunnerService) emit(eventType, message string, data map[string]any) {
	s.emitter.Emit(events.Event{
		Time:    time.Now(),
		Type:    eventType,
		Message: message,
		Data:    data,
	})
}

// LoadProvider creates a provider from app config for shared callers.
func LoadProvider(cfg *config.Config) (providers.Provider, error) {
	return providers.New(cfg)
}

// ResolveMode auto-detects mode when configured as auto.
func ResolveMode(cfgMode, targetPath string) (string, error) {
	mode := cfgMode
	if mode == "" || mode == "auto" {
		absPath, err := filepath.Abs(targetPath)
		if err != nil {
			return "", err
		}
		return runners.Detect(absPath)
	}
	return mode, nil
}
