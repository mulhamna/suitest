package runners

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Detect inspects the project root directory and returns the best runner mode.
// Priority: .suitest.yaml mode > file signals > fallback "unit"
func Detect(projectRoot string) (string, error) {
	// Check for .suitest.yaml override
	suitestYAML := filepath.Join(projectRoot, ".suitest.yaml")
	if data, err := os.ReadFile(suitestYAML); err == nil {
		mode := extractYAMLField(data, "mode")
		if mode != "" && mode != "auto" {
			return mode, nil
		}
	}

	// Check for go.mod → go test
	if fileExists(filepath.Join(projectRoot, "go.mod")) {
		return "unit", nil
	}

	// Check for package.json with jest/vitest
	if hasJestOrVitest(projectRoot) {
		return "unit", nil
	}

	// Check for Python project
	if fileExists(filepath.Join(projectRoot, "requirements.txt")) ||
		fileExists(filepath.Join(projectRoot, "pyproject.toml")) ||
		fileExists(filepath.Join(projectRoot, "setup.py")) {
		return "unit", nil
	}

	// Default fallback
	return "unit", nil
}

// DetectSubtype returns the specific test runner within a mode (e.g., "go", "jest", "pytest").
func DetectSubtype(projectRoot string) string {
	if fileExists(filepath.Join(projectRoot, "go.mod")) {
		return "go"
	}
	if hasJestOrVitest(projectRoot) {
		pkgJSON := filepath.Join(projectRoot, "package.json")
		if data, err := os.ReadFile(pkgJSON); err == nil {
			if containsString(data, "vitest") {
				return "vitest"
			}
		}
		return "jest"
	}
	if fileExists(filepath.Join(projectRoot, "requirements.txt")) ||
		fileExists(filepath.Join(projectRoot, "pyproject.toml")) ||
		fileExists(filepath.Join(projectRoot, "setup.py")) {
		return "pytest"
	}
	return "go"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasJestOrVitest(projectRoot string) bool {
	pkgJSON := filepath.Join(projectRoot, "package.json")
	data, err := os.ReadFile(pkgJSON)
	if err != nil {
		return false
	}
	return containsString(data, "jest") || containsString(data, "vitest")
}

func containsString(data []byte, s string) bool {
	return len(data) > 0 && findInJSON(data, s)
}

func findInJSON(data []byte, needle string) bool {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	return searchJSON(m, needle)
}

func searchJSON(v interface{}, needle string) bool {
	switch val := v.(type) {
	case string:
		return val == needle || containsSubstring(val, needle)
	case map[string]interface{}:
		for k, child := range val {
			if containsSubstring(k, needle) || searchJSON(child, needle) {
				return true
			}
		}
	case []interface{}:
		for _, child := range val {
			if searchJSON(child, needle) {
				return true
			}
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// extractYAMLField extracts a simple string field from YAML (no full YAML parse needed).
func extractYAMLField(data []byte, field string) string {
	lines := splitLines(data)
	prefix := field + ":"
	for _, line := range lines {
		trimmed := trimLeft(line)
		if len(trimmed) > len(prefix) && trimmed[:len(prefix)] == prefix {
			val := trimmed[len(prefix):]
			val = trimLeft(val)
			val = trimRight(val)
			// Remove quotes
			if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
				val = val[1 : len(val)-1]
			}
			return val
		}
	}
	return ""
}

func splitLines(data []byte) []string {
	var lines []string
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, string(data[start:i]))
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, string(data[start:]))
	}
	return lines
}

func trimLeft(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[i:]
}

func trimRight(s string) string {
	i := len(s)
	for i > 0 && (s[i-1] == ' ' || s[i-1] == '\t' || s[i-1] == '\r') {
		i--
	}
	return s[:i]
}
