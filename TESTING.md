# Testing Guide - Claude Integration

This document describes how to test the Clotilde server with Claude Haiku 4.5 integration.

## Quick Start

### Prerequisites
- Claude API key from [console.anthropic.com](https://console.anthropic.com)
- Go 1.21+ installed
- curl installed

### Test 1: Direct Claude API Test

Validates that the Claude API key is working correctly:

```bash
# Set your Claude API key
export CLAUDE_KEY_SECRET_NAME="sk-ant-api03-..."

# Run the direct API test
./test_claude_local.sh
```

**Expected output**:
```
Testing Claude API key...

Test 1: Direct Claude API call
✅ Claude API test PASSED
Olá! 👋
...
Claude API key is valid! ✅
```

### Test 2: Full Server Integration Test

Starts the Clotilde server locally and tests end-to-end functionality:

```bash
# Set environment variables
export CLAUDE_KEY_SECRET_NAME="sk-ant-api03-..."
export PORT=8080

# Run the integration test (builds server, starts it, tests it)
./test_server_local.sh
```

**Expected output**:
```
Building Clotilde server...
...
✅ Health check passed
✅ Chat test passed
Response:
Brasília é a capital do Brasil.
✅ All tests passed! Claude API is working correctly.
```

### Test 3: Performance Validation

Tests that all question types respond within the 25-second Apple Shortcuts timeout:

```bash
# Deploy to Cloud Run first, then run:
./test_timing.sh https://your-service.run.app YOUR_API_KEY
```

**Expected output**:
```
================================================
Clotilde Timing Test - CarPlay Compatibility
================================================
Target: https://your-service.run.app/chat
Max allowed time: 25s (Apple Shortcuts limit: ~30s)

Running 14 test cases...

[Simple] "Qual a capital do Brasil?" OK (2.1s)
[Complex] "Explique a crise climática..." OK (3.2s)
[WebSearch] "Qual o placar do último jogo?" OK (8.4s)

Results: 14/14 passed
✓ All tests passed within 25s limit
CarPlay compatibility: VERIFIED
```

## Environment Variables

### For test_claude_local.sh
- `CLAUDE_KEY_SECRET_NAME` - Claude API key (required)

### For test_server_local.sh
- `CLAUDE_KEY_SECRET_NAME` - Claude API key (required)
- `OPENAI_KEY_SECRET_NAME` - OpenAI key (optional, can be dummy)
- `API_KEY_SECRET_NAME` - Auth key for API (default: test-api-key-123)
- `PORT` - Server port (default: 8080)

### For test_timing.sh
- First argument: Base URL (e.g., https://your-service.run.app)
- Second argument: API key for authentication

## What Gets Tested

### test_claude_local.sh
✅ Claude API connectivity  
✅ API key validity  
✅ Model availability (claude-haiku-4-5-20251001)  
✅ Response format  
✅ Error handling  

### test_server_local.sh
✅ Server builds successfully  
✅ Server starts without errors  
✅ Health endpoint works  
✅ Claude integration works  
✅ Chat endpoint returns valid responses  
✅ Response time < 30s  
✅ Server graceful shutdown  

### test_timing.sh
✅ 14 different question types  
✅ Complexity levels: Simple, Factual, Complex, Creative, Math, WebSearch  
✅ All responses complete within 25s  
✅ HTTP status codes correct  
✅ Response format valid  
✅ Error messages helpful  

## Interpreting Results

### Success Indicators
- ✅ All tests pass
- Response times < 10s for Claude (typical: 1-3s)
- Response times < 15s for complex queries
- Response times < 25s for web search queries
- 100% success rate

### Warning Signs
- ⚠️ Response times 10-20s (Claude should be faster)
- ⚠️ Web search queries > 20s (check Perplexity API)
- ⚠️ High error rate (check API keys)

### Failure Scenarios
- ❌ "API key not configured" - Set CLAUDE_KEY_SECRET_NAME
- ❌ "Failed to make Claude request" - Check internet connectivity
- ❌ "model: claude-haiku-4-5-20251001" not found - Claude API issue
- ❌ Response times > 25s - Timeout imminent on CarPlay

## Performance Expectations

### Response Times by Question Type
```
Simple facts          : 1-2s
Factual questions    : 2-3s
Math calculations    : 1-2s
Creative suggestions : 2-4s
Complex analysis     : 3-5s
Web search (simple)  : 5-8s
Web search (complex) : 8-15s
```

### CPU/Memory Usage
- Idle: ~20MB RAM
- Processing: ~50-100MB RAM
- CPU: <10% during processing

## Troubleshooting

### "Test failed: HTTP 401"
- Claude API key is invalid
- Solution: Check key at https://console.anthropic.com

### "Test failed: HTTP 429"
- Rate limit exceeded on Claude API
- Solution: Wait a minute, then retry

### "Test failed: timeout"
- Server or API took too long
- Solution: Check network, try again

### "Server failed to start"
- Missing required environment variable
- Solution: Verify all env vars are set: `env | grep -i claude`

### "Claude API enabled" not in logs
- Claude key not configured
- Solution: Set CLAUDE_KEY_SECRET_NAME and restart

## Local Development Testing

### Testing with dummy keys
For testing server startup without real keys:

```bash
export CLAUDE_KEY_SECRET_NAME="test-dummy-key"
export OPENAI_KEY_SECRET_NAME="test-dummy-key"
export API_KEY_SECRET_NAME="test-api-key"
go run cmd/clotilde/main.go
```

Server will start with Claude marked as "enabled" but calls will fail.

### Testing production deployment
After deploying to Cloud Run:

```bash
# Get your API key from Secret Manager
API_KEY=$(gcloud secrets versions access latest --secret=clotilde-api-key)

# Run timing tests
./test_timing.sh https://clotilde-xxxxx.run.app "$API_KEY"
```

## Continuous Testing

### GitHub Actions (if configured)
Tests run automatically on:
- Every push to main
- Pull requests

### Manual testing schedule
- Before deploying: Run all tests locally
- After deploying: Run test_timing.sh on production
- Weekly: Run full test suite on production

## Performance Optimization

If tests show slow response times:

1. **Check Claude API status**
   ```bash
   curl -s https://status.anthropic.com/
   ```

2. **Check Cloud Run metrics**
   ```bash
   gcloud run services describe clotilde --region=us-central1
   ```

3. **Review logs for errors**
   ```bash
   gcloud run services logs read clotilde --region=us-central1 --limit=50
   ```

4. **Check rate limits**
   - Claude: 25 req/min for free tier
   - Consider upgrading if hitting limits

## References

- [Claude API Documentation](https://docs.anthropic.com/)
- [Clotilde GitHub Issues](https://github.com/gibavargas/clotilde/issues)
- [CHANGELOG_CLAUDE.md](./CHANGELOG_CLAUDE.md) - Detailed integration notes
- [Apple Shortcuts Timeout Limits](https://developer.apple.com/documentation/shortcuts)

