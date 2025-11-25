# Security Review & Fixes Documentation

This document details the security vulnerabilities and bugs discovered during code review, the fixes implemented, and the reasoning behind each fix.

**Review Date**: 2025-01-XX  
**Reviewer**: AI Code Review Agent  
**Status**: All critical and medium issues fixed ✅

---

## Overview

A comprehensive security review was conducted on the Clotilde CarPlay Assistant codebase. The review identified **7 security issues** (3 critical, 3 medium, 1 low) and **2 code quality issues**. All issues have been fixed and the codebase has been hardened.

---

## 🔴 Critical Security Issues

### 1. CORS Preflight (OPTIONS) Requests Broken

**Severity**: 🔴 Critical  
**Impact**: Complete CORS failure - browsers cannot make requests to the API  
**Files Affected**: `internal/validator/validator.go`, `cmd/clotilde/main.go`

#### Why This Was Found

During code review, I noticed:
1. The validator middleware attempted to parse JSON for **all** requests
2. OPTIONS preflight requests have no body, causing JSON parsing to fail
3. The `handleChat` function only accepted POST, returning 405 for OPTIONS
4. This combination completely broke CORS for any browser-based clients

#### The Problem

```go
// BEFORE: Validator tried to parse JSON for OPTIONS requests
var reqBody requestBody
if err := json.Unmarshal(body, &reqBody); err != nil {
    http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
    return
}
```

OPTIONS requests have no body, so `json.Unmarshal` would fail, blocking all CORS preflight requests.

#### The Fix

**Why this fix was chosen**: 
- OPTIONS requests are standard HTTP preflight for CORS
- They must be handled before validation
- They don't need authentication or rate limiting

**Implementation**:
1. **Validator**: Skip validation for OPTIONS requests
   ```go
   // Skip validation for health check and OPTIONS preflight
   if r.URL.Path == "/health" || r.Method == http.MethodOptions {
       next.ServeHTTP(w, r)
       return
   }
   ```

2. **handleChat**: Add OPTIONS handler with CORS headers
   ```go
   // Handle CORS preflight
   if r.Method == http.MethodOptions {
       setCORSHeaders(w)
       w.WriteHeader(http.StatusNoContent)
       return
   }
   ```

**Result**: CORS preflight requests now work correctly, enabling browser-based clients.

---

### 2. X-Forwarded-For Header Spoofing Vulnerability

**Severity**: 🔴 Critical  
**Impact**: Attackers can bypass rate limiting by spoofing IP addresses  
**Files Affected**: `internal/ratelimit/ratelimit.go`

#### Why This Was Found

During review of the rate limiting implementation, I identified several issues:
1. The code blindly trusted `X-Forwarded-For` header without validation
2. `X-Forwarded-For` can contain multiple IPs: `"client, proxy1, proxy2"`
3. The code used the entire string as the rate limit key, not just the client IP
4. An attacker could send unique `X-Forwarded-For` values to bypass rate limits entirely

#### The Problem

```go
// BEFORE: Blindly trusted X-Forwarded-For
forwarded := r.Header.Get("X-Forwarded-For")
if forwarded != "" {
    return forwarded  // Returns entire string, including multiple IPs!
}
```

**Attack Scenario**:
- Attacker sends: `X-Forwarded-For: unique-ip-1`
- Next request: `X-Forwarded-For: unique-ip-2`
- Each request gets a different rate limit key → unlimited requests

#### The Fix

**Why this fix was chosen**:
- The first (leftmost) IP in `X-Forwarded-For` is the original client IP
- This is the standard way to extract client IP from proxy headers
- We also handle IPv6 addresses and port removal properly

**Implementation**:
```go
// Extract first IP only to prevent header spoofing bypass
forwarded := r.Header.Get("X-Forwarded-For")
if forwarded != "" {
    // Take only the first IP (original client)
    if idx := strings.Index(forwarded, ","); idx != -1 {
        return strings.TrimSpace(forwarded[:idx])
    }
    return strings.TrimSpace(forwarded)
}
```

**Additional Improvements**:
- Proper IPv6 handling (bracket notation: `[::1]:port`)
- Port removal from `RemoteAddr` fallback
- Whitespace trimming

