package report

import (
	"encoding/json"
	"io"

	"github.com/mulhamna/suitest/internal/agent"
)

// JSONReporter writes a JSON report.
type JSONReporter struct {
	Indent bool
}

// NewJSONReporter creates a JSONReporter.
func NewJSONReporter() *JSONReporter {
	return &JSONReporter{Indent: true}
}

func (r *JSONReporter) Write(w io.Writer, result *agent.RunResult) error {
	encoder := json.NewEncoder(w)
	if r.Indent {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(result)
}
