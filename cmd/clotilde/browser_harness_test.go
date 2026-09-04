package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/clotilde/carplay-assistant/internal/admin"
	"github.com/clotilde/carplay-assistant/internal/logging"
)

func TestPerformSearchWithBrowserHarness_UsesHarnessDirectly(t *testing.T) {
	t.Setenv("CLOTILDE_BROWSER_HARNESS_HELPER", "1")

	server := &Server{
		browserHarness: BrowserHarnessConfig{
			Enabled:  true,
			Command:  os.Args[0],
			Args:     []string{"-test.run=TestBrowserHarnessHelper"},
			Mode:     browserHarnessModeAlways,
			MaxPages: 1,
		},
	}

	results, err := server.performSearchWithBrowserHarness(t.Context(), "blocked query")
	if err != nil {
		t.Fatalf("performSearchWithBrowserHarness failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one rendered result, got %+v", results)
	}
	if results[0].Title != "Rendered page" || results[0].Snippet != "Rendered page text" {
		t.Fatalf("expected rendered browser harness result, got %+v", results[0])
	}
}

func TestSearchResultsLookBlocked(t *testing.T) {
	if !searchResultsLookBlocked([]SearchResult{{Title: "Just a moment", Snippet: "Please verify you are not a robot"}}) {
		t.Fatalf("expected blocked search result to be detected")
	}
	if searchResultsLookBlocked([]SearchResult{{Title: "Public page", Snippet: "Normal public snippet."}}) {
		t.Fatalf("expected public search result to stay unblocked")
	}
}

func TestBrowserHarnessConfigFromEnv(t *testing.T) {
	t.Setenv("CAMOUFOX_HARNESS_ENABLED", "true")
	t.Setenv("CAMOUFOX_HARNESS_CMD", "python3")
	t.Setenv("CAMOUFOX_HARNESS_ARGS", "harness/camoufox_fetch.py")
	t.Setenv("CAMOUFOX_HARNESS_MODE", "always")
	t.Setenv("CAMOUFOX_HARNESS_MAX_PAGES", "2")
	t.Setenv("CAMOUFOX_HARNESS_TIMEOUT_SECONDS", "7")

	cfg := browserHarnessConfigFromEnv()
	if !cfg.Enabled || cfg.Command != "python3" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if len(cfg.Args) != 1 || cfg.Args[0] != "harness/camoufox_fetch.py" {
		t.Fatalf("unexpected args: %+v", cfg.Args)
	}
	if cfg.Mode != browserHarnessModeAlways || cfg.MaxPages != 2 || cfg.Timeout.Seconds() != 7 {
		t.Fatalf("unexpected mode/max/timeout: %+v", cfg)
	}
}

func TestBrowserHarnessConfigFromEnv_DefaultsForCarPlayBudget(t *testing.T) {
	t.Setenv("CAMOUFOX_HARNESS_ENABLED", "true")
	t.Setenv("CAMOUFOX_HARNESS_CMD", "python3")

	cfg := browserHarnessConfigFromEnv()
	if cfg.MaxPages != defaultBrowserHarnessMaxPages {
		t.Fatalf("expected default max pages %d, got %d", defaultBrowserHarnessMaxPages, cfg.MaxPages)
	}
	if cfg.Timeout != defaultBrowserHarnessTimeout {
		t.Fatalf("expected default timeout %s, got %s", defaultBrowserHarnessTimeout, cfg.Timeout)
	}
}

func TestBrowserHarnessTimeoutWithinBudget_CapsConfiguredTimeout(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(14*time.Second))
	defer cancel()

	got, err := browserHarnessTimeoutWithinBudget(ctx, 12*time.Second)
	if err != nil {
		t.Fatalf("expected timeout to be capped, got error: %v", err)
	}
	if got < 5*time.Second || got > 7*time.Second {
		t.Fatalf("expected capped timeout around 6s, got %s", got)
	}
}

func TestBrowserHarnessTimeoutWithinBudget_RequiresProviderReserve(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(9*time.Second))
	defer cancel()

	if _, err := browserHarnessTimeoutWithinBudget(ctx, 8*time.Second); err == nil {
		t.Fatalf("expected error when provider generation reserve would be exhausted")
	}
}

func TestLiveClaudeUsesBrowserHarnessEvidence(t *testing.T) {
	if os.Getenv("RUN_LIVE_PROVIDER_TESTS") != "1" {
		t.Skip("set RUN_LIVE_PROVIDER_TESTS=1 to call the external provider")
	}
	claudeKey := os.Getenv("CLAUDE_KEY_SECRET_NAME")
	if claudeKey == "" {
		t.Skip("CLAUDE_KEY_SECRET_NAME is not set")
	}
	t.Setenv("CLOTILDE_BROWSER_HARNESS_HELPER", "1")
	t.Setenv("CLOTILDE_BROWSER_HARNESS_HELPER_TITLE", "Rendered Browser Harness Evidence")
	t.Setenv("CLOTILDE_BROWSER_HARNESS_HELPER_SNIPPET", "BROWSER_HARNESS_SENTINEL_47A9 confirms the rendered page was available.")

	config := admin.GetConfig()
	defer admin.SetConfig(config)
	if err := admin.SetConfig(admin.RuntimeConfig{
		BaseSystemPrompt: clotildeBaseSystemPromptTemplate,
		StandardModel:    defaultClaudeHaikuModel,
		PremiumModel:     defaultClaudeHaikuModel,
	}); err != nil {
		t.Fatalf("failed to configure live Claude browser harness test: %v", err)
	}

	server := &Server{
		claudeAPIKey: claudeKey,
		logger:       logging.NewLogger(4),
		browserHarness: BrowserHarnessConfig{
			Enabled:  true,
			Command:  os.Args[0],
			Args:     []string{"-test.run=TestBrowserHarnessHelper"},
			Mode:     browserHarnessModeAlways,
			MaxPages: 1,
		},
	}

	input := "Responda somente com o marcador exato que aparece no contexto renderizado pelo browser harness."
	response, err := server.createResponse(t.Context(), RouteDecision{
		Model:     defaultClaudeHaikuModel,
		WebSearch: true,
	}, "Use somente o contexto anexado. Se houver um marcador BROWSER_HARNESS_SENTINEL, responda exatamente com esse marcador e nada mais.", input)
	if err != nil {
		t.Fatalf("live Claude browser harness request failed: %v", err)
	}
	if !strings.Contains(response, "BROWSER_HARNESS_SENTINEL_47A9") {
		t.Fatalf("expected Claude to use browser harness evidence, got %q", response)
	}
	t.Logf("live Claude browser harness response=%q", response)
}

func TestBrowserHarnessHelper(t *testing.T) {
	if os.Getenv("CLOTILDE_BROWSER_HARNESS_HELPER") != "1" {
		return
	}

	var req BrowserHarnessRequest
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		t.Fatalf("failed to decode harness request: %v", err)
	}
	if req.Query == "" || req.MaxPages != 1 {
		t.Fatalf("unexpected harness request: %+v", req)
	}

	title := os.Getenv("CLOTILDE_BROWSER_HARNESS_HELPER_TITLE")
	if title == "" {
		title = "Rendered page"
	}
	snippet := os.Getenv("CLOTILDE_BROWSER_HARNESS_HELPER_SNIPPET")
	if snippet == "" {
		snippet = "Rendered page text"
	}

	json.NewEncoder(os.Stdout).Encode(BrowserHarnessResponse{
		Results: []SearchResult{
			{
				Title:   title,
				URL:     "https://example.com/rendered",
				Snippet: snippet,
			},
		},
	})
	os.Exit(0)
}