**Result**: Rate limiting now works correctly even when attackers try to spoof headers.

---

### 3. CORS Wildcard in Production

**Severity**: 🔴 Critical  
**Impact**: Any website can make requests to the API (CSRF risk)  
**Files Affected**: `cmd/clotilde/main.go`

#### Why This Was Found

The CORS configuration had a dangerous default:
```go
allowedOrigin = "*" // Temporary - should be restricted in production
```

This comment indicated it was a known issue but not fixed.

#### The Problem

Using `Access-Control-Allow-Origin: *` allows **any origin** to make requests to the API. While the API uses API key authentication, this still:
- Enables CSRF attacks from malicious websites
- Violates security best practices
- Unnecessary since Apple Shortcuts doesn't need CORS (not browser-based)

#### The Fix

**Why this fix was chosen**:
- Apple Shortcuts doesn't use CORS (it's not a browser)
- Defaulting to no CORS is the safest option
- If CORS is needed later, it can be enabled via environment variable

**Implementation**:
```go
allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
if allowedOrigin == "" {
    // Default: no CORS (don't set Access-Control-Allow-Origin)
    // This is the safest default - set CORS_ALLOWED_ORIGIN env var if needed
    return
}
```

**Result**: CORS is disabled by default, only enabled when explicitly configured.

---

## 🟡 Medium Security Issues

### 4. Rate Limiting Before Authentication

**Severity**: 🟡 Medium  
**Impact**: Unauthenticated requests consume rate limit quotas, enabling DoS attacks  
**Files Affected**: `cmd/clotilde/main.go`

#### Why This Was Found

The middleware order was incorrect:
```go
handler := auth.Middleware(apiKeySecret)(mux)
handler = ratelimit.Middleware()(handler)
handler = validator.Middleware()(handler)
```

**Execution order** (outer to inner): validator → ratelimit → auth

This meant:
- Unauthenticated requests still consumed rate limit quotas
- An attacker could exhaust rate limits with invalid API keys
- Legitimate users could be blocked by attackers

#### The Fix

**Why this fix was chosen**:
- Authentication should happen before rate limiting
- Rate limiting should track by authenticated API key, not IP
- Validator should run first to reject large/invalid requests early

**Implementation**:
```go
// Middleware order (outer to inner): validator → auth → ratelimit
// 1. Validator: Limits request size early (prevents large payloads)
// 2. Auth: Validates API key before rate limiting
// 3. Ratelimit: Only rate-limits authenticated requests (by API key)
handler := ratelimit.Middleware()(mux)
handler = auth.Middleware(apiKeySecret)(handler)
handler = validator.Middleware()(handler)
```

**Result**: 
- Invalid requests are rejected before consuming rate limit quotas
- Rate limiting is per authenticated API key, not per IP
- Better protection against DoS attacks

---

### 5. Content-Type Handling (Updated 2025-11-24)

**Severity**: 🟡 Medium → ⚪ Resolved  
**Impact**: Apple Shortcuts compatibility  
**Files Affected**: `cmd/clotilde/main.go`

#### Original Issue

The server initially accepted any Content-Type but expected JSON. A strict Content-Type validation was added to prevent content-type confusion attacks.

#### Problem with Strict Validation

Apple Shortcuts sometimes sends `text/plain` as the Content-Type even when the body is valid JSON. This caused HTTP 415 (Unsupported Media Type) errors, breaking the shortcut.

#### The Fix (Updated)

**Why this fix was chosen**:
- Apple Shortcuts is the primary client and must work
- The JSON decoder will fail anyway if the body isn't valid JSON
- Security is maintained because invalid JSON is rejected by the decoder

**Implementation**:
```go
// Note: We don't strictly validate Content-Type because Apple Shortcuts
// sometimes sends text/plain even when the body is valid JSON.
// The JSON decoder will fail if the body isn't valid JSON anyway.
```

**Result**: Apple Shortcuts works correctly. Invalid requests are still rejected by JSON parsing.

---

### 6. Timeout Mismatch Between Cloud Run and Code

