package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mulhamna/suitest/internal/agent"
	appconfig "github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/internal/events"
	"github.com/mulhamna/suitest/internal/jobs"
	"github.com/mulhamna/suitest/internal/runners"
)

// Server provides the minimal local web API surface.
type Server struct {
	jobs *jobs.Manager
}

// NewServer creates a new API server.
func NewServer(jobManager *jobs.Manager) *Server {
	return &Server{jobs: jobManager}
}

// Handler returns the HTTP mux for the local API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/runs/", s.handleRunByID)
	mux.HandleFunc("/api/projects/detect", s.handleProjectDetect)
	return mux
}

type runEmitter struct {
	runID string
	jobs  *jobs.Manager
}

func (e runEmitter) Emit(event events.Event) {
	if status, ok := event.Data["status"].(string); ok && status != "" {
		e.jobs.UpdateStatus(e.runID, jobs.Status(status))
	}
	e.jobs.AppendEvent(e.runID, event)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "suitest-web-api",
	})
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"runs": s.jobs.List()})
	case http.MethodPost:
		var req struct {
			ProjectPath string `json:"project_path"`
			Mode        string `json:"mode"`
			Provider    string `json:"provider"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json body"})
			return
		}
		if req.ProjectPath == "" {
			req.ProjectPath = "."
		}
		projectPath, err := filepath.Abs(req.ProjectPath)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid project path"})
			return
		}
		if _, err := os.Stat(projectPath); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "project path does not exist"})
			return
		}
		if req.Mode == "" {
			req.Mode = "auto"
		}
		if req.Provider == "" {
			req.Provider = "auto"
		}
		run := s.jobs.Create(projectPath, req.Mode, req.Provider)
		go s.executeRun(context.Background(), run.ID, projectPath, req.Mode, req.Provider)
		writeJSON(w, http.StatusAccepted, run)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRunByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	runID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		run, ok := s.jobs.Get(runID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "run not found"})
			return
		}
		writeJSON(w, http.StatusOK, run)
		return
	}

	if len(parts) == 2 && parts[1] == "events" {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleRunEvents(w, r, runID)
		return
	}

	http.NotFound(w, r)
}

func (s *Server) handleRunEvents(w http.ResponseWriter, r *http.Request, runID string) {
	run, ok := s.jobs.Get(runID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "run not found"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, event := range run.Events {
		writeSSE(w, "run_event", event)
	}
	flusher.Flush()

	ch, cancel := s.jobs.Subscribe(runID)
	defer cancel()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, "run_event", event)
			flusher.Flush()
		}
	}
}

func (s *Server) handleProjectDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectPath := r.URL.Query().Get("path")
	if projectPath == "" {
		projectPath = "."
	}

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid project path"})
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "project directory not found"})
		return
	}

	mode, err := runners.Detect(absPath)
	if err != nil {
		mode = "auto"
	}

	signals := detectSignals(absPath)
	writeJSON(w, http.StatusOK, map[string]any{
		"path":             absPath,
		"name":             filepath.Base(absPath),
		"detected_mode":    mode,
		"detected_signals": signals,
	})
}

func (s *Server) executeRun(ctx context.Context, runID, projectPath, requestedMode, requestedProvider string) {
	cfg, err := appconfig.Load()
	if err != nil {
		s.jobs.UpdateStatus(runID, jobs.StatusFailed)
		s.jobs.AppendEvent(runID, events.Event{Time: time.Now(), Type: "error", Message: fmt.Sprintf("Failed to load config: %v", err), Data: map[string]any{"status": string(jobs.StatusFailed)}})
		return
	}

	if requestedProvider != "" && requestedProvider != "auto" {
		cfg.DefaultProvider = requestedProvider
	}

	provider, err := agent.LoadProvider(cfg)
	if err != nil {
		s.jobs.UpdateStatus(runID, jobs.StatusFailed)
		s.jobs.AppendEvent(runID, events.Event{Time: time.Now(), Type: "error", Message: fmt.Sprintf("Failed to initialize provider: %v", err), Data: map[string]any{"status": string(jobs.StatusFailed)}})
		return
	}

	mode, err := agent.ResolveMode(requestedMode, projectPath)
	if err != nil {
		s.jobs.UpdateStatus(runID, jobs.StatusFailed)
		s.jobs.AppendEvent(runID, events.Event{Time: time.Now(), Type: "error", Message: fmt.Sprintf("Failed to resolve mode: %v", err), Data: map[string]any{"status": string(jobs.StatusFailed)}})
		return
	}

	s.jobs.AppendEvent(runID, events.Event{
		Time:    time.Now(),
		Type:    "run_queued",
		Message: fmt.Sprintf("Queued run for %s", projectPath),
		Data: map[string]any{
			"status":   string(jobs.StatusQueued),
			"mode":     mode,
			"provider": provider.Name(),
		},
	})

	service := agent.NewRunnerService(agent.Config{
		Provider:    provider,
		Path:        projectPath,
		Mode:        mode,
		MaxRetries:  cfg.Agent.MaxRetries,
		Concurrency: cfg.Agent.Concurrency,
		AutoFix:     cfg.Agent.AutoFix,
	}, runEmitter{runID: runID, jobs: s.jobs})

	if _, err := service.Run(ctx); err != nil {
		s.jobs.UpdateStatus(runID, jobs.StatusFailed)
		s.jobs.AppendEvent(runID, events.Event{
			Time:    time.Now(),
			Type:    "error",
			Message: fmt.Sprintf("Run failed: %v", err),
			Data:    map[string]any{"status": string(jobs.StatusFailed)},
		})
	}
}

func detectSignals(root string) []string {
	signals := make([]string, 0, 6)
	checks := []struct {
		file  string
		label string
	}{
		{"go.mod", "go"},
		{"package.json", "node"},
		{"requirements.txt", "python"},
		{"pyproject.toml", "python"},
		{"playwright.config.ts", "playwright"},
		{".suitest.yaml", "suitest-config"},
	}

	for _, check := range checks {
		if _, err := os.Stat(filepath.Join(root, check.file)); err == nil {
			signals = append(signals, check.label)
		}
	}

	entries, err := os.ReadDir(root)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if entry.IsDir() && (name == "cmd" || name == "src" || name == "tests") {
				signals = append(signals, name)
			}
		}
	}

	sort.Strings(signals)
	return compact(signals)
}

func compact(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSSE(w http.ResponseWriter, eventType string, payload any) {
	data, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(w, "event: %s\n", eventType)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
}
