package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/clotilde/carplay-assistant/internal/admin"
	"github.com/clotilde/carplay-assistant/internal/auth"
	"github.com/clotilde/carplay-assistant/internal/clientip"
	"github.com/clotilde/carplay-assistant/internal/logging"
	"github.com/clotilde/carplay-assistant/internal/promptinjection"
	"github.com/clotilde/carplay-assistant/internal/ratelimit"
	"github.com/clotilde/carplay-assistant/internal/router"
	"github.com/clotilde/carplay-assistant/internal/validator"
)

var startTime = time.Now()

const (
	defaultClaudeHaikuModel    = "claude-haiku-4-5-20251001"
	openRouterClaudeHaikuModel = "openrouter/anthropic/claude-haiku-4.5"
	openAIFallbackModel        = "gpt-4o-mini"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

// RouteDecision is the internal format for createResponse (compatible with router.RouteDecision)
type RouteDecision struct {
	Model           string
	WebSearch       bool
	ReasoningEffort string
}

type Server struct {
	openaiAPIKey     string
	openRouterAPIKey string
	perplexityAPIKey string
	claudeAPIKey     string // Anthropic Claude API key for fast responses
	apiKeySecret     string
	logger           *logging.Logger
}

// ClaudeRequest represents the request body for Claude Messages API
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    interface{}     `json:"system,omitempty"`
	Messages  []ClaudeMessage `json:"messages"`
	Tools     []ClaudeTool    `json:"tools,omitempty"`
}

// ClaudeMessage represents a message in the Claude conversation
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeTool represents a server-side Claude tool.
type ClaudeTool struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	MaxUses int    `json:"max_uses,omitempty"`
}

// ClaudeCacheControl marks cacheable request prefixes for Anthropic prompt caching.
type ClaudeCacheControl struct {
	Type string `json:"type"`
}

// ClaudeSystemBlock represents a system prompt block.
type ClaudeSystemBlock struct {
	Type         string              `json:"type"`
	Text         string              `json:"text"`
	CacheControl *ClaudeCacheControl `json:"cache_control,omitempty"`
}

// ClaudeContentBlock represents a content block in the Claude response.
type ClaudeContentBlock struct {
	Type    string          `json:"type"`
	Text    string          `json:"text,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
}

// ClaudeResponse represents the response from Claude Messages API
type ClaudeResponse struct {
	ID         string               `json:"id"`
	Type       string               `json:"type"`
	Role       string               `json:"role"`
	Content    []ClaudeContentBlock `json:"content"`
	StopReason string               `json:"stop_reason"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Usage *ClaudeUsage `json:"usage,omitempty"`
}

// ClaudeUsage captures token accounting, including prompt cache activity.
type ClaudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// ResponsesAPIRequest represents the request body for Responses API
type ResponsesAPIRequest struct {
	Model        string           `json:"model"`
	Input        interface{}      `json:"input"` // Can be string or []map[string]interface{}
	Instructions string           `json:"instructions,omitempty"`
	Store        *bool            `json:"store,omitempty"`
	Tools        []interface{}    `json:"tools,omitempty"` // Tools like web_search
	Reasoning    *ReasoningConfig `json:"reasoning,omitempty"`
}

// ReasoningConfig controls reasoning behavior for models that support it
type ReasoningConfig struct {
	Effort string `json:"effort"` // "none", "low", "medium", "high"
}

// WebSearchTool represents the web_search tool configuration
type WebSearchTool struct {
	Type string `json:"type"` // "web_search" or "web_search_preview" depending on API version
}

