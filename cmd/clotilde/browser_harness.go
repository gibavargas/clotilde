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
	browserHarnessModeBlocked = "blocked"
	browserHarnessModeAlways  = "always"
)

type BrowserHarnessConfig struct {
	Enabled  bool
	Command  string
	Args     []string
	Mode     string
	MaxPages int
	Timeout  time.Duration
}

type BrowserHarnessRequest struct {
	Query    string                   `json:"query"`
	Results  []PerplexitySearchResult `json:"results,omitempty"`
	MaxPages int                      `json:"max_pages,omitempty"`
}

type BrowserHarnessResponse struct {
	Results []PerplexitySearchResult `json:"results,omitempty"`
	Error   string                   `json:"error,omitempty"`
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

	maxPages := 3
	if raw := strings.TrimSpace(os.Getenv("CAMOUFOX_HARNESS_MAX_PAGES")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			maxPages = parsed
		}
	}

	timeout := 12 * time.Second
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

func (s *Server) performSearchWithBrowserHarness(ctx context.Context, query string) ([]PerplexitySearchResult, error) {
	results, err := s.performPerplexitySearch(ctx, query)
	if err != nil {
		return nil, err
	}
	if !s.shouldUseBrowserHarness(results) {
		return results, nil
	}

	rendered, err := s.performBrowserHarnessSearch(ctx, query, results)
	if err != nil {
		log.Printf("Browser search harness failed: %v; using Perplexity results", err)
		return results, nil
	}
	if len(rendered) == 0 {
		log.Printf("Browser search harness returned no usable rendered context; using Perplexity results")
		return results, nil
	}
	return mergeSearchResults(results, rendered), nil
}

func (s *Server) shouldUseBrowserHarness(results []PerplexitySearchResult) bool {
	cfg := s.browserHarness
	if !cfg.Enabled || cfg.Command == "" {
		return false
	}
	if cfg.Mode == browserHarnessModeAlways {
		return len(results) > 0
	}
	return searchResultsLookBlocked(results)
}

func searchResultsLookBlocked(results []PerplexitySearchResult) bool {
	if len(results) == 0 {
		return false
	}
	blockedSignals := []string{
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
	for _, result := range results {
		haystack := strings.ToLower(strings.Join([]string{result.Title, result.Snippet}, " "))
		for _, signal := range blockedSignals {
			if strings.Contains(haystack, signal) {
				return true
			}
		}
	}
	return false
}

func (s *Server) performBrowserHarnessSearch(ctx context.Context, query string, results []PerplexitySearchResult) ([]PerplexitySearchResult, error) {
	cfg := s.browserHarness
	if !cfg.Enabled || cfg.Command == "" {
		return nil, fmt.Errorf("browser harness is not configured")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 12 * time.Second
	}
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 3
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

	harnessCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	args := append([]string{}, cfg.Args...)
	cmd := exec.CommandContext(harnessCtx, cfg.Command, args...)
	cmd.Stdin = bytes.NewReader(payload)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if harnessCtx.Err() != nil {
			return nil, fmt.Errorf("browser harness timed out after %s", cfg.Timeout)
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

func mergeSearchResults(primary, rendered []PerplexitySearchResult) []PerplexitySearchResult {
	if len(rendered) == 0 {
		return primary
	}
	merged := make([]PerplexitySearchResult, 0, len(primary)+len(rendered))
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
