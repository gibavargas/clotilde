package logging

import (
	"os"
	"regexp"
	"sync"
)

var (
	redactPIIEnabled bool
	redactPIIOnce    sync.Once
)

// IsRedactPIIEnabled checks if PII redaction is enabled via environment variable
// Exported so it can be used by other packages
func IsRedactPIIEnabled() bool {
	redactPIIOnce.Do(func() {
		redactPIIEnabled = os.Getenv("LOG_REDACT_PII") == "true"
	})
	return redactPIIEnabled
}

// RedactPII removes or masks personally identifiable information from text
// This includes phone numbers, email addresses, and other common PII patterns
func RedactPII(text string) string {
	if !IsRedactPIIEnabled() || text == "" {
		return text
	}

	result := text

	// Redact email addresses (e.g., user@example.com -> [EMAIL_REDACTED])
	emailPattern := regexp.MustCompile(`(?i)\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	result = emailPattern.ReplaceAllString(result, "[EMAIL_REDACTED]")

	// Redact phone numbers (various formats)
	// Brazilian phone: (11) 98765-4321, 11 98765-4321, 11987654321
	// US phone: (555) 123-4567, 555-123-4567, 5551234567
	phonePattern := regexp.MustCompile(`(?i)(\+?\d{1,3}[-.\s]?)?\(?\d{2,3}\)?[-.\s]?\d{4,5}[-.\s]?\d{4}`)
	result = phonePattern.ReplaceAllString(result, "[PHONE_REDACTED]")

	// Redact credit card numbers (16 digits, possibly with spaces/dashes)
	creditCardPattern := regexp.MustCompile(`\b\d{4}[-.\s]?\d{4}[-.\s]?\d{4}[-.\s]?\d{4}\b`)
	result = creditCardPattern.ReplaceAllString(result, "[CARD_REDACTED]")

	// Redact CPF (Brazilian tax ID): 123.456.789-00 or 12345678900
	cpfPattern := regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{3}[-.\s]?\d{2}\b`)
	result = cpfPattern.ReplaceAllString(result, "[CPF_REDACTED]")

	// Redact common patterns that might indicate addresses
	// This is more conservative to avoid false positives
	// Address patterns like "Rua X, 123" or "Av. Y, 456" are harder to detect reliably

	return result
}

// ShouldLogFullContent checks if full content logging is enabled
// If LOG_FULL_CONTENT=false, only metadata should be logged
func ShouldLogFullContent() bool {
	// Default to true (full content logging) for backward compatibility
	if os.Getenv("LOG_FULL_CONTENT") == "false" {
		return false
	}
	return true
}

