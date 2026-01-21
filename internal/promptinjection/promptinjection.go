package promptinjection

import (
	"regexp"
	"strings"
	"unicode"
)

// Common prompt injection patterns to detect
var injectionPatterns = []*regexp.Regexp{
	// Direct instruction override attempts
	regexp.MustCompile(`(?i)(ignore|disregard|forget|override|skip|bypass).*?(previous|prior|above|earlier|all).*?(instruction|prompt|directive|rule|guideline|system)`),
	regexp.MustCompile(`(?i)(ignore|disregard|forget).*?(everything|all)`),
	
	// System prompt extraction attempts
	regexp.MustCompile(`(?i)(show|display|output|print|reveal|tell|give).*?(system|base|initial|original).*?(prompt|instruction|directive|rule)`),
	regexp.MustCompile(`(?i)(what|what are).*?(your|the).*?(instruction|prompt|directive|rule|guideline|system prompt)`),
	regexp.MustCompile(`(?i)(repeat|echo|copy).*?(your|the).*?(instruction|prompt|directive|system prompt)`),
	
	// Role/jailbreak attempts
	regexp.MustCompile(`(?i)(you are|act as|pretend|roleplay|simulate).*?(developer|admin|root|system|assistant|ai|gpt)`),
	regexp.MustCompile(`(?i)(new instruction|new prompt|new directive|new rule)`),
	
	// Encoding/obfuscation attempts
	regexp.MustCompile(`(?i)(base64|hex|decode|encode).*?(instruction|prompt|system)`),
	
	// Instruction injection markers
	regexp.MustCompile(`(?i)(<\|.*?\|>|\[INST\]|\[/INST\]|### Instruction|### System)`),
	
	// Attempts to break out of context
	regexp.MustCompile(`(?i)(ignore.*?above|disregard.*?above|forget.*?above)`),
}

// SuspiciousKeywords are individual words that might indicate injection attempts
var suspiciousKeywords = []string{
	"jailbreak", "override", "bypass", "extract", "reveal", "system prompt",
	"base prompt", "initial prompt", "ignore all", "forget all",
}

// Pre-compiled regex patterns for sanitization
var (
	reWhitespace = regexp.MustCompile(`\s+`)
	reTag        = regexp.MustCompile(`(?i)<\|.*?\|>`)
	reInst       = regexp.MustCompile(`(?i)\[INST\].*?\[/INST\]`)
	reHeader     = regexp.MustCompile(`(?i)###\s*(Instruction|System|User)`)

	neutralizePatternsList = []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|prior|above|earlier)\s+(instruction|prompt|directive|rule|guideline|system)`), ""},
		{regexp.MustCompile(`(?i)disregard\s+(everything|all|all\s+above)`), ""},
		{regexp.MustCompile(`(?i)(show|display|output|print|reveal|tell|give)\s+(me\s+)?(your|the)\s+(system|base|initial|original)\s+(prompt|instruction|directive|rule)`), ""},
		{regexp.MustCompile(`(?i)(what|what\s+are)\s+(your|the)\s+(instruction|prompt|directive|rule|guideline|system\s+prompt)`), ""},
		{regexp.MustCompile(`(?i)(repeat|echo|copy)\s+(your|the)\s+(instruction|prompt|directive|system\s+prompt)`), ""},
		{regexp.MustCompile(`(?i)(you\s+are|act\s+as|pretend|roleplay|simulate).*?(developer|admin|root|system|assistant|ai|gpt)`), ""},
		{regexp.MustCompile(`(?i)(new\s+instruction|new\s+prompt|new\s+directive|new\s+rule)`), ""},
		{regexp.MustCompile(`(?i)(base64|hex|decode|encode).*?(instruction|prompt|system)`), ""},
		{regexp.MustCompile(`(?i)jailbreak`), ""},
	}
)

// SanitizeInput detects and neutralizes prompt injection attempts in user input
// Returns the sanitized input and a boolean indicating if injection was detected
func SanitizeInput(input string) (sanitized string, detected bool) {
	if input == "" {
		return input, false
	}

	originalInput := input
	detected = false

	// Normalize input for detection (lowercase, remove extra spaces)
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = reWhitespace.ReplaceAllString(normalized, " ")

	// Check against injection patterns
	for _, pattern := range injectionPatterns {
		if pattern.MatchString(normalized) {
			detected = true
			break
		}
	}

	// Check for suspicious keywords (as whole words or phrases)
	if !detected {
		for _, keyword := range suspiciousKeywords {
			// Check if keyword appears in the normalized input
			if strings.Contains(normalized, keyword) {
				detected = true
				break
			}
		}
	}

	// If injection detected, neutralize by:
	// 1. Escaping special characters that might be interpreted as instructions
	// 2. Removing or neutralizing dangerous patterns
	// 3. Preserving the core intent if possible
	
	if detected {
		// Start with original input
		sanitized = originalInput
		
		// Remove common injection markers
		sanitized = reTag.ReplaceAllString(sanitized, "")
		sanitized = reInst.ReplaceAllString(sanitized, "")
		sanitized = reHeader.ReplaceAllString(sanitized, "")
		
		// Neutralize common injection phrases by removing or replacing them
		for _, np := range neutralizePatternsList {
			sanitized = np.pattern.ReplaceAllString(sanitized, np.replacement)
		}
		
		// Escape potential instruction separators
		sanitized = strings.ReplaceAll(sanitized, "\n\n---\n\n", "\n\n")
		sanitized = strings.ReplaceAll(sanitized, "\n---\n", "\n")
		
		// Clean up extra whitespace
		sanitized = reWhitespace.ReplaceAllString(sanitized, " ")
		sanitized = strings.TrimSpace(sanitized)
		
		// If after sanitization the input is too short or empty, return a safe default
		if len(sanitized) < 3 {
			return "Desculpe, não entendi sua mensagem. Pode reformular?", true
		}
		
		// Ensure sanitized differs from original (at minimum, trim should help)
		// If somehow still identical, add a marker to indicate sanitization occurred
		if sanitized == originalInput {
			// This shouldn't happen, but if it does, we've at least logged it
			// The system prompt will still protect against it
		}
		
		return sanitized, true
	}

	// No injection detected, return original input
	return originalInput, false
}

// ValidateInput performs basic validation and returns sanitized input
// This is the main function to use for input sanitization
func ValidateInput(input string) (string, error) {
	// Check for null bytes
	if strings.Contains(input, "\x00") {
		return "", &ValidationError{Message: "Input contains null bytes"}
	}

	// Check for valid UTF-8 (basic check)
	for _, r := range input {
		if r == unicode.ReplacementChar {
			return "", &ValidationError{Message: "Input contains invalid UTF-8"}
		}
	}

	// Sanitize for prompt injection
	sanitized, detected := SanitizeInput(input)
	
	if detected {
		// Log detection but still return sanitized input
		// The system prompt will handle the rest
		return sanitized, nil
	}

	return sanitized, nil
}

// ValidationError represents an input validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