// ResponsesAPIResponse represents the response from Responses API
type ResponsesAPIResponse struct {
	ID         string                   `json:"id"`
	OutputText string                   `json:"output_text"`
	Output     interface{}              `json:"output,omitempty"` // Can be string or array of items
	Items      []map[string]interface{} `json:"items,omitempty"`
	Error      *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// OpenRouterChatRequest represents OpenRouter's OpenAI-compatible chat request.
type OpenRouterChatRequest struct {
	Model     string              `json:"model"`
	Messages  []OpenRouterMessage `json:"messages"`
	MaxTokens int                 `json:"max_tokens,omitempty"`
	Tools     []OpenRouterTool    `json:"tools,omitempty"`
}

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterTool struct {
	Type string `json:"type"`
}

type OpenRouterChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string      `json:"message"`
		Type    string      `json:"type,omitempty"`
		Code    interface{} `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize Secret Manager client
	ctx := context.Background()
	secretClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create secret manager client: %v", err)
	}
	defer secretClient.Close()

	apiKeySecret, err := loadRequiredSecret(ctx, secretClient, "API_KEY_SECRET_NAME", "API_SECRET_NAME", "API key secret")
	if err != nil {
		log.Fatal(err)
	}

	openaiKey := loadOptionalSecret(ctx, secretClient, "OPENAI_KEY_SECRET_NAME", "OPENAI_SECRET_NAME", "OpenAI Responses API")
	openRouterKey := loadOptionalSecret(ctx, secretClient, "OPENROUTER_KEY_SECRET_NAME", "OPENROUTER_SECRET_NAME", "OpenRouter API")
	perplexityKey := loadOptionalSecret(ctx, secretClient, "PERPLEXITY_KEY_SECRET_NAME", "PERPLEXITY_SECRET_NAME", "Perplexity Search API")
	claudeKey := loadOptionalSecret(ctx, secretClient, "CLAUDE_KEY_SECRET_NAME", "CLAUDE_SECRET_NAME", "Claude API")
	if claudeKey == "" && openaiKey == "" && openRouterKey == "" {
		log.Fatal("No AI provider configured. Set CLAUDE_KEY_SECRET_NAME, OPENROUTER_KEY_SECRET_NAME, or OPENAI_KEY_SECRET_NAME with a direct key value, or set the matching *_SECRET_NAME for Secret Manager lookup.")
	}
	if claudeKey != "" {
		log.Printf("Claude API enabled - direct Haiku responses available")
	}
	if openRouterKey != "" {
		log.Printf("OpenRouter API enabled - OpenAI-compatible fallback available")
	}
	if openaiKey != "" {
		log.Printf("OpenAI Responses API enabled - native web_search fallback available")
	}

	// Initialize logger
	logger := logging.GetLogger()

	server := &Server{
		openaiAPIKey:     openaiKey,
		openRouterAPIKey: openRouterKey,
		perplexityAPIKey: perplexityKey,
		claudeAPIKey:     claudeKey,
		apiKeySecret:     apiKeySecret,
		logger:           logger,
	}

	// Setup middleware chain
	mux := http.NewServeMux()
	mux.HandleFunc("/chat", server.handleChat)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/", handleOptions) // CORS preflight for root

	// Register API config endpoint (protected by X-API-Key auth)
	mux.HandleFunc("/api/config", server.handleConfigAPI)

	// Register admin routes (protected by HTTP Basic Auth)
	// Always register routes - BasicAuthMiddleware will handle the case when admin is not configured
	// This prevents 404 errors and provides better user feedback
	adminHandler := admin.NewHandler(logger)
	adminHandler.RegisterRoutes(mux)
	if adminHandler.IsEnabled() {
		log.Printf("Admin dashboard enabled at /admin/")
	} else {
		log.Printf("Admin dashboard routes registered but disabled (ADMIN_USER and ADMIN_PASSWORD not set)")
	}

	// Initialize default runtime configuration with the base system prompt template
	admin.SetDefaultConfig(clotildeBaseSystemPromptTemplate)

	// Initialize default category prompts for UI display
	defaultCategoryPrompts := map[string]string{
		"web_search":   categoryPromptWebSearch,
		"complex":      categoryPromptComplex,
		"factual":      categoryPromptFactual,
		"mathematical": categoryPromptMathematical,
		"creative":     categoryPromptCreative,
	}
	admin.SetDefaultCategoryPrompts(defaultCategoryPrompts)

	// Middleware order (execution order when request arrives):
	// 1. PreAuth: IP-based rate limiting BEFORE authentication (prevents brute force)
	// 2. RequestID: Adds unique request ID for tracing
	// 3. Validator: Limits request size early (prevents large payloads)
	// 4. Auth: Validates API key
	// 5. RateLimit: Rate-limits using VALIDATED API keys (prevents bypass attacks)
	//
	// Note: In Go middleware wrapping, the last wrapped executes first.
	// So we wrap in reverse order: RateLimit → Auth → PreAuth → Validator → RequestID → Mux
	// Execution Order: RequestID → Validator → PreAuth → Auth → RateLimit
	handler := ratelimit.Middleware()(mux)           // Uses validated API key from context (runs LAST)
	handler = auth.Middleware(apiKeySecret)(handler) // Validates API key, sets context
	handler = ratelimit.PreAuthMiddleware()(handler) // IP-based, runs BEFORE auth
	handler = validator.Middleware()(handler)        // Limits request size early
	handler = logging.RequestIDMiddleware(handler)   // Adds ID first (runs FIRST)

	serverAddr := fmt.Sprintf(":%s", port)

	// Create HTTP server with graceful shutdown and timeouts
	// These timeouts protect against slow clients and ensure requests complete within Apple Shortcuts limits
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second, // Time to read request body
		WriteTimeout: 30 * time.Second, // Time to write response (matches Apple Shortcuts limit)
		IdleTimeout:  60 * time.Second, // Keep-alive timeout
	}

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Flush Cloud Logging before shutdown
	cloudLogger := logging.GetCloudLogger()
	if cloudLogger.IsEnabled() {
		log.Println("Flushing Cloud Logging...")
		if err := cloudLogger.Flush(); err != nil {
			log.Printf("Error flushing Cloud Logging: %v", err)
		}
		// Give it a moment to send
		time.Sleep(2 * time.Second)
	}
	logging.StopPeriodicFlush()

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close Cloud Logging client
	if cloudLogger.IsEnabled() {
		if err := cloudLogger.Close(); err != nil {
			log.Printf("Error closing Cloud Logging client: %v", err)
		}
	}

	log.Println("Server exited")
}

func getSecret(ctx context.Context, client *secretmanager.Client, projectID, secretName string) (string, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName)
	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}
	return string(result.Payload.Data), nil
}

// loadRequiredSecret prefers a direct value for local development and falls back
// to the named Secret Manager secret when the direct value is not set.
func loadRequiredSecret(ctx context.Context, client *secretmanager.Client, valueEnv, secretNameEnv, label string) (string, error) {
	if value := os.Getenv(valueEnv); value != "" {
		return value, nil
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return "", fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}

	secretName := os.Getenv(secretNameEnv)
	if secretName == "" {
		return "", fmt.Errorf("%s environment variable not set (required for Secret Manager lookup)", secretNameEnv)
	}

	value, err := getSecret(ctx, client, projectID, secretName)
	if err != nil {
		return "", fmt.Errorf("failed to get %s: %w", label, err)
	}
	return value, nil
}

// loadOptionalSecret follows the same precedence as loadRequiredSecret, but it
// quietly disables the feature when neither a direct value nor a secret name is configured.
func loadOptionalSecret(ctx context.Context, client *secretmanager.Client, valueEnv, secretNameEnv, feature string) string {
	if value := os.Getenv(valueEnv); value != "" {
		return value
	}

	secretName := os.Getenv(secretNameEnv)
	if secretName == "" {
		log.Printf("%s not set - %s will be disabled", secretNameEnv, feature)
		return ""
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Printf("%s set but GOOGLE_CLOUD_PROJECT not set - %s will be disabled", secretNameEnv, feature)
		return ""
	}

	value, err := getSecret(ctx, client, projectID, secretName)
	if err != nil {
		log.Printf("Failed to get %s key: %v - %s will be disabled", feature, err, feature)
		return ""
	}
	return value
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	stats := s.logger.GetStats()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	response := map[string]interface{}{
		"status":            "ok",
		"uptime":            time.Since(startTime).Round(time.Second).String(),
		"total_requests":    stats.TotalRequests,
		"memory_mb":         memStats.Alloc / 1024 / 1024,
		"last_request_time": stats.LastRequestTime,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func handleOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		setCORSHeaders(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		setCORSHeaders(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Start timing for logging
	startTime := time.Now()

	// Get request ID from context (added by middleware)
	requestID := logging.GetRequestID(r.Context())
	if requestID == "" {
		requestID = logging.GenerateRequestID()
	}

	// Add request ID to response headers
	w.Header().Set("X-Request-ID", requestID)

	// Note: We don't strictly validate Content-Type because Apple Shortcuts
	// sometimes sends text/plain even when the body is valid JSON.
	// The JSON decoder will fail if the body isn't valid JSON anyway.

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logRequest(requestID, r, "", "", "", "", "", time.Since(startTime), "error", "Invalid request body")
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		s.logRequest(requestID, r, "", "", "", "", "", time.Since(startTime), "error", "Message is required")
		respondError(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Sanitize input to prevent prompt injection attacks (OWASP LLM Top 10 A1)
	sanitizedMessage, err := promptinjection.ValidateInput(req.Message)
	if err != nil {
		s.logRequest(requestID, r, req.Message, "", "", "", "", time.Since(startTime), "error", "Invalid input: "+err.Error())
		respondError(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Log if prompt injection was detected (for monitoring)
	if sanitizedMessage != req.Message {
		log.Printf("[%s] Prompt injection detected and neutralized: IP=%s", requestID, hashIP(clientip.FromRequest(r)))
	}

	// Log request metadata (no sensitive data)
	log.Printf("[%s] Request received: IP=%s, MessageLength=%d", requestID, hashIP(clientip.FromRequest(r)), len(sanitizedMessage))

	// Route to appropriate model and determine if web search is needed
	// Use sanitized message for routing to prevent injection via routing logic
	route := router.Route(sanitizedMessage)
	log.Printf("[%s] Route decision: Category=%s, Model=%s, WebSearch=%v", requestID, route.Category, route.Model, route.WebSearch)

	// Call the configured AI provider with the selected model and tools
	// IMPORTANT: Apple Shortcuts has ~30s internal timeout. We use 25s to leave buffer
	// for network latency and response processing on the client side.
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// Get current date/time in Brazil timezone for context
	currentTime := getCurrentBrazilTime()
	// Get dynamic system prompt from runtime config with category-specific override
	config := admin.GetConfig()
	systemPrompt := s.buildSystemPrompt(config, route.Category, currentTime)

	// Convert router.RouteDecision to internal RouteDecision format
	internalRoute := RouteDecision{
		Model:           route.Model,
		WebSearch:       route.WebSearch,
		ReasoningEffort: route.ReasoningEffort,
	}
	// Use sanitized message to prevent prompt injection
	response, err := s.createResponse(ctx, internalRoute, systemPrompt, sanitizedMessage)
	if err != nil {
		log.Printf("[%s] AI provider error: %v", requestID, err)
		// Log the raw message for auditability, but use sanitized for API calls
		s.logRequest(requestID, r, req.Message, sanitizedMessage, "", route.Model, string(route.Category), time.Since(startTime), "error", err.Error())

		// Check if it's a timeout error and provide friendly message
		if ctx.Err() == context.DeadlineExceeded || strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "timeout") {
			// Provide a helpful response for timeouts - spoken via CarPlay
			respondSuccess(w, "Desculpe, a pergunta demorou demais para processar. Tente uma pergunta mais simples ou tente novamente.")
			return
		}
		respondError(w, "Failed to get response from AI", http.StatusInternalServerError)
		return
	}

	if response == "" {
		response = "Desculpe, não consegui processar sua solicitação. Pode repetir?"
	}

	// Log successful request
	responseTime := time.Since(startTime)
	log.Printf("[%s] Response generated: Length=%d, Time=%v", requestID, len(response), responseTime)
	// Log the raw message for auditability while preserving the sanitized form for processing
	s.logRequest(requestID, r, req.Message, sanitizedMessage, response, route.Model, string(route.Category), responseTime, "success", "")

	respondSuccess(w, response)
}

// logRequest adds a structured log entry with raw and sanitized input for Cloud Logging
func (s *Server) logRequest(requestID string, r *http.Request, rawInput, sanitizedInput, output, model, category string, responseTime time.Duration, status, errorMsg string) {
	// Apply PII redaction if enabled
	loggedRawInput := rawInput
	loggedInput := sanitizedInput
	loggedOutput := output
	if logging.IsRedactPIIEnabled() {
		loggedRawInput = logging.RedactPII(rawInput)
		loggedInput = logging.RedactPII(sanitizedInput)
		loggedOutput = logging.RedactPII(output)
	}

	// Check if full content logging is enabled
	// If disabled, only log metadata (lengths, hashes, etc.)
	var finalRawInput, finalInput, finalOutput string
	if logging.ShouldLogFullContent() {
		finalRawInput = loggedRawInput
		finalInput = loggedInput
		finalOutput = loggedOutput
	} else {
		// Full content logging disabled - only log metadata
		// Content fields will be empty, but lengths are preserved
		finalRawInput = ""
		finalInput = ""
		finalOutput = ""
	}

	entry := logging.LogEntry{
		ID:            requestID,
		Timestamp:     time.Now(),
		IPHash:        hashIP(clientip.FromRequest(r)),
		MessageLength: len(rawInput), // Always log original length, even if content is redacted
		Model:         model,
		Category:      category,
		ResponseTime:  responseTime.Milliseconds(),
		TokenEstimate: len(rawInput) / 4, // Rough estimate: ~4 chars per token
		Status:        status,
		ErrorMessage:  errorMsg,
		RawInput:      finalRawInput,
		Input:         finalInput,
		Output:        finalOutput,
	}
	s.logger.Add(entry)
}

var (
	// Compile regular expressions once at package level
	markdownLinkInParensRegexp = regexp.MustCompile(`\(\[[^\]]+\]\([^\)]+\)\)`)
	markdownLinkRegexp         = regexp.MustCompile(`\[[^\]]+\]\([^\)]+\)`)
	urlRegexp                  = regexp.MustCompile(`(?i)(https?://|www\.)[^\s]+`)
	domainRegexp               = regexp.MustCompile(`(?i)\b[a-z0-9]+([.-][a-z0-9]+)*\.(com|br|org|net|gov|edu|io|co|info|me|tv|xyz)[^\s]*`)
	spaceRegexp                = regexp.MustCompile(`\s+`)
)