**Severity**: 🟡 Medium  
**Impact**: Web search queries fail due to premature timeout  
**Files Affected**: `cloudbuild.yaml`

#### Why This Was Found

During review, I noticed:
- Code expects 60 seconds for web search: `context.WithTimeout(context.Background(), 60*time.Second)`
- Cloud Run was configured for 30 seconds: `--timeout 30`
- Cloud Run kills requests after its timeout, causing web search to fail

#### The Fix

**Why this fix was chosen**:
- Code timeout must match Cloud Run timeout
- Web search needs more time than simple queries
- 60 seconds is reasonable for web search operations

**Implementation**:
```yaml
# cloudbuild.yaml
- '--timeout'
- '60'  # Changed from 30 to match code expectation
```

**Result**: Web search queries now have sufficient time to complete.

---

## 🟢 Code Quality Issues

### 7. Unused Function and Integer Overflow

**Severity**: 🟢 Low  
**Impact**: Dead code and potential integer overflow  
**Files Affected**: `cmd/clotilde/main.go`

#### Why This Was Found

Code review identified:
1. `min()` function was defined but never used
2. `hashIP()` used `int` which can overflow on long strings

#### The Fix

**Why this fix was chosen**:
- Remove dead code to reduce maintenance burden
- Use `uint64` to prevent overflow
- Use hex output for better readability

**Implementation**:
```go
// BEFORE: int can overflow, unused min() function
func min(a, b int) int { ... }  // Never used
func hashIP(ip string) string {
    hash := 0  // int can overflow
    ...
    return fmt.Sprintf("ip_%d", hash)  // Can be negative
}

// AFTER: uint64 prevents overflow, removed dead code
func hashIP(ip string) string {
    var hash uint64  // Prevents overflow
    for _, c := range ip {
        hash = hash*31 + uint64(c)
    }
    return fmt.Sprintf("ip_%x", hash)  // Hex output, always positive
}
```

**Result**: Cleaner code, no overflow risk, better hash representation.

---

## Summary of All Fixes

| # | Issue | Severity | Status | Files Changed |
|---|-------|----------|--------|---------------|
| 1 | CORS Preflight Broken | 🔴 Critical | ✅ Fixed | `validator.go`, `main.go` |
| 2 | X-Forwarded-For Spoofing | 🔴 Critical | ✅ Fixed | `ratelimit.go` |
| 3 | CORS Wildcard | 🔴 Critical | ✅ Fixed | `main.go` |
| 4 | Rate Limiting Order | 🟡 Medium | ✅ Fixed | `main.go` |
| 5 | Content-Type Handling | 🟡 Medium | ✅ Updated | `main.go` |
| 6 | Timeout Mismatch | 🟡 Medium | ✅ Fixed | `cloudbuild.yaml` |
| 7 | Dead Code & Overflow | 🟢 Low | ✅ Fixed | `main.go` |

---

## Testing Recommendations

After these fixes, verify:

1. **CORS**: Test with browser-based client (if applicable)
2. **Rate Limiting**: Verify it works per API key, not per spoofed IP
3. **Content-Type**: Accepts both `application/json` and `text/plain` (for Apple Shortcuts compatibility)
4. **Web Search**: Verify queries complete within 60 seconds
5. **OPTIONS**: Verify preflight requests work correctly

---

## Security Best Practices Applied

1. **Defense in Depth**: Multiple layers of security (validation → auth → rate limit)
2. **Fail Secure**: Default to most secure option (no CORS)
3. **Input Validation**: Validate all inputs early
4. **Header Parsing**: Never trust headers blindly, always validate
5. **Principle of Least Privilege**: Only enable features when needed
6. **Secure Defaults**: Most secure configuration by default

