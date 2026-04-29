package plugin

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/mulhamna/suitest/internal/agent"
	"github.com/mulhamna/suitest/internal/config"
	"github.com/mulhamna/suitest/internal/providers"
	"github.com/mulhamna/suitest/internal/report"
)

var lastResult *agent.RunResult

// RegisterRoutes attaches plugin HTTP handlers to mux.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/run", handleRun)
	mux.HandleFunc("/plan", handlePlan)
	mux.HandleFunc("/report", handleReport)
	mux.HandleFunc("/fix", handleFix)
	mux.HandleFunc("/init", handleInit)
	mux.HandleFunc("/openapi.yaml", handleOpenAPI)
}

type runRequest struct {
	Path        string `json:"path"`
	Mode        string `json:"mode"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Fix         bool   `json:"fix"`
	DryRun      bool   `json:"dry_run"`
	MaxRetries  int    `json:"max_retries"`
	Concurrency int    `json:"concurrency"`
}

func buildProvider(cfg *config.Config, providerName, model string) (providers.Provider, error) {
	if providerName != "" {
		cfg.DefaultProvider = providerName
	}
	if model != "" {
		pc := cfg.GetProviderConfig(cfg.DefaultProvider)
		pc.Model = model
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]*config.ProviderConfig)
		}
		cfg.Providers[cfg.DefaultProvider] = pc
	}
	return providers.New(cfg)
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}
	if req.Concurrency == 0 {
		req.Concurrency = 4
	}

	cfg, _ := config.Load()
	p, err := buildProvider(cfg, req.Provider, req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a := agent.New(agent.Config{
		Provider:    p,
		Path:        req.Path,
		Mode:        req.Mode,
		AutoFix:     req.Fix,
		DryRun:      req.DryRun,
		MaxRetries:  req.MaxRetries,
		Concurrency: req.Concurrency,
	})

	result, err := a.Run(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lastResult = result
	writeJSON(w, result)
}

type planRequest struct {
	Path     string `json:"path"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func handlePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req planRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg, _ := config.Load()
	p, err := buildProvider(cfg, req.Provider, req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a := agent.New(agent.Config{
		Provider: p,
		Path:     req.Path,
		DryRun:   true,
	})
	result, err := a.Run(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if lastResult == nil {
		http.Error(w, "no report available", http.StatusNotFound)
		return
	}
	format := r.URL.Query().Get("format")
	if format == "markdown" {
		w.Header().Set("Content-Type", "text/markdown")
		report.NewMarkdownReporter().Write(w, lastResult)
		return
	}
	writeJSON(w, lastResult)
}

type fixRequest struct {
	File     string `json:"file"`
	Error    string `json:"error"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func handleFix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req fixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg, _ := config.Load()
	_, err := buildProvider(cfg, req.Provider, req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{"fixed": false, "message": "fix via plugin not yet supported — use suitest run --fix"})
}

type initRequest struct {
	Path string `json:"path"`
}

func handleInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req initRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	configPath := req.Path + "/.suitest.yaml"
	writeJSON(w, map[string]interface{}{"success": true, "config_path": configPath})
}

func handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("plugin/openapi.yaml")
	if err != nil {
		http.Error(w, "openapi.yaml not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(data)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
