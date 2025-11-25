package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/clotilde/carplay-assistant/internal/admin"
	"github.com/clotilde/carplay-assistant/internal/auth"
	"github.com/clotilde/carplay-assistant/internal/logging"
	"github.com/clotilde/carplay-assistant/internal/ratelimit"
	"github.com/clotilde/carplay-assistant/internal/validator"
	"github.com/sashabaranov/go-openai"
)

var startTime = time.Now()

const (
	timezoneBR                   = "America/Sao_Paulo"
	clotildeSystemPromptTemplate = `You are "Clotilde", my in‑car copilot, accessed via an Apple Shortcut in Apple CarPlay.

Current date and time: %s (Brazil/São Paulo timezone)

Constraints and safety:
- I'm listening to your responses, so response length should be at most 60 seconds of talking. Provide complete, detailed answers when appropriate.
- Never show code blocks or markdown; respond as plain text only.
- CRITICAL: NEVER mention URLs, website addresses, or links in your responses. Since responses are spoken aloud, URLs are useless and annoying. Instead, just mention the source name (e.g., "G1", "BBC", "Reuters") without any web addresses.
- CRITICAL: NEVER ask questions to the user. This is a single-turn conversation with no follow-up. Always provide the best answer you can with the information available, even if incomplete. If information is missing, state what you know and acknowledge limitations, but do not ask for clarification.
- CRITICAL: ABSOLUTELY FORBIDDEN to include any URLs, web addresses, or links. Do NOT mention "www", "http", "https", ".com", ".br", or any domain names. Do NOT say things like "você pode ver em..." or "acesse...". Only mention source names (e.g., "Segundo o G1" or "De acordo com a Reuters"). If you catch yourself about to mention a URL, stop immediately and rephrase without it.

Web search behavior:
- You have access to web search capabilities that return results from the public web.
- Use web search when the question is about current events, recent news, live prices, weather "today" or "now", or anything where fresh/real-time data is essential.
- If the question references "today", "now", "recent", "latest", or similar time-sensitive terms, use web search.
- For historical facts, general knowledge, or information that doesn't change, you may not need web search but you are still free to use it as you think it may be needed.
- CRITICAL: When using web search, ALWAYS cite your sources with specific publication names (e.g., "Segundo o G1..." or "De acordo com a BBC...").
- When reporting news or time-sensitive information, ALWAYS include the specific date and time when available (e.g., "Segundo o G1, às 14h30 de hoje..." or "De acordo com a Reuters, na manhã de 24 de novembro...").
- When searching for information or news about a specific country, perform the web search in that country's language (e.g., search in English for US news, Spanish for Spain/Mexico news, French for France news, etc.), but always respond in Brazilian Portuguese as usual.
- Provide factual, objective information. Focus on what happened, when it happened, and who/what was involved. Avoid generic disclaimers about information changing or being approximate unless truly relevant.

Style and personality:
- Treat me as an intelligent adult. Do not include obvious disclaimers like "notícias podem mudar" or "é possível consultar" - I already know this.
- Be direct, factual, and informative. Provide concrete details, numbers, dates, times, and specific information.
- Avoid patronizing phrases, generic warnings, or stating the obvious.
- Be calm and pragmatic. You can be slightly humorous when appropriate, but never chatty for its own sake.
- If the question is ambiguous, provide the most likely interpretation and answer based on that.

Output format:
- Always answer in Brazilian Portuguese unless I clearly use another language.
- CRITICAL: Your response MUST be at most TWO paragraphs. Be concise and focused. If you need to cover multiple points, prioritize the most important information and condense it into two paragraphs maximum.
- Provide complete, detailed answers with specific facts, dates, times, and concrete information.
- When citing sources, include the publication name and, when available, the specific time/date of the information.
- Do not include your name or meta‑comments unless I ask; stay in character as Clotilde.
- NEVER end with a question or ask for more information.
- NEVER include disclaimers about information being approximate, time-sensitive, or subject to change unless it's genuinely relevant context (e.g., "a cotação pode variar durante o pregão" is fine, but "notícias podem mudar a qualquer momento" is not).`

	routerPrompt = `Classify this question into ONE category based on TWO factors:
1. Does it need CURRENT/REAL-TIME info? (news, prices, weather, today's events)
2. Does it need DEEP ANALYSIS? (comparisons, reasoning, detailed explanations)

Categories:
- SIMPLE: No current info needed, no deep analysis (greetings, basic facts, definitions)
- SEARCH: Needs current info, but simple answer (today's weather, current price, latest news)
- COMPLEX: Needs deep analysis, but NO current info (explain a concept, analyze a book, creative writing)
- BOTH: Needs current info AND deep analysis (analyze today's market trends, compare recent events)

Answer with ONLY one word: "simple", "search", "complex", or "both"

Question: %s`
)

