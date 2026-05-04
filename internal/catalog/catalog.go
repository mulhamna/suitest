package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mulhamna/suitest/internal/agent"
	"github.com/mulhamna/suitest/internal/storage"
	"gopkg.in/yaml.v3"
)

const (
	targetsDir   = "targets"
	scenariosDir = "scenarios"
)

// Target is a saved testable asset that QA can rerun without re-entering input.
type Target struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Path        string `yaml:"path"`
	URL         string `yaml:"url,omitempty"`
	Curl        string `yaml:"curl,omitempty"`
	Expectation string `yaml:"expectation,omitempty"`
	ScenarioSet string `yaml:"scenario_set,omitempty"`
}

// ScenarioSet stores mapped scenarios for a target.
type ScenarioSet struct {
	Name        string           `yaml:"name"`
	TargetName  string           `yaml:"target_name"`
	Mode        string           `yaml:"mode"`
	Summary     string           `yaml:"summary,omitempty"`
	Expectation string           `yaml:"expectation,omitempty"`
	Approved    bool             `yaml:"approved"`
	ApprovedAt  string           `yaml:"approved_at,omitempty"`
	Plans       []agent.TestPlan `yaml:"plans"`
}

func SaveTarget(target *Target) error {
	if err := validateTarget(target); err != nil {
		return err
	}
	if storage.CurrentDriver() == storage.DriverSQLite {
		data, err := json.Marshal(target)
		if err != nil {
			return err
		}
		return storage.UpsertTarget(target.Name, data)
	}
	path, err := targetPath(target.Name)
	if err != nil {
		return err
	}
	return writeYAML(path, target)
}

func LoadTarget(name string) (*Target, error) {
	if storage.CurrentDriver() == storage.DriverSQLite {
		data, err := storage.GetTarget(name)
		if err != nil {
			return nil, err
		}
		var target Target
		if err := json.Unmarshal(data, &target); err != nil {
			return nil, err
		}
		return &target, nil
	}
	path, err := targetPath(name)
	if err != nil {
		return nil, err
	}
	var target Target
	if err := readYAML(path, &target); err != nil {
		return nil, err
	}
	return &target, nil
}

func ListTargets() ([]Target, error) {
	if storage.CurrentDriver() == storage.DriverSQLite {
		records, err := storage.ListTargets()
		if err != nil {
			return nil, err
		}
		targets := make([]Target, 0, len(records))
		for _, record := range records {
			var target Target
			if err := json.Unmarshal(record, &target); err == nil {
				targets = append(targets, target)
			}
		}
		sort.Slice(targets, func(i, j int) bool { return targets[i].Name < targets[j].Name })
		return targets, nil
	}
	dir, err := storageDir(targetsDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Target{}, nil
		}
		return nil, err
	}

	var targets []Target
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		var target Target
		if err := readYAML(filepath.Join(dir, entry.Name()), &target); err == nil {
			targets = append(targets, target)
		}
	}

	sort.Slice(targets, func(i, j int) bool { return targets[i].Name < targets[j].Name })
	return targets, nil
}

func SaveScenarioSet(set *ScenarioSet) error {
	if set == nil {
		return fmt.Errorf("scenario set is required")
	}
	if strings.TrimSpace(set.Name) == "" {
		return fmt.Errorf("scenario set name is required")
	}
	if strings.TrimSpace(set.TargetName) == "" {
		return fmt.Errorf("target name is required")
	}
	if storage.CurrentDriver() == storage.DriverSQLite {
		data, err := json.Marshal(set)
		if err != nil {
			return err
		}
		return storage.UpsertScenarioSet(set.TargetName, set.Name, data)
	}
	path, err := scenarioPath(set.TargetName, set.Name)
	if err != nil {
		return err
	}
	return writeYAML(path, set)
}

func LoadScenarioSet(targetName, setName string) (*ScenarioSet, error) {
	if storage.CurrentDriver() == storage.DriverSQLite {
		data, err := storage.GetScenarioSet(targetName, setName)
		if err != nil {
			return nil, err
		}
		var set ScenarioSet
		if err := json.Unmarshal(data, &set); err != nil {
			return nil, err
		}
		return &set, nil
	}
	path, err := scenarioPath(targetName, setName)
	if err != nil {
		return nil, err
	}
	var set ScenarioSet
	if err := readYAML(path, &set); err != nil {
		return nil, err
	}
	return &set, nil
}

func ListScenarioSets(targetName string) ([]ScenarioSet, error) {
	if storage.CurrentDriver() == storage.DriverSQLite {
		records, err := storage.ListScenarioSets(targetName)
		if err != nil {
			return nil, err
		}
		sets := make([]ScenarioSet, 0, len(records))
		for _, record := range records {
			var set ScenarioSet
			if err := json.Unmarshal(record, &set); err == nil {
				sets = append(sets, set)
			}
		}
		sort.Slice(sets, func(i, j int) bool { return sets[i].Name < sets[j].Name })
		return sets, nil
	}
	dir, err := storageDir(filepath.Join(scenariosDir, sanitizeName(targetName)))
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScenarioSet{}, nil
		}
		return nil, err
	}

	var sets []ScenarioSet
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		var set ScenarioSet
		if err := readYAML(filepath.Join(dir, entry.Name()), &set); err == nil {
			sets = append(sets, set)
		}
	}

	sort.Slice(sets, func(i, j int) bool { return sets[i].Name < sets[j].Name })
	return sets, nil
}

func validateTarget(target *Target) error {
	if target == nil {
		return fmt.Errorf("target is required")
	}
	target.Name = strings.TrimSpace(target.Name)
	target.Type = strings.TrimSpace(strings.ToLower(target.Type))
	target.Path = strings.TrimSpace(target.Path)
	target.URL = strings.TrimSpace(target.URL)
	target.Curl = strings.TrimSpace(target.Curl)
	target.Expectation = strings.TrimSpace(target.Expectation)
	target.ScenarioSet = strings.TrimSpace(target.ScenarioSet)

	if target.Name == "" {
		return fmt.Errorf("target name is required")
	}
	if target.Type != "frontend" && target.Type != "backend" {
		return fmt.Errorf("target type must be frontend or backend")
	}
	if target.Path == "" {
		target.Path = "."
	}
	if target.Type == "frontend" && target.URL == "" {
		return fmt.Errorf("frontend target requires a URL")
	}
	if target.Type == "backend" && target.Curl == "" {
		return fmt.Errorf("backend target requires a curl command")
	}
	if target.ScenarioSet == "" {
		target.ScenarioSet = "default"
	}
	return nil
}

func targetPath(name string) (string, error) {
	dir, err := storageDir(targetsDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sanitizeName(name)+".yaml"), nil
}

func scenarioPath(targetName, setName string) (string, error) {
	dir, err := storageDir(filepath.Join(scenariosDir, sanitizeName(targetName)))
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sanitizeName(setName)+".yaml"), nil
}

func storageDir(rel string) (string, error) {
	return storage.Dir(rel)
}

func writeYAML(path string, value interface{}) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func readYAML(path string, value interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, value)
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	var builder strings.Builder
	lastDash := false
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	sanitized := strings.Trim(builder.String(), "-")
	if sanitized == "" {
		return "target"
	}
	return sanitized
}