// removeURLsFromText removes any URLs, web addresses, or domain names from text
// This is a safety net to ensure no URLs make it to the voice interface
func removeURLsFromText(text string) string {
	// Remove markdown links: [text](url) or ([text](url))
	// First, remove markdown links wrapped in parentheses: ([text](url))
	text = markdownLinkInParensRegexp.ReplaceAllString(text, "")
	// Then remove standard markdown links: [text](url)
	text = markdownLinkRegexp.ReplaceAllString(text, "")

	// Remove URLs (http://, https://, www.)
	text = urlRegexp.ReplaceAllString(text, "")

	// Remove domain patterns like "example.com" or "g1.com.br"
	text = domainRegexp.ReplaceAllString(text, "")

	// Remove phrases that might lead to URLs
	text = strings.ReplaceAll(text, "você pode ver em", "")
	text = strings.ReplaceAll(text, "acesse", "")
	text = strings.ReplaceAll(text, "visite", "")
	text = strings.ReplaceAll(text, "veja em", "")

	// Clean up extra spaces and empty parentheses
	text = strings.ReplaceAll(text, "()", "")
	text = strings.ReplaceAll(text, "( )", "")
	text = spaceRegexp.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

func respondSuccess(w http.ResponseWriter, response string) {
	// Remove any URLs that might have escaped the system prompt
	response = removeURLsFromText(response)

	w.Header().Set("Content-Type", "application/json")
	// CORS restricted to Apple Shortcuts origin for security
	setCORSHeaders(w)
	json.NewEncoder(w).Encode(ChatResponse{Response: response})
}

func respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	setCORSHeaders(w)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ChatResponse{Error: message})
}

