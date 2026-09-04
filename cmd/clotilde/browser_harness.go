package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	browserHarnessModeBlocked       = "blocked"
	browserHarnessModeAlways        = "always"
	defaultBrowserHarnessMaxPages   = 2
	defaultBrowserHarnessTimeout    = 8 * time.Second
	browserHarnessProviderReserve   = 8 * time.Second
	browserHarnessMinimumUsefulTime = 2 * time.Second
)

var blockedSearchSignals = []string{
	"access denied",
	"are you a robot",
	"blocked",
	"bot detection",
	"captcha",
	"enable javascript",
	"forbidden",
	"just a moment",
	"please verify",
	"robot check",
	"unusual traffic",
	"verifique que voce",
	"verifique que você",
}

type BrowserHarnessConfig struct {
	Enabled  bool
	Command  string
	Args     []string
	Mode     string
	MaxPages int
	Timeout  time.Duration
}

type BrowserHarnessRequest struct {
	Query    string         `json:"query"`
	Results  []SearchResult `json:"results,omitempty"`
	MaxPages int            `json:"max_pages,omitempty"`
}

type BrowserHarnessResponse struct {
	Results []SearchResult `json:"results,omitempty"`
	Error   string         `json:"error,omitempty"`
}

func browserHarnessConfigFromEnv() BrowserHarnessConfig {
	enabled := strings.EqualFold(os.Getenv("CAMOUFOX_HARNESS_ENABLED"), "true")
	command := strings.TrimSpace(os.Getenv("CAMOUFOX_HARNESS_CMD"))
	if command == "" {
		enabled = false
	}

	mode := strings.ToLower(strings.TrimSpace(os.Getenv("CAMOUFOX_HARNESS_MODE")))
	if mode == "" {
		mode = browserHarnessModeBlocked
	}
	if mode != browserHarnessModeBlocked && mode != browserHarnessModeAlways {
		log.Printf("Unknown CAMOUFOX_HARNESS_MODE=%q; using %q", mode, browserHarnessModeBlocked)
		mode = browserHarnessModeBlocked
	}

	maxPages := defaultBrowserHarnessMaxPages
	if raw := strings.TrimSpace(os.Getenv("CAMOUFOX_HARNESS_MAX_PAGES")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			maxPages = parsed
		}
	}

	timeout := defaultBrowserHarnessTimeout
	if raw := strings.TrimSpace(os.Getenv("CAMOUFOX_HARNESS_TIMEOUT_SECONDS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			timeout = time.Duration(parsed) * time.Second
		}
	}

	return BrowserHarnessConfig{
		Enabled:  enabled,
		Command:  command,
		Args:     strings.Fields(os.Getenv("CAMOUFOX_HARNESS_ARGS")),
		Mode:     mode,
		MaxPages: maxPages,
		Timeout:  timeout,
	}
}

func (s *Server) performSearchWithBrowserHarness(ctx context.Context, query string) ([]SearchResult, error) {
	rendered, err := s.performBrowserHarnessSearch(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	if len(rendered) == 0 {
		return nil, fmt.Errorf("browser search harness returned no usable rendered context")
	}
	return rendered, nil
}

func (s *Server) shouldUseBrowserHarness(results []SearchResult) bool {
	cfg := s.browserHarness
	if !cfg.Enabled || cfg.Command == "" {
		return false
	}
	if cfg.Mode == browserHarnessModeAlways {
		return len(results) > 0
	}
	return searchResultsLookBlocked(results)
}

func searchResultsLookBlocked(results []SearchResult) bool {
	if len(results) == 0 {
		return false
	}
	for _, result := range results {
		haystack := strings.ToLower(strings.Join([]string{result.Title, result.Snippet}, " "))
		if searchTextLooksBlocked(haystack) {
			return true
		}
	}
	return false
}

func searchTextLooksBlocked(text string) bool {
	haystack := strings.ToLower(text)
	for _, signal := range blockedSearchSignals {
		if strings.Contains(haystack, signal) {
			return true
		}
	}
	return false
}

func (s *Server) performBrowserHarnessSearch(ctx context.Context, query string, results []SearchResult) ([]SearchResult, error) {
	cfg := s.browserHarness
	if !cfg.Enabled || cfg.Command == "" {
		return nil, fmt.Errorf("browser harness is not configured")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultBrowserHarnessTimeout
	}
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = defaultBrowserHarnessMaxPages
	}
	timeout, err := browserHarnessTimeoutWithinBudget(ctx, cfg.Timeout)
	if err != nil {
		return nil, err
	}

	reqBody := BrowserHarnessRequest{
		Query:    query,
		Results:  results,
		MaxPages: cfg.MaxPages,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal browser harness request: %w", err)
	}

	harnessCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := append([]string{}, cfg.Args...)
	cmd := exec.CommandContext(harnessCtx, cfg.Command, args...)
	cmd.Stdin = bytes.NewReader(payload)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if harnessCtx.Err() != nil {
			return nil, fmt.Errorf("browser harness timed out after %s", timeout)
		}
		return nil, fmt.Errorf("browser harness command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var resp BrowserHarnessResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse browser harness response: %w", err)
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.Results, nil
}

func browserHarnessTimeoutWithinBudget(ctx context.Context, configured time.Duration) (time.Duration, error) {
	if configured <= 0 {
		configured = defaultBrowserHarnessTimeout
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return configured, nil
	}

	remaining := time.Until(deadline)
	maxHarnessTime := remaining - browserHarnessProviderReserve
	if maxHarnessTime < browserHarnessMinimumUsefulTime {
		return 0, fmt.Errorf("not enough request time remaining for browser harness: %s", remaining.Round(time.Millisecond))
	}
	if configured > maxHarnessTime {
		return maxHarnessTime, nil
	}
	return configured, nil
}

func mergeSearchResults(primary, rendered []SearchResult) []SearchResult {
	if len(rendered) == 0 {
		return primary
	}
	merged := make([]SearchResult, 0, len(primary)+len(rendered))
	seen := map[string]bool{}
	for _, result := range primary {
		if result.URL == "" || seen[result.URL] {
			continue
		}
		seen[result.URL] = true
		merged = append(merged, result)
	}
	for _, result := range rendered {
		if result.URL == "" || seen[result.URL] {
			continue
		}
		seen[result.URL] = true
		merged = append(merged, result)
	}
	return merged
}
