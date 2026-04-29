package runners

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APITestCase defines a single HTTP API test case.
type APITestCase struct {
	Name     string            `json:"name"`
	Method   string            `json:"method"`
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     interface{}       `json:"body,omitempty"`
	Expect   APIExpectation    `json:"expect"`
}

// APIExpectation defines what a test expects from the response.
type APIExpectation struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
	BodyContains string       `json:"body_contains,omitempty"`
}

// APIRunner runs HTTP API tests.
type APIRunner struct {
	client *http.Client
}

// NewAPIRunner creates an API runner.
func NewAPIRunner() *APIRunner {
	return &APIRunner{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *APIRunner) Name() string { return "api" }

func (r *APIRunner) Run(ctx context.Context, path string, testCode string) ([]RunResult, error) {
	if testCode == "" {
		return nil, fmt.Errorf("API runner requires test code (JSON test cases)")
	}

	var testCases []APITestCase
	if err := json.Unmarshal([]byte(testCode), &testCases); err != nil {
		// Try single test case
		var single APITestCase
		if err2 := json.Unmarshal([]byte(testCode), &single); err2 != nil {
			return nil, fmt.Errorf("invalid API test format: %w", err)
		}
		testCases = []APITestCase{single}
	}

	var results []RunResult
	for _, tc := range testCases {
		result := r.runTestCase(ctx, tc)
		results = append(results, result)
	}
	return results, nil
}

func (r *APIRunner) RunFile(ctx context.Context, path string) ([]RunResult, error) {
	return nil, fmt.Errorf("APIRunner.RunFile not supported; use Run with JSON test code")
}

func (r *APIRunner) runTestCase(ctx context.Context, tc APITestCase) RunResult {
	start := time.Now()

	result := RunResult{
		Name: tc.Name,
	}

	// Build request body
	var bodyReader io.Reader
	if tc.Body != nil {
		bodyBytes, err := json.Marshal(tc.Body)
		if err != nil {
			result.Error = fmt.Sprintf("marshal body: %v", err)
			return result
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, tc.Method, tc.URL, bodyReader)
	if err != nil {
		result.Error = fmt.Sprintf("create request: %v", err)
		return result
	}

	if tc.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range tc.Headers {
		req.Header.Set(k, v)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("execute request: %v", err)
		return result
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("read response: %v", err)
		return result
	}

	elapsed := time.Since(start)
	result.Runtime = elapsed.String()

	var failures []string

	// Check status
	if tc.Expect.Status != 0 && resp.StatusCode != tc.Expect.Status {
		failures = append(failures, fmt.Sprintf("expected status %d, got %d", tc.Expect.Status, resp.StatusCode))
	}

	// Check headers
	for k, v := range tc.Expect.Headers {
		actual := resp.Header.Get(k)
		if actual != v {
			failures = append(failures, fmt.Sprintf("expected header %s=%q, got %q", k, v, actual))
		}
	}

	// Check body contains
	if tc.Expect.BodyContains != "" && !strings.Contains(string(respBody), tc.Expect.BodyContains) {
		failures = append(failures, fmt.Sprintf("response body does not contain %q", tc.Expect.BodyContains))
	}

	// Check body JSON match
	if tc.Expect.Body != nil {
		expectedBytes, _ := json.Marshal(tc.Expect.Body)
		if !jsonEqual(expectedBytes, respBody) {
			failures = append(failures, fmt.Sprintf("body mismatch: expected %s, got %s", string(expectedBytes), string(respBody)))
		}
	}

	result.Output = fmt.Sprintf("%s %s → %d (%s)\n%s", tc.Method, tc.URL, resp.StatusCode, elapsed, string(respBody))

	if len(failures) > 0 {
		result.Passed = false
		result.Error = strings.Join(failures, "; ")
	} else {
		result.Passed = true
	}

	return result
}

// jsonEqual checks if two JSON byte slices represent the same value.
func jsonEqual(a, b []byte) bool {
	var va, vb interface{}
	if err := json.Unmarshal(a, &va); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		return false
	}
	aa, _ := json.Marshal(va)
	bb, _ := json.Marshal(vb)
	return string(aa) == string(bb)
}