func setCORSHeaders(w http.ResponseWriter) {
	// CORS configuration for API access
	// Apple Shortcuts doesn't need CORS (not browser-based), but we allow it
	// for potential web clients or testing tools
	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		// Default: no CORS (don't set Access-Control-Allow-Origin)
		// This is the safest default - set CORS_ALLOWED_ORIGIN env var if needed
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
	w.Header().Set("Access-Control-Max-Age", "3600")
}

var (
	ipHashSalt     string
	ipHashSaltOnce sync.Once
)

// getIPHashSalt returns the salt for IP hashing, loading it once from environment variable
// In production (Cloud Run), IP_HASH_SALT MUST be set or the application will fail to start
func getIPHashSalt() string {
	ipHashSaltOnce.Do(func() {
		ipHashSalt = os.Getenv("IP_HASH_SALT")

		// Check if running in production (Cloud Run)
		// Cloud Run sets GOOGLE_CLOUD_PROJECT, K_SERVICE, and K_REVISION
		isProduction := os.Getenv("GOOGLE_CLOUD_PROJECT") != "" &&
			(os.Getenv("K_SERVICE") != "" || os.Getenv("K_REVISION") != "")

		if ipHashSalt == "" {
			if isProduction {
				// Production: fail to start if salt is not configured
				log.Fatal("IP_HASH_SALT environment variable is required in production but is not set. " +
					"Set IP_HASH_SALT to a cryptographically secure random string (e.g., 32+ characters).")
			} else {
				// Development: log severe warning but allow to continue
				log.Printf("WARNING: IP_HASH_SALT is not set. Using a weak default salt. " +
					"This is INSECURE and should NEVER be used in production. " +
					"Set IP_HASH_SALT environment variable to a secure random string.")
				ipHashSalt = "clotilde-ip-hash-salt-default-INSECURE-DEVELOPMENT-ONLY"
			}
		} else if len(ipHashSalt) < 16 {
			// Warn if salt is too short
			log.Printf("WARNING: IP_HASH_SALT is too short (%d characters). "+
				"Recommend using at least 32 characters for better security.", len(ipHashSalt))
		}
	})
	return ipHashSalt
}

func hashIP(ip string) string {
	// Cryptographically secure IP hashing using SHA-256 with salt
	// This prevents rainbow table attacks and makes it difficult to reverse hashes
	// The salt is loaded from IP_HASH_SALT environment variable (or uses default)
	salt := getIPHashSalt()

	// Hash IP with salt using SHA-256
	hasher := sha256.New()
	hasher.Write([]byte(salt + ip))
	hash := hasher.Sum(nil)

	// Return hex-encoded hash with prefix for identification
	return fmt.Sprintf("ip_%s", hex.EncodeToString(hash[:16])) // Use first 16 bytes (128 bits) for shorter hash
}

// getCurrentBrazilTime returns current date and time in Brazil/São Paulo timezone
func getCurrentBrazilTime() string {
	loc, err := time.LoadLocation(timezoneBR)
	if err != nil {
		// Fallback to UTC if timezone loading fails
		loc = time.UTC
	}
	now := time.Now().In(loc)

	// Format date in Portuguese
	months := map[time.Month]string{
		time.January:   "janeiro",
		time.February:  "fevereiro",
		time.March:     "março",
		time.April:     "abril",
		time.May:       "maio",
		time.June:      "junho",
		time.July:      "julho",
		time.August:    "agosto",
		time.September: "setembro",
		time.October:   "outubro",
		time.November:  "novembro",
		time.December:  "dezembro",
	}

	monthName := months[now.Month()]
	return fmt.Sprintf("%02d de %s de %d, %02d:%02d (horário de Brasília)",
		now.Day(), monthName, now.Year(), now.Hour(), now.Minute())
}

// PerplexitySearchRequest represents the request body for Perplexity Search API
type PerplexitySearchRequest struct {
	Query                string   `json:"query"`
	MaxResults           int      `json:"max_results,omitempty"`
	MaxTokensPerPage     int      `json:"max_tokens_per_page,omitempty"`
	SearchLanguageFilter []string `json:"search_language_filter,omitempty"`
}

// PerplexitySearchResponse represents the response from Perplexity Search API
type PerplexitySearchResponse struct {
	Results []PerplexitySearchResult `json:"results"`
}

// PerplexitySearchResult represents a single search result from Perplexity
type PerplexitySearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
	Date        string `json:"date,omitempty"`
	LastUpdated string `json:"last_updated,omitempty"`
}

// performPerplexitySearch calls the Perplexity Search API to get web search results
func (s *Server) performPerplexitySearch(ctx context.Context, query string) ([]PerplexitySearchResult, error) {
	if s.perplexityAPIKey == "" {
		return nil, fmt.Errorf("Perplexity API key not configured")
	}

	// Build request body
	reqBody := PerplexitySearchRequest{
		Query:            query,
		MaxResults:       5,    // Default to 5 results
		MaxTokensPerPage: 1024, // Default token limit per page
	}

	// Determine language filter based on query (Portuguese for Brazilian queries)
	// Simple heuristic: if query contains Portuguese words, use Portuguese filter
	if strings.Contains(strings.ToLower(query), "hoje") ||
		strings.Contains(strings.ToLower(query), "notícias") ||
		strings.Contains(strings.ToLower(query), "brasil") {
		reqBody.SearchLanguageFilter = []string{"pt"}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Perplexity request: %w", err)
	}

	// Create HTTP request to Perplexity Search API
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.perplexity.ai/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create Perplexity request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.perplexityAPIKey))

	// Make HTTP request
	// Use 8s timeout for Perplexity to leave time for OpenAI call within 25s total budget
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make Perplexity request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Perplexity response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Perplexity API returned status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("Perplexity API returned status %d", resp.StatusCode)
	}

	// Parse response
	var apiResp PerplexitySearchResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("Failed to parse Perplexity response body: %s", string(body))
		return nil, fmt.Errorf("failed to parse Perplexity response: %w", err)
	}

	return apiResp.Results, nil
}

