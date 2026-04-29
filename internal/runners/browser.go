package runners

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// BrowserTestCase defines a single browser-based test scenario.
type BrowserTestCase struct {
	Name    string        `json:"name"`
	URL     string        `json:"url"`
	Actions []BrowserAction `json:"actions"`
}

// BrowserAction is a single step in a browser test.
type BrowserAction struct {
	Type     string `json:"type"`     // navigate, click, type, assert_text, assert_visible, screenshot, wait
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"`
	URL      string `json:"url,omitempty"`
}

// BrowserRunner runs E2E browser tests using chromedp.
type BrowserRunner struct {
	headless bool
}

// NewBrowserRunner creates a browser runner.
func NewBrowserRunner(headless bool) *BrowserRunner {
	return &BrowserRunner{headless: headless}
}

func (r *BrowserRunner) Name() string { return "browser" }

func (r *BrowserRunner) Run(ctx context.Context, path string, testCode string) ([]RunResult, error) {
	if testCode == "" {
		return nil, fmt.Errorf("browser runner requires test code (JSON test cases)")
	}

	var testCases []BrowserTestCase
	if err := json.Unmarshal([]byte(testCode), &testCases); err != nil {
		var single BrowserTestCase
		if err2 := json.Unmarshal([]byte(testCode), &single); err2 != nil {
			return nil, fmt.Errorf("invalid browser test format: %w", err)
		}
		testCases = []BrowserTestCase{single}
	}

	var results []RunResult
	for _, tc := range testCases {
		result := r.runTestCase(ctx, tc)
		results = append(results, result)
	}
	return results, nil
}

func (r *BrowserRunner) RunFile(ctx context.Context, path string) ([]RunResult, error) {
	return nil, fmt.Errorf("BrowserRunner.RunFile not supported; use Run with JSON test code")
}

func (r *BrowserRunner) runTestCase(ctx context.Context, tc BrowserTestCase) RunResult {
	start := time.Now()
	result := RunResult{Name: tc.Name}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", r.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()

	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx)
	defer cancelChrome()

	timeoutCtx, cancelTimeout := context.WithTimeout(chromeCtx, 60*time.Second)
	defer cancelTimeout()

	var outputLines []string
	var failures []string

	for _, action := range tc.Actions {
		var task chromedp.Action
		var taskErr error

		switch action.Type {
		case "navigate":
			url := action.URL
			if url == "" {
				url = tc.URL
			}
			task = chromedp.Navigate(url)
			outputLines = append(outputLines, fmt.Sprintf("navigate → %s", url))

		case "click":
			task = chromedp.Click(action.Selector, chromedp.ByQuery)
			outputLines = append(outputLines, fmt.Sprintf("click %s", action.Selector))

		case "type":
			task = chromedp.SendKeys(action.Selector, action.Value, chromedp.ByQuery)
			outputLines = append(outputLines, fmt.Sprintf("type %q into %s", action.Value, action.Selector))

		case "wait":
			task = chromedp.WaitVisible(action.Selector, chromedp.ByQuery)
			outputLines = append(outputLines, fmt.Sprintf("wait for %s", action.Selector))

		case "assert_text":
			var text string
			task = chromedp.Text(action.Selector, &text, chromedp.ByQuery)
			if taskErr = chromedp.Run(timeoutCtx, task); taskErr == nil {
				if !strings.Contains(text, action.Value) {
					failures = append(failures, fmt.Sprintf("assert_text: %s does not contain %q (got %q)", action.Selector, action.Value, text))
				}
				outputLines = append(outputLines, fmt.Sprintf("assert_text %s contains %q: ok", action.Selector, action.Value))
			}
			task = nil // Already executed

		case "assert_visible":
			var isVisible bool
			task = chromedp.Evaluate(fmt.Sprintf(`document.querySelector(%q) !== null`, action.Selector), &isVisible)
			if taskErr = chromedp.Run(timeoutCtx, task); taskErr == nil {
				if !isVisible {
					failures = append(failures, fmt.Sprintf("assert_visible: %s is not visible", action.Selector))
				}
				outputLines = append(outputLines, fmt.Sprintf("assert_visible %s: ok", action.Selector))
			}
			task = nil

		case "screenshot":
			var buf []byte
			task = chromedp.FullScreenshot(&buf, 90)
			outputLines = append(outputLines, "screenshot captured")

		default:
			outputLines = append(outputLines, fmt.Sprintf("unknown action: %s (skipped)", action.Type))
			continue
		}

		if task != nil {
			if taskErr = chromedp.Run(timeoutCtx, task); taskErr != nil {
				failures = append(failures, fmt.Sprintf("action %s failed: %v", action.Type, taskErr))
			}
		} else if taskErr != nil {
			failures = append(failures, fmt.Sprintf("action %s failed: %v", action.Type, taskErr))
		}
	}

	result.Runtime = time.Since(start).String()
	result.Output = strings.Join(outputLines, "\n")

	if len(failures) > 0 {
		result.Passed = false
		result.Error = strings.Join(failures, "; ")
	} else {
		result.Passed = true
	}

	return result
}