---

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CORS Security](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [X-Forwarded-For Header Security](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-For)
- [HTTP Security Headers](https://owasp.org/www-project-secure-headers/)

---

**Last Updated**: 2025-01-XX  
**Review Status**: Complete ✅  
**All Issues**: Resolved ✅

---

## 🔒 Secret Name Configuration (Critical Security Feature)

### Overview

**Date Implemented**: 2025-11-24  
**Severity**: 🔴 Critical  
**Impact**: Prevents secret name exposure in public GitHub repositories

### Why This Was Implemented

The original codebase had hardcoded secret names (`openai-api-key`, `clotilde-api-key`) visible in the public GitHub repository. While this doesn't expose the actual secret values, it reveals:
- Which secrets to target if someone gains access to your GCP project
- Predictable secret names that attackers could try to access
- Security through obscurity is eliminated

### The Problem

**Before**: Secret names were hardcoded in multiple places:
- `cmd/clotilde/main.go`: Lines 126, 142 (hardcoded fallbacks)
- `cloudbuild.yaml`: Line 47 (hardcoded in deployment)
- `deploy.sh`: Line 43 (hardcoded in script)
- `setup-gcloud.sh`: Lines 48, 62 (hardcoded in setup)

Anyone viewing the public repo could see exactly which secrets exist in Secret Manager.

### The Solution

**Implementation**: All secret names are now configurable via environment variables:

1. **Application Code** (`cmd/clotilde/main.go`):
   - `OPENAI_SECRET_NAME`: Name of OpenAI API key secret (required for Secret Manager lookup)
   - `API_SECRET_NAME`: Name of API key secret (required for Secret Manager lookup)
   - No hardcoded fallbacks - fails with clear error if not configured

2. **Cloud Build** (`cloudbuild.yaml`):
   - `_OPENAI_SECRET`: Substitution variable (default: placeholder)
   - `_API_SECRET`: Substitution variable (default: placeholder)
   - Must be provided at deploy time: `--substitutions=_OPENAI_SECRET=actual-name,_API_SECRET=actual-name`

3. **Deploy Script** (`deploy.sh`):
   - `OPENAI_SECRET`: Environment variable (required, no default)
   - `API_SECRET`: Environment variable (required, no default)
   - Script fails with helpful error if not set

4. **Setup Script** (`setup-gcloud.sh`):
   - Same pattern: requires `OPENAI_SECRET` and `API_SECRET` env vars
   - Uses these for secret creation and IAM binding

### Migration Process

A migration script (`migrate-secrets.sh`) automates the process:

1. **Creates new secrets** with unique, unpredictable names (e.g., `clotilde-oai-a1b2c3`)
2. **Copies values** from old secrets (if they exist)
3. **Grants IAM permissions** to Cloud Run service account
4. **Provides deployment commands** with new secret names

**Usage**:
```bash
chmod +x migrate-secrets.sh
./migrate-secrets.sh
```

The script generates unique suffixes using `openssl rand -hex 4` to ensure unpredictability.

### Local Development

For local Docker development, you need to set environment variables:

**Option 1: Direct values (for testing)**:
```bash
export OPENAI_KEY_SECRET_NAME="your-actual-openai-key-value"
export API_KEY_SECRET_NAME="your-actual-api-key-value"
export PORT=8080
go run cmd/clotilde/main.go
```

**Option 2: Secret Manager lookup (for production-like testing)**:
```bash
export OPENAI_SECRET_NAME="your-openai-secret-name"
export API_SECRET_NAME="your-api-secret-name"
export GOOGLE_CLOUD_PROJECT="your-project-id"
export PORT=8080
go run cmd/clotilde/main.go
```

**Important**: The application prefers `OPENAI_KEY_SECRET_NAME` and `API_KEY_SECRET_NAME` (direct values) over Secret Manager lookup. This allows:
- Local development without Secret Manager access
- Cloud Run secret mounting (secrets mounted as environment variables)
- Fallback to Secret Manager only when direct values aren't available

### Deployment Workflow

**Cloud Build** (recommended):
```bash
gcloud builds submit --config=cloudbuild.yaml \
  --substitutions=_OPENAI_SECRET=clotilde-oai-abc123,_API_SECRET=clotilde-auth-xyz789
```

**Manual Deploy**:
```bash
export OPENAI_SECRET=clotilde-oai-abc123
export API_SECRET=clotilde-auth-xyz789
./deploy.sh
```

### Security Benefits

1. **No Secret Names in Public Repo**: Only placeholders visible
2. **Unpredictable Names**: Attackers can't guess secret names
3. **Explicit Configuration**: Forces conscious secret name management
4. **Git-Safe**: Repository can be public without exposing infrastructure details

### Best Practices

1. **Use Unique Names**: Generate unpredictable secret names (e.g., `myapp-oai-abc123`)
2. **Store Secret Names Securely**: Don't commit actual secret names to git
3. **Rotate Regularly**: Create new secrets with new names periodically
4. **Document Locally**: Keep actual secret names in a secure local file (not in repo)

### Files Changed

| File | Change |
|------|--------|
| `cmd/clotilde/main.go` | Removed hardcoded secret names, added env var requirements |
| `cloudbuild.yaml` | Added substitution variables for secret names |
| `deploy.sh` | Added required environment variables |
| `setup-gcloud.sh` | Added required environment variables |
| `migrate-secrets.sh` | New migration script |
| All docs | Replaced specific names with placeholders |

### Testing Checklist

After implementing this feature, verify:

- [ ] Application fails with clear error if `OPENAI_SECRET_NAME` not set (local dev)
- [ ] Application fails with clear error if `API_SECRET_NAME` not set (local dev)
- [ ] Cloud Build deployment works with `--substitutions` provided
- [ ] Deploy script fails if `OPENAI_SECRET` or `API_SECRET` not set
- [ ] Migration script creates secrets with unique names
- [ ] IAM permissions granted correctly
- [ ] Service starts and works with new secret names
- [ ] No secret names visible in public GitHub repo

### Critical Notes

⚠️ **This is a breaking change**:
- Existing deployments will fail until secret names are configured
- Must run migration script or manually create new secrets
- Old secret names (`openai-api-key`, `clotilde-api-key`) should be deleted after migration

✅ **This is a critical security improvement**:
- Prevents information disclosure via public repositories
- Follows security best practices (no hardcoded secrets/names)
- Makes the codebase safe for public sharing

---

**Last Updated**: 2025-11-24  
**Status**: Implemented ✅  
**Migration Required**: Yes (use `migrate-secrets.sh`)

---

## ⚙️ Dynamic Runtime Configuration (Admin UI Feature)

### Overview

**Date Implemented**: 2025-11-25  
**Feature**: Dynamic system prompt and model selection without redeployment  
**Impact**: Saves Cloud Run costs by avoiding redeployment for configuration changes

### What Was Added

The admin UI now supports runtime configuration changes:
1. **System Prompt**: Editable via admin UI, takes effect immediately
2. **Model Selection**: Switch between OpenAI models that support the Responses API

### Architecture

```
┌─────────────────┐       ┌──────────────────────┐       ┌─────────────────┐
│   Admin UI      │ ───── │  /admin/config       │ ───── │  RuntimeConfig  │
│  (dashboard.go) │ POST  │  (admin.go handlers) │       │  (config.go)    │
└─────────────────┘       └──────────────────────┘       └─────────────────┘
                                                                  │
                                                                  ▼
                                                         ┌─────────────────┐
                                                         │   handleChat    │
                                                         │   routeToModel  │
                                                         │   (main.go)     │
                                                         └─────────────────┘
```

### Files Added/Modified

| File | Purpose |
|------|---------|
| `internal/admin/config.go` | Thread-safe in-memory configuration store |
| `internal/admin/admin.go` | GET/POST handlers for `/admin/config` |
| `internal/admin/dashboard.go` | Updated model dropdowns for Responses API models |
| `cmd/clotilde/main.go` | Uses `admin.GetConfig()` for dynamic values |

### Critical Code Paths (For AI Maintainers)

#### 1. Configuration Store (`internal/admin/config.go`)

```go
// Thread-safe singleton pattern
var (
    configMutex   sync.RWMutex
    runtimeConfig = RuntimeConfig{...}
    initialized   = false
)
```

**CRITICAL**: 
- Always use `GetConfig()` to read (uses `RLock`)
- Always use `SetConfig()` to write (uses `Lock`)
- Never access `runtimeConfig` directly from outside the package

#### 2. Default Initialization (`main.go`)

```go
// Called once at startup AFTER admin handler is created
admin.SetDefaultConfig(clotildeSystemPromptTemplate)
```

**CRITICAL**: 
- Must be called after admin routes are registered
- If not called, system prompt will be empty
- The `initialized` flag prevents overwriting config on subsequent calls

#### 3. Model Validation (`config.go`)

```go
validStandardModels := map[string]bool{
    "gpt-4.1-nano": true,
    "gpt-4.1-mini": true,
    "gpt-4o-mini":  true,
    "gpt-3.5-turbo": true,
}
```

**CRITICAL**: 
- If you add models to the dashboard dropdown, you MUST add them here too
- Validation is strict - invalid models are rejected with HTTP 400
- Keep the lists in sync between `config.go` and `dashboard.go`

#### 4. Dynamic Model Usage (`main.go`)

```go
func (s *Server) routeToModel(ctx context.Context, question string) RouteDecision {
    config := admin.GetConfig()
    standardModel := config.StandardModel
    premiumModel := config.PremiumModel
    // ... uses these instead of hardcoded values
}
```

**CRITICAL**:
- Config is fetched on EVERY request (not cached)
- This ensures changes take effect immediately
- The RWMutex makes this fast for concurrent reads

### Potential Breaking Points

1. **Model name mismatch**: If dashboard has a model not in validation list → HTTP 400 on save
2. **Empty system prompt**: If `SetDefaultConfig()` not called → empty prompts sent to OpenAI
3. **Invalid format string**: System prompt must have exactly one `%s` for date/time injection
4. **Concurrent modification**: Safe due to mutex, but excessive writes could slow reads

### Testing Checklist

- [ ] Admin UI loads current config on page load
- [ ] Save button updates config (check toast notification)
- [ ] Changes take effect immediately (no restart needed)
- [ ] Invalid models are rejected with error message
- [ ] System prompt with `%s` placeholder works correctly
- [ ] Model dropdown options match validation list

### Supported Models (Responses API)

**Standard (Fast/Cheap)**:
- `gpt-4.1-nano` - Cheapest option
- `gpt-4.1-mini` - Balanced performance/cost
- `gpt-4o-mini` - Legacy, still supported
- `gpt-3.5-turbo` - Older model

**Premium (Powerful)**:
- `gpt-4.1` - Recommended for complex queries
- `gpt-4o` - Previous generation flagship
- `gpt-4o-2024-08-06` - Specific snapshot
- `chatgpt-4o-latest` - Latest ChatGPT model
- `gpt-4-turbo` - High performance
- `o4-mini` - Reasoning model (cheaper)
- `o3` - Advanced reasoning
- `gpt-5.1` - Preview/experimental

---

## 🚨 Common Issues and Fixes for AI Models

### Issue: "empty response from API"

**Cause**: The Responses API response format changed or model doesn't support it.

**Fix in `createResponse()` (`main.go`)**:
```go
// Parse output array correctly
if outputArr, ok := apiResp.Output.([]interface{}); ok {
    for _, item := range outputArr {
        if itemMap, ok := item.(map[string]interface{}); ok {
            if itemType, ok := itemMap["type"].(string); ok && itemType == "message" {
                // Extract text from content array
            }
        }
    }
}
```

### Issue: Config changes don't take effect

**Cause**: Reading from const instead of `admin.GetConfig()`.

**Check**: Search for `clotildeSystemPromptTemplate` - it should only appear in:
1. The const definition
2. The `SetDefaultConfig()` call

### Issue: "Invalid standard model" or "Invalid premium model"

**Cause**: Model in dropdown not in validation list.

**Fix**: Add the model to both:
1. `internal/admin/config.go` - `validStandardModels` or `validPremiumModels` map
2. `internal/admin/dashboard.go` - The HTML `<select>` element

### Issue: Race condition in config access

**Cause**: Accessing `runtimeConfig` without mutex.

**Prevention**: NEVER export `runtimeConfig`. Only use `GetConfig()` and `SetConfig()`.

---

**Last Updated**: 2025-11-25  
**Status**: Implemented ✅  
**Breaking Changes**: None (backward compatible)