// formatPerplexityResults formats Perplexity search results into a readable context string
func formatPerplexityResults(results []PerplexitySearchResult) string {
	if len(results) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("The following web search results were retrieved using Perplexity AI:\n\n")

	for i, result := range results {
		builder.WriteString(fmt.Sprintf("[%s] (source: %s)\n", result.Title, result.URL))
		if result.Snippet != "" {
			builder.WriteString(result.Snippet)
			builder.WriteString("\n")
		}
		if i < len(results)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// createResponse routes to the appropriate AI provider.
// Claude Haiku is preferred for speed-critical CarPlay scenarios, with
// OpenRouter and OpenAI used as configured fallbacks.
func (s *Server) createResponse(ctx context.Context, route RouteDecision, instructions, input string) (string, error) {
	// Get current config to check Perplexity setting
	config := admin.GetConfig()
	route = s.routeForAvailableProvider(route)

	if isOpenRouterModel(route.Model) {
		log.Printf("Using OpenRouter API: model=%s, web_search=%v", route.Model, route.WebSearch)
		return s.makeOpenRouterRequest(ctx, route.Model, instructions, input, route.WebSearch)
	}

	// Check if this is a Claude model
	if isClaudeModel(route.Model) && s.claudeAPIKey != "" {
		// CRITICAL: If web search is needed, use Claude's native search first to get real-time data
		// We should NEVER rely on training data for recent information
		if route.WebSearch {
			log.Printf("Using Claude native web_search tool for real-time data")
			response, err := s.makeClaudeWebSearchRequest(ctx, route.Model, instructions, input)
			if err == nil {
				return response, nil
			}
			log.Printf("Claude native web_search failed: %v", err)

			if config.PerplexityEnabled && s.perplexityAPIKey != "" {
				log.Printf("Using Perplexity Search API with Claude for web search (real-time data required)")
				perplexityResults, err := s.performPerplexitySearch(ctx, input)
				if err != nil {
					log.Printf("Perplexity search failed: %v", err)
					return s.makeFallbackWebSearchRequest(ctx, route, instructions, input)
				} else {
					// Format Perplexity results and append to instructions
					formattedResults := formatPerplexityResults(perplexityResults)
					if formattedResults != "" {
						instructions = fmt.Sprintf("%s\n\n%s", instructions, formattedResults)
						log.Printf("Perplexity results appended to Claude instructions for real-time data")
					}
				}
			} else {
				log.Printf("Web search needed but Perplexity is disabled or not configured")
				return s.makeFallbackWebSearchRequest(ctx, route, instructions, input)
			}
		}
		log.Printf("Using Claude API for fast response: model=%s", route.Model)
		return s.makeClaudeRequest(ctx, route.Model, instructions, input)
	}

	if s.openaiAPIKey == "" {
		return "", fmt.Errorf("no AI provider key configured for model %s", route.Model)
	}

	// Build request body for Responses API
	store := true // Enable logging so usage appears in OpenAI logs

	// Handle web search: use Perplexity if enabled, otherwise use OpenAI's web_search tool
	if route.WebSearch {
		if config.PerplexityEnabled && s.perplexityAPIKey != "" {
			// Use Perplexity Search API
			log.Printf("Using Perplexity Search API for web search")
			perplexityResults, err := s.performPerplexitySearch(ctx, input)
			if err != nil {
				log.Printf("Perplexity search failed: %v, falling back to OpenAI web_search", err)
				return s.makeOpenAIWebSearchRequest(ctx, route, instructions, input)
			}

			// Format Perplexity results and append to instructions
			formattedResults := formatPerplexityResults(perplexityResults)
			enhancedInstructions := instructions
			if formattedResults != "" {
				enhancedInstructions = fmt.Sprintf("%s\n\n%s", instructions, formattedResults)
			}

			// Create request without web_search tool (using Perplexity results in instructions)
			reqBody := ResponsesAPIRequest{
				Model:        route.Model,
				Input:        input,
				Instructions: enhancedInstructions,
				Store:        &store,
			}
			return s.makeOpenAIRequest(ctx, reqBody, route)
		} else {
			// Use OpenAI's native web_search tool
			log.Printf("Using OpenAI web_search tool for web search")
			webSearchTool := WebSearchTool{Type: "web_search"}
			reqBody := ResponsesAPIRequest{
				Model:        route.Model,
				Input:        input,
				Instructions: instructions,
				Store:        &store,
				Tools:        []interface{}{webSearchTool},
			}
			return s.makeOpenAIRequest(ctx, reqBody, route)
		}
	}

	// No web search needed - create standard request
	reqBody := ResponsesAPIRequest{
		Model:        route.Model,
		Input:        input,
		Instructions: instructions,
		Store:        &store,
	}
	return s.makeOpenAIRequest(ctx, reqBody, route)
}

func (s *Server) makeFallbackWebSearchRequest(ctx context.Context, route RouteDecision, instructions, input string) (string, error) {
	if s.openaiAPIKey != "" {
		log.Printf("Using OpenAI web_search fallback")
		return s.makeOpenAIWebSearchRequest(ctx, route, instructions, input)
	}
	if s.openRouterAPIKey != "" {
		log.Printf("Using OpenRouter web_search fallback")
		return s.makeOpenRouterRequest(ctx, openRouterClaudeHaikuModel, instructions, input, true)
	}
	return "", fmt.Errorf("web search is required but no fallback web search provider is configured")
}

func (s *Server) makeOpenAIWebSearchRequest(ctx context.Context, route RouteDecision, instructions, input string) (string, error) {
	store := true
	reqBody, fallbackRoute := buildOpenAIWebSearchRequest(route, instructions, input, &store)
	return s.makeOpenAIRequest(ctx, reqBody, fallbackRoute)
}

func buildOpenAIWebSearchRequest(route RouteDecision, instructions, input string, store *bool) (ResponsesAPIRequest, RouteDecision) {
	fallbackRoute := route
	if isClaudeModel(fallbackRoute.Model) {
		log.Printf("Claude model %s cannot use OpenAI web_search; using fallback model %s", fallbackRoute.Model, openAIFallbackModel)
		fallbackRoute.Model = openAIFallbackModel
		fallbackRoute.ReasoningEffort = ""
	}

	webSearchTool := WebSearchTool{Type: "web_search"}
	reqBody := ResponsesAPIRequest{
		Model:        fallbackRoute.Model,
		Input:        input,
		Instructions: instructions,
		Store:        store,
		Tools:        []interface{}{webSearchTool},
	}

	return reqBody, fallbackRoute
}

func (s *Server) routeForAvailableProvider(route RouteDecision) RouteDecision {
	switch {
	case isClaudeModel(route.Model) && s.claudeAPIKey != "":
		return route
	case isOpenRouterModel(route.Model) && s.openRouterAPIKey != "":
		return route
	case isOpenAIModel(route.Model) && s.openaiAPIKey != "":
		return route
	}

	switch {
	case isClaudeModel(route.Model) && s.openRouterAPIKey != "":
		log.Printf("Claude model %s selected but Claude API key is not configured; using OpenRouter Haiku fallback %s", route.Model, openRouterClaudeHaikuModel)
		route.Model = openRouterClaudeHaikuModel
		route.ReasoningEffort = ""
	case isOpenRouterModel(route.Model) && s.claudeAPIKey != "":
		log.Printf("OpenRouter model %s selected but OpenRouter API key is not configured; using direct Claude Haiku fallback %s", route.Model, defaultClaudeHaikuModel)
		route.Model = defaultClaudeHaikuModel
		route.ReasoningEffort = ""
	case s.openRouterAPIKey != "":
		log.Printf("Model %s selected but its provider is not configured; using OpenRouter Haiku fallback %s", route.Model, openRouterClaudeHaikuModel)
		route.Model = openRouterClaudeHaikuModel
		route.ReasoningEffort = ""
	case s.claudeAPIKey != "":
		log.Printf("Model %s selected but its provider is not configured; using direct Claude Haiku fallback %s", route.Model, defaultClaudeHaikuModel)
		route.Model = defaultClaudeHaikuModel
		route.ReasoningEffort = ""
	case s.openaiAPIKey != "":
		log.Printf("Model %s selected but its provider is not configured; using OpenAI fallback %s", route.Model, openAIFallbackModel)
		route.Model = openAIFallbackModel
		route.ReasoningEffort = ""
	}
	return route
}

// makeOpenRouterRequest calls OpenRouter's OpenAI-compatible Chat Completions API.
func (s *Server) makeOpenRouterRequest(ctx context.Context, model, systemPrompt, userMessage string, webSearch bool) (string, error) {
	if s.openRouterAPIKey == "" {
		return "", fmt.Errorf("OpenRouter API key not configured")
	}

	reqBody := buildOpenRouterChatRequest(model, systemPrompt, userMessage, webSearch)
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenRouter request: %w", err)
	}

	log.Printf("OpenRouter request: model=%s, max_tokens=%d, web_search=%v", reqBody.Model, reqBody.MaxTokens, webSearch)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenRouter request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.openRouterAPIKey))
	if siteURL := os.Getenv("OPENROUTER_SITE_URL"); siteURL != "" {
		httpReq.Header.Set("HTTP-Referer", siteURL)
	}
	title := os.Getenv("OPENROUTER_APP_TITLE")
	if title == "" {
		title = "Clotilde CarPlay Assistant"
	}
	httpReq.Header.Set("X-OpenRouter-Title", title)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to make OpenRouter request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OpenRouter response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("OpenRouter API returned status %d: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("OpenRouter API returned status %d", resp.StatusCode)
	}

	var apiResp OpenRouterChatResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("Failed to parse OpenRouter response body: %s", string(body))
		return "", fmt.Errorf("failed to parse OpenRouter response: %w", err)
	}
	if apiResp.Error != nil {
		return "", fmt.Errorf("OpenRouter API error: %s", apiResp.Error.Message)
	}
	for _, choice := range apiResp.Choices {
		if choice.Message.Content != "" {
			return choice.Message.Content, nil
		}
	}

	log.Printf("Empty response from OpenRouter. Full response: %s", string(body))
	return "", fmt.Errorf("empty response from OpenRouter")
}

