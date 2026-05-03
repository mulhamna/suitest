package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the merged global + project configuration.
type Config struct {
	DefaultProvider string                     `yaml:"default_provider"`
	Providers       map[string]*ProviderConfig `yaml:"providers"`
	Agent           AgentConfig                `yaml:"agent"`
	Storage         StorageConfig              `yaml:"storage"`
	Operator        OperatorConfig             `yaml:"operator"`

	// Project-level fields (from .suitest.yaml)
	Mode     string `yaml:"mode"`
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	EntryURL string `yaml:"entry_url"`
	TestDir  string `yaml:"test_dir"`

	// Runtime-only fields (set via CLI flags)
	DryRun bool   `yaml:"-"`
	Output string `yaml:"-"`
}

// ProviderConfig holds per-provider settings.
type ProviderConfig struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

// AgentConfig holds agent behaviour settings.
type AgentConfig struct {
	MaxRetries  int  `yaml:"max_retries"`
	Concurrency int  `yaml:"concurrency"`
	AutoFix     bool `yaml:"auto_fix"`
}

// StorageConfig holds report persistence preferences.
type StorageConfig struct {
	Driver string `yaml:"driver"`
	Path   string `yaml:"path,omitempty"`
}

// OperatorConfig holds how suitest should invoke AI.
type OperatorConfig struct {
	Mode string `yaml:"mode"`
}

// Load loads and merges the global config (~/.suitest/config.yaml)
// with the project config (.suitest.yaml in cwd).
func Load() (*Config, error) {
	cfg := Default()

	// Load global config
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".suitest", "config.yaml")
		if data, err := os.ReadFile(globalPath); err == nil {
			var globalCfg Config
			if err := yaml.Unmarshal(interpolateEnv(data), &globalCfg); err == nil {
				mergeGlobal(cfg, &globalCfg)
			}
		}
	}

	// Load project config
	projectPath := ".suitest.yaml"
	if data, err := os.ReadFile(projectPath); err == nil {
		var projectCfg Config
		if err := yaml.Unmarshal(interpolateEnv(data), &projectCfg); err == nil {
			mergeProject(cfg, &projectCfg)
		}
	}

	// Resolve effective provider
	if cfg.Provider != "" {
		cfg.DefaultProvider = cfg.Provider
	}
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "auto"
	}

	return cfg, nil
}

// GetProviderConfig returns the config for the named provider.
func (c *Config) GetProviderConfig(name string) *ProviderConfig {
	if c.Providers == nil {
		c.Providers = make(map[string]*ProviderConfig)
	}
	if pc, ok := c.Providers[name]; ok {
		return pc
	}
	return &ProviderConfig{}
}

func mergeGlobal(dst, src *Config) {
	if src.DefaultProvider != "" {
		dst.DefaultProvider = src.DefaultProvider
	}
	if src.Storage.Driver != "" {
		dst.Storage.Driver = src.Storage.Driver
	}
	if src.Storage.Path != "" {
		dst.Storage.Path = src.Storage.Path
	}
	if src.Operator.Mode != "" {
		dst.Operator.Mode = src.Operator.Mode
	}
	if src.Providers != nil {
		for k, v := range src.Providers {
			dst.Providers[k] = v
		}
	}
	if src.Agent.MaxRetries != 0 {
		dst.Agent.MaxRetries = src.Agent.MaxRetries
	}
	if src.Agent.Concurrency != 0 {
		dst.Agent.Concurrency = src.Agent.Concurrency
	}
	if src.Agent.AutoFix {
		dst.Agent.AutoFix = true
	}
}

func mergeProject(dst, src *Config) {
	if src.Mode != "" {
		dst.Mode = src.Mode
	}
	if src.Provider != "" {
		dst.Provider = src.Provider
		dst.DefaultProvider = src.Provider
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.EntryURL != "" {
		dst.EntryURL = src.EntryURL
	}
	if src.TestDir != "" {
		dst.TestDir = src.TestDir
	}
}

// interpolateEnv replaces ${VAR} and $VAR patterns in YAML data with env values.
func interpolateEnv(data []byte) []byte {
	re := regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Z_][A-Z0-9_]*)`)
	result := re.ReplaceAllFunc(data, func(match []byte) []byte {
		s := string(match)
		var varName string
		if strings.HasPrefix(s, "${") {
			varName = s[2 : len(s)-1]
		} else {
			varName = s[1:]
		}
		if val, ok := os.LookupEnv(varName); ok {
			return []byte(val)
		}
		return match
	})
	return result
}

// SaveReport saves a JSON report to ~/.suitest/last-report.json.
func SaveReport(data []byte) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home dir: %w", err)
	}
	dir := filepath.Join(home, ".suitest")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create dir: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "last-report.json"), data, 0644)
}

// SaveGlobal writes the config to ~/.suitest/config.yaml.
func SaveGlobal(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home dir: %w", err)
	}
	dir := filepath.Join(home, ".suitest")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0644)
}
