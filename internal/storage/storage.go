package storage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mulhamna/suitest/internal/config"
)

const (
	defaultDBName     = "suitest.db"
	lastReportJSON    = "last-report.json"
	reportRecordKey   = "last"
	runsDirName       = "runs"
	targetsTableSQL   = "CREATE TABLE IF NOT EXISTS targets (name TEXT PRIMARY KEY, body TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);"
	scenariosTableSQL = "CREATE TABLE IF NOT EXISTS scenario_sets (target_name TEXT NOT NULL, name TEXT NOT NULL, body TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (target_name, name));"
	reportsTableSQL   = "CREATE TABLE IF NOT EXISTS reports (key TEXT PRIMARY KEY, body TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);"
	runsTableSQL      = "CREATE TABLE IF NOT EXISTS runs (run_id TEXT PRIMARY KEY, body TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);"
)

type Driver string

const (
	DriverJSON   Driver = "json"
	DriverSQLite Driver = "sqlite"
)

func CurrentDriver() Driver {
	cfg, err := config.Load()
	if err != nil {
		return DriverJSON
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Storage.Driver)) {
	case string(DriverSQLite):
		return DriverSQLite
	default:
		return DriverJSON
	}
}

func RootDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home dir: %w", err)
	}
	dir := filepath.Join(home, ".suitest")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("could not create dir: %w", err)
	}
	return dir, nil
}

func DBPath() (string, error) {
	cfg, err := config.Load()
	if err == nil && strings.TrimSpace(cfg.Storage.Path) != "" {
		return expandAndPrepareDBPath(cfg.Storage.Path)
	}
	root, err := RootDir()
	if err != nil {
		return "", err
	}
	return expandAndPrepareDBPath(filepath.Join(root, defaultDBName))
}

func Dir(rel string) (string, error) {
	root, err := RootDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, rel)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("could not create %s: %w", dir, err)
	}
	return dir, nil
}

func SaveReport(data []byte) error {
	switch CurrentDriver() {
	case DriverSQLite:
		if err := ensureSQLiteSchema(); err != nil {
			return err
		}
		dbPath, err := DBPath()
		if err != nil {
			return err
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		runID := extractRunID(data)
		sqlParts := []string{
			fmt.Sprintf(
				"INSERT INTO reports (key, body, updated_at) VALUES (%s, %s, CURRENT_TIMESTAMP) ON CONFLICT(key) DO UPDATE SET body = excluded.body, updated_at = CURRENT_TIMESTAMP;",
				sqlQuote(reportRecordKey), sqlQuote(encoded),
			),
		}
		if runID != "" {
			sqlParts = append(sqlParts, fmt.Sprintf(
				"INSERT INTO runs (run_id, body, updated_at) VALUES (%s, %s, CURRENT_TIMESTAMP) ON CONFLICT(run_id) DO UPDATE SET body = excluded.body, updated_at = CURRENT_TIMESTAMP;",
				sqlQuote(runID), sqlQuote(encoded),
			))
		}
		if _, err := runSQLite(dbPath, strings.Join(sqlParts, "\n")); err != nil {
			return err
		}
		return nil
	default:
		root, err := RootDir()
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, lastReportJSON), data, 0644); err != nil {
			return err
		}
		runID := extractRunID(data)
		if runID != "" {
			runsDir, err := Dir(runsDirName)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(runsDir, runID+".json"), data, 0644); err != nil {
				return err
			}
		}
		return nil
	}
}

func LoadLastReportData() ([]byte, error) {
	var data []byte
	switch CurrentDriver() {
	case DriverSQLite:
		if err := ensureSQLiteSchema(); err != nil {
			return nil, err
		}
		dbPath, err := DBPath()
		if err != nil {
			return nil, err
		}
		out, err := runSQLite(dbPath, "SELECT body FROM reports WHERE key = 'last' LIMIT 1;")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(out) == "" {
			return nil, fmt.Errorf("no report found")
		}
		data, err = base64.StdEncoding.DecodeString(strings.TrimSpace(out))
		if err != nil {
			return nil, fmt.Errorf("decode sqlite report: %w", err)
		}
	default:
		root, err := RootDir()
		if err != nil {
			return nil, err
		}
		data, err = os.ReadFile(filepath.Join(root, lastReportJSON))
		if err != nil {
			return nil, fmt.Errorf("could not read last report: %w", err)
		}
	}
	return data, nil
}