func buildOpenRouterChatRequest(model, systemPrompt, userMessage string, webSearch bool) OpenRouterChatRequest {
	reqBody := OpenRouterChatRequest{
		Model:     openRouterModelID(model),
		MaxTokens: 500,
		Messages: []OpenRouterMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
	}
	if webSearch {
		reqBody.Tools = []OpenRouterTool{{Type: "openrouter:web_search"}}
	}
	return reqBody
}

// makeOpenAIRequest makes the actual HTTP request to OpenAI Responses API
func (s *Server) makeOpenAIRequest(ctx context.Context, reqBody ResponsesAPIRequest, route RouteDecision) (string, error) {
	// Set reasoning effort only for models that support it (o1, o3, gpt-5 series)
	// Models like gpt-4o, gpt-4-turbo don't support reasoning parameter
	// IMPORTANT: gpt-5 requires reasoning >= "low" for web search to work
	// According to OpenAI docs: "Web search is currently not supported in gpt-5 with minimal reasoning"
	// Note: This only applies when using OpenAI's web_search tool, not when using Perplexity
	if modelSupportsReasoning(route.Model) {
		reasoningEffort := route.ReasoningEffort
		// Check if web_search tool is being used (not Perplexity)
		usingWebSearchTool := false
		if len(reqBody.Tools) > 0 {
			// Check if any tool is a web_search tool
			for _, tool := range reqBody.Tools {
				if toolMap, ok := tool.(map[string]interface{}); ok {
					if toolType, ok := toolMap["type"].(string); ok && toolType == "web_search" {
						usingWebSearchTool = true
						break
					}
				} else if toolStruct, ok := tool.(WebSearchTool); ok && toolStruct.Type == "web_search" {
					usingWebSearchTool = true
					break
				}
			}
		}

		// If using gpt-5 with OpenAI's web_search tool, must use at least "medium" reasoning
		if strings.HasPrefix(route.Model, "gpt-5") && route.WebSearch && usingWebSearchTool {
			if reasoningEffort == "" || reasoningEffort == "none" {
				reasoningEffort = "medium" // Minimum required for web search
				log.Printf("gpt-5 with web search: using reasoning='medium' (minimum required)")
			}
		}
		if reasoningEffort != "" && reasoningEffort != "none" {
			reqBody.Reasoning = &ReasoningConfig{Effort: reasoningEffort}
			log.Printf("Reasoning effort: %s", reasoningEffort)
		}
	}

	// Ensure Store is always set to true for logging
	if reqBody.Store == nil {
		store := true
		reqBody.Store = &store
	} else {
		*reqBody.Store = true // Force to true to ensure logging is enabled
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log request details (without sensitive data) for debugging
	log.Printf("OpenAI Responses API request: model=%s, store=%v, has_tools=%v",
		reqBody.Model, reqBody.Store != nil && *reqBody.Store, len(reqBody.Tools) > 0)

	// Create HTTP request to Responses API
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.openaiAPIKey))

	// Make HTTP request
	// Use 20s timeout for OpenAI to fit within 25s total budget (leaves buffer for processing)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		log.Printf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var apiResp ResponsesAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("Failed to parse response body: %s", string(body))
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API-level errors
	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s (type: %s)", apiResp.Error.Message, apiResp.Error.Type)
	}

	// Responses API returns output as an array of items
	// Structure: output[0].content[0].text (for message type items)
	if apiResp.Output != nil {
		if outputArr, ok := apiResp.Output.([]interface{}); ok {
			for _, item := range outputArr {
				if itemMap, ok := item.(map[string]interface{}); ok {
					// Look for message type items
					if itemType, ok := itemMap["type"].(string); ok && itemType == "message" {
						// Content is an array of content items
						if contentArr, ok := itemMap["content"].([]interface{}); ok {
							for _, contentItem := range contentArr {
								if contentMap, ok := contentItem.(map[string]interface{}); ok {
									// Look for output_text type content
									if contentType, ok := contentMap["type"].(string); ok && contentType == "output_text" {
										if text, ok := contentMap["text"].(string); ok && text != "" {
											return text, nil
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Fallback: try output_text field (SDK-only convenience property, may not be in raw API response)
	if apiResp.OutputText != "" {
		return apiResp.OutputText, nil
	}

	log.Printf("Empty response from API. Full response: %s", string(body))
	return "", fmt.Errorf("empty response from API")
}

// makeClaudeRequest makes a request to Claude Messages API (Anthropic)
// Claude Haiku 4.5 is extremely fast (~1-3s) and ideal for CarPlay where speed is critical
func (s *Server) makeClaudeRequest(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	return s.makeClaudeRequestWithTools(ctx, model, systemPrompt, userMessage, nil)
}

func (s *Server) makeClaudeWebSearchRequest(ctx context.Context, model, systemPrompt, userMessage string) (string, error) {
	tools := []ClaudeTool{
		{
			Type:    "web_search_20250305",
			Name:    "web_search",
			MaxUses: 3,
		},
	}
	return s.makeClaudeRequestWithTools(ctx, model, systemPrompt, userMessage, tools)
}

func (s *Server) makeClaudeRequestWithTools(ctx context.Context, model, systemPrompt, userMessage string, tools []ClaudeTool) (string, error) {
	if s.claudeAPIKey == "" {
		return "", fmt.Errorf("Claude API key not configured")
	}

	// Build request body for Claude Messages API
	reqBody := buildClaudeRequest(model, systemPrompt, userMessage, tools)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Claude request: %w", err)
	}

	log.Printf("Claude API request: model=%s, max_tokens=%d, has_tools=%v", model, reqBody.MaxTokens, len(reqBody.Tools) > 0)

	// Create HTTP request to Claude Messages API
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create Claude request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", s.claudeAPIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Use 15s timeout for Claude (it's very fast, Haiku typically responds in 1-3s)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to make Claude request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Claude response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Claude API returned status %d: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("Claude API returned status %d", resp.StatusCode)
	}

	// Parse response
	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		log.Printf("Failed to parse Claude response body: %s", string(body))
		return "", fmt.Errorf("failed to parse Claude response: %w", err)
	}

	// Check for API-level errors
	if claudeResp.Error != nil {
		return "", fmt.Errorf("Claude API error: %s (type: %s)", claudeResp.Error.Message, claudeResp.Error.Type)
	}

	if claudeResp.Usage != nil {
		log.Printf("Claude usage: input_tokens=%d, output_tokens=%d, cache_creation_input_tokens=%d, cache_read_input_tokens=%d",
			claudeResp.Usage.InputTokens,
			claudeResp.Usage.OutputTokens,
			claudeResp.Usage.CacheCreationInputTokens,
			claudeResp.Usage.CacheReadInputTokens,
		)
	}

	if err := claudeWebSearchError(claudeResp); err != nil {
		return "", err
	}

	// Extract text from response content
	for _, content := range claudeResp.Content {
		if content.Type == "text" && content.Text != "" {
			return content.Text, nil
		}
	}

	log.Printf("Empty response from Claude. Full response: %s", string(body))
	return "", fmt.Errorf("empty response from Claude")
}

func buildClaudeRequest(model, systemPrompt, userMessage string, tools []ClaudeTool) ClaudeRequest {
	return ClaudeRequest{
		Model:     model,
		MaxTokens: 500, // Keep responses concise for CarPlay
		System: []ClaudeSystemBlock{
			{
				Type: "text",
				Text: systemPrompt,
				CacheControl: &ClaudeCacheControl{
					Type: "ephemeral",
				},
			},
		},
		Messages: []ClaudeMessage{
			{Role: "user", Content: userMessage},
		},
		Tools: tools,
	}
}

func claudeWebSearchError(resp ClaudeResponse) error {
	for _, block := range resp.Content {
		if block.Type != "web_search_tool_result" || len(block.Content) == 0 {
			continue
		}

		var result struct {
			Type      string `json:"type"`
			ErrorCode string `json:"error_code"`
		}
		if err := json.Unmarshal(block.Content, &result); err != nil {
			continue
		}
		if result.Type == "web_search_tool_result_error" {
			if result.ErrorCode == "" {
				result.ErrorCode = "unknown"
			}
			return fmt.Errorf("Claude web search tool error: %s", result.ErrorCode)
		}
	}
	return nil
}

// isClaudeModel checks if the model name is a Claude model
func isClaudeModel(model string) bool {
	return strings.HasPrefix(model, "claude-")
}

func isOpenRouterModel(model string) bool {
	return strings.HasPrefix(model, "openrouter/")
}

func isOpenAIModel(model string) bool {
	return !isClaudeModel(model) && !isOpenRouterModel(model)
}

func openRouterModelID(model string) string {
	return strings.TrimPrefix(model, "openrouter/")
}

// handleConfigAPI handles GET and POST requests for /api/config endpoint
// GET: Returns current runtime configuration
// POST: Updates runtime configuration
func (s *Server) handleConfigAPI(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		setCORSHeaders(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetConfigAPI(w, r)
	case http.MethodPost:
		s.handleSetConfigAPI(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		setCORSHeaders(w)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

// handleGetConfigAPI returns the current runtime configuration as JSON
func (s *Server) handleGetConfigAPI(w http.ResponseWriter, r *http.Request) {
	config := admin.GetConfig()

	w.Header().Set("Content-Type", "application/json")
	setCORSHeaders(w)
	if err := json.NewEncoder(w).Encode(config); err != nil {
		log.Printf("Error encoding config: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
		return
	}
}

// handleSetConfigAPI updates the runtime configuration from JSON POST body
func (s *Server) handleSetConfigAPI(w http.ResponseWriter, r *http.Request) {
	const (
		maxSystemPromptSize = 10 * 1024 // 10KB
		maxConfigBodySize   = 50 * 1024 // 50KB
	)

	// Limit request body size
	limitedReader := io.LimitReader(r.Body, maxConfigBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		setCORSHeaders(w)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read request body"})
		return
	}
	r.Body.Close()

	// Check if body is too large
	if len(body) >= maxConfigBodySize {
		w.Header().Set("Content-Type", "application/json")
		setCORSHeaders(w)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		json.NewEncoder(w).Encode(map[string]string{"error": "Request body too large"})
		return
	}

	var newConfig admin.RuntimeConfig
	if err := json.Unmarshal(body, &newConfig); err != nil {
		w.Header().Set("Content-Type", "application/json")
		setCORSHeaders(w)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	var providedFields map[string]json.RawMessage
	if err := json.Unmarshal(body, &providedFields); err != nil {
		w.Header().Set("Content-Type", "application/json")
		setCORSHeaders(w)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}
	mergedConfig := mergeRuntimeConfig(admin.GetConfig(), newConfig, providedFields)

	// Validate base system prompt size (prefer BaseSystemPrompt, fallback to SystemPrompt for legacy)
	basePrompt := mergedConfig.BaseSystemPrompt
	if basePrompt == "" {
		basePrompt = mergedConfig.SystemPrompt
	}
	if basePrompt != "" && len(basePrompt) > maxSystemPromptSize {
		w.Header().Set("Content-Type", "application/json")
		setCORSHeaders(w)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Base system prompt exceeds maximum size"})
		return
	}

	// Validate category prompts size
	for category, prompt := range mergedConfig.CategoryPrompts {
		if prompt != "" && len(prompt) > maxSystemPromptSize {
			w.Header().Set("Content-Type", "application/json")
			setCORSHeaders(w)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Category prompt %s exceeds maximum size", category),
			})
			return
		}
	}

	// Update config using admin.SetConfig (includes model validation, prompt format validation, etc.)
	if err := admin.SetConfig(mergedConfig); err != nil {
		log.Printf("Error setting config via API: %v", err)
		w.Header().Set("Content-Type", "application/json")
		setCORSHeaders(w)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Log successful config update
	log.Printf("Config updated via API: standard_model=%s premium_model=%s", mergedConfig.StandardModel, mergedConfig.PremiumModel)

	// Return updated config
	w.Header().Set("Content-Type", "application/json")
	setCORSHeaders(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(admin.GetConfig())
}

func mergeRuntimeConfig(current, incoming admin.RuntimeConfig, provided map[string]json.RawMessage) admin.RuntimeConfig {
	merged := current

	if _, ok := provided["base_system_prompt"]; ok {
		merged.BaseSystemPrompt = incoming.BaseSystemPrompt
		merged.SystemPrompt = incoming.BaseSystemPrompt
	}
	if _, ok := provided["system_prompt"]; ok && incoming.SystemPrompt != "" {
		merged.BaseSystemPrompt = incoming.SystemPrompt
		merged.SystemPrompt = incoming.SystemPrompt
	}
	if _, ok := provided["category_prompts"]; ok {
		merged.CategoryPrompts = incoming.CategoryPrompts
	}
	if _, ok := provided["standard_model"]; ok {
		merged.StandardModel = incoming.StandardModel
	}
	if _, ok := provided["premium_model"]; ok {
		merged.PremiumModel = incoming.PremiumModel
	}
	if _, ok := provided["category_models"]; ok {
		merged.CategoryModels = incoming.CategoryModels
	}
	if _, ok := provided["perplexity_enabled"]; ok {
		merged.PerplexityEnabled = incoming.PerplexityEnabled
	}

	return merged
}

// buildSystemPrompt constructs the system prompt using specialized category prompts
// Category prompts are now self-contained (include all necessary rules) for token efficiency
func (s *Server) buildSystemPrompt(config admin.RuntimeConfig, category router.Category, currentTime string) string {
	// Get category-specific prompt override from config
	categoryKey := string(category)
	categoryPrompt := config.CategoryPrompts[categoryKey]

	// If no override, use default category prompt
	if categoryPrompt == "" {
		switch category {
		case router.CategoryWebSearch:
			categoryPrompt = categoryPromptWebSearch
		case router.CategoryComplex:
			categoryPrompt = categoryPromptComplex
		case router.CategoryFactual:
			categoryPrompt = categoryPromptFactual
		case router.CategoryMathematical:
			categoryPrompt = categoryPromptMathematical
		case router.CategoryCreative:
			categoryPrompt = categoryPromptCreative
		default:
			// CategorySimple or unknown - use minimal base prompt
			basePrompt := config.BaseSystemPrompt
			if basePrompt == "" {
				// Fallback to legacy SystemPrompt for backward compatibility
				basePrompt = config.SystemPrompt
			}
			if basePrompt == "" {
				// Ultimate fallback to default
				basePrompt = clotildeBaseSystemPromptTemplate
			}
			return injectCurrentTime(basePrompt, currentTime)
		}
	}

	// Category prompts are self-contained and include %s for date/time
	return injectCurrentTime(categoryPrompt, currentTime)
}

// injectCurrentTime replaces the optional time placeholder without invoking fmt formatting.
// Runtime category prompts are allowed to omit %s, and literal percent signs should stay literal.
func injectCurrentTime(prompt, currentTime string) string {
	if !strings.Contains(prompt, "%s") {
		return prompt
	}
	return strings.ReplaceAll(prompt, "%s", currentTime)
}

// modelSupportsReasoning checks if a model supports the reasoning parameter
// Only o-series and gpt-5 series models support reasoning configuration
func modelSupportsReasoning(model string) bool {
	reasoningModels := []string{
		"o1", "o1-mini", "o1-pro",
		"o3", "o3-mini",
		"o4-mini",
		"gpt-5", "gpt-5-mini", "gpt-5-nano", "gpt-5-pro", "gpt-5.1",
	}
	for _, m := range reasoningModels {
		if strings.HasPrefix(model, m) {
			return true
		}
	}
	return false
}