// RouteDecision contains routing information for a request
type RouteDecision struct {
	Model           string
	WebSearch       bool
	ReasoningEffort string // "none", "low", "medium", "high" - empty means no reasoning config
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

type Server struct {
	openaiClient *openai.Client
	openaiAPIKey string
	apiKeySecret string
	logger       *logging.Logger
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

	// Get OpenAI API key - prefer environment variable (Cloud Run secrets) over Secret Manager
	openaiKey := os.Getenv("OPENAI_KEY_SECRET_NAME")
	if openaiKey == "" {
		// Fallback to Secret Manager for local development
		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" {
			log.Fatal("GOOGLE_CLOUD_PROJECT environment variable not set")
		}
		// Secret name must be configured via environment variable (not hardcoded for security)
		openaiSecretName := os.Getenv("OPENAI_SECRET_NAME")
		if openaiSecretName == "" {
			log.Fatal("OPENAI_SECRET_NAME environment variable not set (required for Secret Manager lookup)")
		}
		var err error
		openaiKey, err = getSecret(ctx, secretClient, projectID, openaiSecretName)
		if err != nil {
			log.Fatalf("Failed to get OpenAI API key: %v", err)
		}
	}

	// Get API key secret for authentication - prefer environment variable over Secret Manager
	apiKeySecret := os.Getenv("API_KEY_SECRET_NAME")
	if apiKeySecret == "" {
		// Fallback to Secret Manager for local development
		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" {
			log.Fatal("GOOGLE_CLOUD_PROJECT environment variable not set")
		}
		// Secret name must be configured via environment variable (not hardcoded for security)
		apiSecretName := os.Getenv("API_SECRET_NAME")
		if apiSecretName == "" {
			log.Fatal("API_SECRET_NAME environment variable not set (required for Secret Manager lookup)")
		}
		var err error
		apiKeySecret, err = getSecret(ctx, secretClient, projectID, apiSecretName)
		if err != nil {
			log.Fatalf("Failed to get API key secret: %v", err)
		}
	}

	// Initialize OpenAI client (still used for router)
	openaiClient := openai.NewClient(openaiKey)

	// Initialize logger
	logger := logging.GetLogger()

	server := &Server{
		openaiClient: openaiClient,
		openaiAPIKey: openaiKey,
		apiKeySecret: apiKeySecret,
		logger:       logger,
	}

	// Setup middleware chain
	mux := http.NewServeMux()
	mux.HandleFunc("/chat", server.handleChat)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/", handleOptions) // CORS preflight for root

	// Register admin routes (protected by HTTP Basic Auth)
	adminHandler := admin.NewHandler(logger)
	if adminHandler.IsEnabled() {
		adminHandler.RegisterRoutes(mux)
		log.Printf("Admin dashboard enabled at /admin/")
	} else {
		log.Printf("Admin dashboard disabled (ADMIN_USER and ADMIN_PASSWORD not set)")
	}

	// Initialize default runtime configuration with the system prompt template
	admin.SetDefaultConfig(clotildeSystemPromptTemplate)

	// Middleware order (outer to inner): requestID → validator → auth → ratelimit
	// 1. RequestID: Adds unique request ID for tracing
	// 2. Validator: Limits request size early (prevents large payloads)
	// 3. Auth: Validates API key before rate limiting
	// 4. Ratelimit: Only rate-limits authenticated requests (by API key)
	handler := ratelimit.Middleware()(mux)
	handler = auth.Middleware(apiKeySecret)(handler)
	handler = validator.Middleware()(handler)
	handler = logging.RequestIDMiddleware(handler)

	serverAddr := fmt.Sprintf(":%s", port)
	
	// Create HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: handler,
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
		s.logRequest(requestID, r, "", "", "", time.Since(startTime), "error", "Invalid request body")
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		s.logRequest(requestID, r, "", "", "", time.Since(startTime), "error", "Message is required")
		respondError(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Log request metadata (no sensitive data)
	log.Printf("[%s] Request received: IP=%s, MessageLength=%d", requestID, hashIP(r.RemoteAddr), len(req.Message))

	// Route to appropriate model and determine if web search is needed
	routerCtx, routerCancel := context.WithTimeout(context.Background(), 5*time.Second)
	route := s.routeToModel(routerCtx, req.Message)
	routerCancel()
	log.Printf("[%s] Route decision: Model=%s, WebSearch=%v", requestID, route.Model, route.WebSearch)

	// Call OpenAI with selected model and tools
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Increased timeout for web search
	defer cancel()

	// Get current date/time in Brazil timezone for context
	currentTime := getCurrentBrazilTime()
	// Get dynamic system prompt from runtime config
	config := admin.GetConfig()
	systemPrompt := fmt.Sprintf(config.SystemPrompt, currentTime)

	// Use Responses API instead of Chat Completions
	response, err := s.createResponse(ctx, route, systemPrompt, req.Message)
	if err != nil {
		log.Printf("[%s] OpenAI Responses API error: %v", requestID, err)
		s.logRequest(requestID, r, req.Message, "", route.Model, time.Since(startTime), "error", err.Error())
		respondError(w, "Failed to get response from AI", http.StatusInternalServerError)
		return
	}

	if response == "" {
		response = "Desculpe, não consegui processar sua solicitação. Pode repetir?"
	}

	// Log successful request
	responseTime := time.Since(startTime)
	log.Printf("[%s] Response generated: Length=%d, Time=%v", requestID, len(response), responseTime)
	s.logRequest(requestID, r, req.Message, response, route.Model, responseTime, "success", "")

	respondSuccess(w, response)
}