func LoadRunReportData(runID string) ([]byte, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("run id is required")
	}
	switch CurrentDriver() {
	case DriverSQLite:
		if err := ensureSQLiteSchema(); err != nil {
			return nil, err
		}
		dbPath, err := DBPath()
		if err != nil {
			return nil, err
		}
		out, err := runSQLite(dbPath, fmt.Sprintf("SELECT body FROM runs WHERE run_id = %s LIMIT 1;", sqlQuote(runID)))
		if err != nil {
			return nil, err
		}
		out = strings.TrimSpace(out)
		if out == "" {
			return nil, os.ErrNotExist
		}
		return base64.StdEncoding.DecodeString(out)
	default:
		runsDir, err := Dir(runsDirName)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(filepath.Join(runsDir, runID+".json"))
		if err != nil {
			return nil, err
		}
		return data, nil
	}
}

type RunSummary struct {
	RunID      string
	Mode       string
	Provider   string
	Path       string
	StartedAt  string
	FinishedAt string
	Passed     int
	Failed     int
	TotalTests int
	DryRun     bool
}

func ListRunSummaries(limit int) ([]RunSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	switch CurrentDriver() {
	case DriverSQLite:
		if err := ensureSQLiteSchema(); err != nil {
			return nil, err
		}
		dbPath, err := DBPath()
		if err != nil {
			return nil, err
		}
		query := fmt.Sprintf(".mode line\nSELECT run_id, body FROM runs ORDER BY updated_at DESC LIMIT %d;", limit)
		out, err := runSQLite(dbPath, query)
		if err != nil {
			return nil, err
		}
		return parseRunSummaries(out), nil
	default:
		runsDir, err := Dir(runsDirName)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(runsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return []RunSummary{}, nil
			}
			return nil, err
		}
		summaries := make([]RunSummary, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(runsDir, entry.Name()))
			if err != nil {
				continue
			}
			summary, ok := decodeRunSummary(data)
			if ok {
				summaries = append(summaries, summary)
			}
		}
		sortRunSummaries(summaries)
		if len(summaries) > limit {
			summaries = summaries[:limit]
		}
		return summaries, nil
	}
}

func EnsureCatalogSchema() error {
	return ensureSQLiteSchema()
}

func ValidateSQLiteConfig() error {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return fmt.Errorf("sqlite storage requires sqlite3 to be installed and available in PATH")
	}
	if _, err := DBPath(); err != nil {
		return err
	}
	return nil
}

func ValidateSQLitePath(path string) error {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return fmt.Errorf("sqlite storage requires sqlite3 to be installed and available in PATH")
	}
	if _, err := expandAndPrepareDBPath(path); err != nil {
		return err
	}
	return nil
}

func UpsertTarget(name string, body []byte) error {
	dbPath, err := DBPath()
	if err != nil {
		return err
	}
	if err := ensureSQLiteSchema(); err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(body)
	sql := fmt.Sprintf(
		"INSERT INTO targets (name, body, updated_at) VALUES (%s, %s, CURRENT_TIMESTAMP) ON CONFLICT(name) DO UPDATE SET body = excluded.body, updated_at = CURRENT_TIMESTAMP;",
		sqlQuote(name), sqlQuote(encoded),
	)
	_, err = runSQLite(dbPath, sql)
	return err
}

func GetTarget(name string) ([]byte, error) {
	dbPath, err := DBPath()
	if err != nil {
		return nil, err
	}
	if err := ensureSQLiteSchema(); err != nil {
		return nil, err
	}
	out, err := runSQLite(dbPath, fmt.Sprintf("SELECT body FROM targets WHERE name = %s LIMIT 1;", sqlQuote(name)))
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, os.ErrNotExist
	}
	decoded, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		return nil, fmt.Errorf("decode target: %w", err)
	}
	return decoded, nil
}

func ListTargets() ([][]byte, error) {
	dbPath, err := DBPath()
	if err != nil {
		return nil, err
	}
	if err := ensureSQLiteSchema(); err != nil {
		return nil, err
	}
	out, err := runSQLite(dbPath, ".mode line\nSELECT body FROM targets ORDER BY name;")
	if err != nil {
		return nil, err
	}
	return parseLineModeBodies(out, "body"), nil
}

func UpsertScenarioSet(targetName, setName string, body []byte) error {
	dbPath, err := DBPath()
	if err != nil {
		return err
	}
	if err := ensureSQLiteSchema(); err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(body)
	sql := fmt.Sprintf(
		"INSERT INTO scenario_sets (target_name, name, body, updated_at) VALUES (%s, %s, %s, CURRENT_TIMESTAMP) ON CONFLICT(target_name, name) DO UPDATE SET body = excluded.body, updated_at = CURRENT_TIMESTAMP;",
		sqlQuote(targetName), sqlQuote(setName), sqlQuote(encoded),
	)
	_, err = runSQLite(dbPath, sql)
	return err
}