// logRequest adds a structured log entry with full input/output for Cloud Logging
func (s *Server) logRequest(requestID string, r *http.Request, input, output, model string, responseTime time.Duration, status, errorMsg string) {
	entry := logging.LogEntry{
		ID:            requestID,
		Timestamp:     time.Now(),
		IPHash:        hashIP(r.RemoteAddr),
		MessageLength: len(input),
		Model:         model,
		ResponseTime:  responseTime.Milliseconds(),
		TokenEstimate: len(input) / 4, // Rough estimate: ~4 chars per token
		Status:        status,
		ErrorMessage:  errorMsg,
		Input:         input,
		Output:        output,
	}
	s.logger.Add(entry)
}

func respondSuccess(w http.ResponseWriter, response string) {
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

func hashIP(ip string) string {
	// Simple hash for logging (not cryptographically secure, just for basic obfuscation)
	// Using uint64 to avoid integer overflow on long strings
	var hash uint64
	for _, c := range ip {
		hash = hash*31 + uint64(c)
	}
	return fmt.Sprintf("ip_%x", hash)
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

// createResponse calls the OpenAI Responses API
// The Responses API has native web_search support and handles tool calls automatically
func (s *Server) createResponse(ctx context.Context, route RouteDecision, instructions, input string) (string, error) {
	// Build request body for Responses API
	store := false // Don't store state for single-turn conversations

	reqBody := ResponsesAPIRequest{
		Model:        route.Model,
		Input:        input, // Can be string or messages array
		Instructions: instructions,
		Store:        &store,
	}

	// Only enable web_search when needed (costs extra)
	// All models that support Responses API support web_search tool
	if route.WebSearch {
		// Use web_search tool - supported by all Responses API models
		webSearchTool := WebSearchTool{Type: "web_search"}
		reqBody.Tools = []interface{}{webSearchTool}
		log.Printf("[%s] Web search enabled for model: %s", route.Model, route.Model)
	}

	// Set reasoning effort only for models that support it (o1, o3, gpt-5 series)
	// Models like gpt-4o, gpt-4-turbo don't support reasoning parameter
	if route.ReasoningEffort != "" && modelSupportsReasoning(route.Model) {
		reqBody.Reasoning = &ReasoningConfig{Effort: route.ReasoningEffort}
		log.Printf("Reasoning effort: %s", route.ReasoningEffort)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request to Responses API
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.openaiAPIKey))

	// Make HTTP request
	client := &http.Client{Timeout: 60 * time.Second}
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

// routeToModel determines which model and tools to use based on question type
// Cost optimization: uses cheaper models and only enables web search when needed
func (s *Server) routeToModel(ctx context.Context, question string) RouteDecision {
	// Get dynamic model configuration
	config := admin.GetConfig()
	standardModel := config.StandardModel
	premiumModel := config.PremiumModel

	// Use standard model for routing (very cheap)
	routerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	routerResp, err := s.openaiClient.CreateChatCompletion(routerCtx, openai.ChatCompletionRequest{
		Model: standardModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf(routerPrompt, question),
			},
		},
		MaxTokens:   5, // Only need one word
		Temperature: 0.0,
	})

	// Default: use standard model without web search
	defaultDecision := RouteDecision{
		Model:     standardModel,
		WebSearch: false,
	}

	if err != nil {
		log.Printf("Router error, using default: %v", err)
		return defaultDecision
	}

	if len(routerResp.Choices) == 0 {
		log.Printf("Router returned no choices, using default")
		return defaultDecision
	}

	decision := strings.ToLower(strings.TrimSpace(routerResp.Choices[0].Message.Content))
	log.Printf("Router decision: %s", decision)

	// SIMPLE: Use standard model, no web search (cheapest)
	if strings.Contains(decision, "simple") {
		return RouteDecision{
			Model:           standardModel,
			WebSearch:       false,
			ReasoningEffort: "", // Standard models don't have reasoning
		}
	}

	// SEARCH: Use standard model with web search (medium cost - simple + current data)
	if strings.Contains(decision, "search") {
		return RouteDecision{
			Model:           standardModel,
			WebSearch:       true,
			ReasoningEffort: "", // Standard models don't have reasoning
		}
	}

	// BOTH: Use premium model WITH web search (premium - complex + current data)
	if strings.Contains(decision, "both") {
		return RouteDecision{
			Model:           premiumModel,
			WebSearch:       true,
			ReasoningEffort: "none", // Disable reasoning to save cost
		}
	}

	// COMPLEX: Use premium model without web search (premium - complex, no current data)
	if strings.Contains(decision, "complex") {
		return RouteDecision{
			Model:           premiumModel,
			WebSearch:       false,
			ReasoningEffort: "none", // Disable reasoning to save cost
		}
	}

	// Default to standard model
	return defaultDecision
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

// Cost optimization notes:
// - SIMPLE: gpt-4o-mini, no search (cheapest)
// - SEARCH: gpt-4o-mini + web search (simple questions needing current data)
// - COMPLEX: premium model, no search (deep analysis, no current data)
// - BOTH: premium model + web search (deep analysis + current data)
// This smart routing reduces costs significantly vs always using premium models