func GetScenarioSet(targetName, setName string) ([]byte, error) {
	dbPath, err := DBPath()
	if err != nil {
		return nil, err
	}
	if err := ensureSQLiteSchema(); err != nil {
		return nil, err
	}
	out, err := runSQLite(dbPath, fmt.Sprintf(
		"SELECT body FROM scenario_sets WHERE target_name = %s AND name = %s LIMIT 1;",
		sqlQuote(targetName), sqlQuote(setName),
	))
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, os.ErrNotExist
	}
	decoded, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		return nil, fmt.Errorf("decode scenario set: %w", err)
	}
	return decoded, nil
}

func ListScenarioSets(targetName string) ([][]byte, error) {
	dbPath, err := DBPath()
	if err != nil {
		return nil, err
	}
	if err := ensureSQLiteSchema(); err != nil {
		return nil, err
	}
	out, err := runSQLite(dbPath, fmt.Sprintf(
		".mode line\nSELECT body FROM scenario_sets WHERE target_name = %s ORDER BY name;",
		sqlQuote(targetName),
	))
	if err != nil {
		return nil, err
	}
	return parseLineModeBodies(out, "body"), nil
}

func ensureSQLiteSchema() error {
	if CurrentDriver() != DriverSQLite {
		return nil
	}
	if err := ValidateSQLiteConfig(); err != nil {
		return err
	}
	dbPath, err := DBPath()
	if err != nil {
		return err
	}
	sql := strings.Join([]string{targetsTableSQL, scenariosTableSQL, reportsTableSQL, runsTableSQL}, "\n")
	_, err = runSQLite(dbPath, sql)
	return err
}

func runSQLite(dbPath, sql string) (string, error) {
	cmd := exec.Command("sqlite3", dbPath)
	cmd.Stdin = strings.NewReader(sql)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("sqlite3 failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func sqlQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func parseLineModeBodies(output, key string) [][]byte {
	lines := strings.Split(output, "\n")
	bodies := make([][]byte, 0)
	prefix := key + " = "
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, prefix))
			if err == nil {
				bodies = append(bodies, decoded)
			}
		}
	}
	return bodies
}

func extractRunID(data []byte) string {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	runID, _ := payload["run_id"].(string)
	return strings.TrimSpace(runID)
}

func parseRunSummaries(output string) []RunSummary {
	lines := strings.Split(output, "\n")
	summaries := make([]RunSummary, 0)
	var currentRunID string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "run_id = ") {
			currentRunID = strings.TrimSpace(strings.TrimPrefix(line, "run_id = "))
			continue
		}
		if strings.HasPrefix(line, "body = ") {
			encoded := strings.TrimSpace(strings.TrimPrefix(line, "body = "))
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				continue
			}
			summary, ok := decodeRunSummary(decoded)
			if ok {
				if summary.RunID == "" {
					summary.RunID = currentRunID
				}
				summaries = append(summaries, summary)
			}
		}
	}
	return summaries
}

func decodeRunSummary(data []byte) (RunSummary, bool) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return RunSummary{}, false
	}
	summary := RunSummary{
		RunID:      stringValue(payload["run_id"]),
		Mode:       stringValue(payload["mode"]),
		Provider:   stringValue(payload["provider"]),
		Path:       stringValue(payload["path"]),
		StartedAt:  stringValue(payload["started_at"]),
		FinishedAt: stringValue(payload["finished_at"]),
		Passed:     intValue(payload["passed"]),
		Failed:     intValue(payload["failed"]),
		TotalTests: intValue(payload["total_tests"]),
		DryRun:     boolValue(payload["dry_run"]),
	}
	return summary, true
}

func sortRunSummaries(summaries []RunSummary) {
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].StartedAt == summaries[j].StartedAt {
			return summaries[i].RunID > summaries[j].RunID
		}
		return summaries[i].StartedAt > summaries[j].StartedAt
	})
}

func stringValue(value interface{}) string {
	s, _ := value.(string)
	return s
}

func intValue(value interface{}) int {
	number, ok := value.(float64)
	if !ok {
		return 0
	}
	return int(number)
}

func boolValue(value interface{}) bool {
	flag, _ := value.(bool)
	return flag
}

func expandAndPrepareDBPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("storage path cannot be empty")
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not find home dir: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve storage path: %w", err)
		}
		path = absPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create storage directory: %w", err)
	}
	return path, nil
}
