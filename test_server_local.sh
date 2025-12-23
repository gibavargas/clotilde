#!/bin/bash
# Test the Clotilde server locally with Claude API
# Before running: export CLAUDE_KEY_SECRET_NAME='your-claude-api-key'

set -e

# Check if Claude key is provided
if [ -z "$CLAUDE_KEY_SECRET_NAME" ]; then
    echo "ERROR: CLAUDE_KEY_SECRET_NAME environment variable not set"
    echo "Usage: CLAUDE_KEY_SECRET_NAME='your-key' $0"
    exit 1
fi

echo "Building Clotilde server..."
cd /Users/jvguidi/Documents/clotilde
go build -o /tmp/clotilde ./cmd/clotilde/

echo ""
echo "Starting server in background..."
echo ""

# Set environment variables
export PORT=8080
# CLAUDE_KEY_SECRET_NAME is already set from environment
# For local testing, we need minimal OpenAI key (can be dummy for testing Claude)
export OPENAI_KEY_SECRET_NAME="${OPENAI_KEY_SECRET_NAME:-dummy-key-for-claude-test}"
export API_KEY_SECRET_NAME="${API_KEY_SECRET_NAME:-test-api-key-123}"

# Start server in background
/tmp/clotilde > /tmp/clotilde.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
echo "Waiting for server to start..."
sleep 3

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start"
    cat /tmp/clotilde.log
    exit 1
fi

echo "✅ Server started (PID: $SERVER_PID)"
echo ""

# Test the /health endpoint
echo "Test 1: Health check"
HEALTH=$(curl -s http://localhost:8080/health)
if echo "$HEALTH" | grep -q "ok"; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed"
    echo "$HEALTH"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

echo ""

# Test Claude API with a simple question
echo "Test 2: Claude API integration"
TEST_RESPONSE=$(curl -s -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-api-key-123" \
  -d '{"message": "Qual a capital do Brasil?"}' \
  --max-time 30)

if echo "$TEST_RESPONSE" | grep -q "error"; then
    echo "❌ Chat test failed"
    echo "$TEST_RESPONSE" | jq '.' 2>/dev/null || echo "$TEST_RESPONSE"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
else
    echo "✅ Chat test passed"
    echo "Response:"
    echo "$TEST_RESPONSE" | jq -r '.response' 2>/dev/null || echo "$TEST_RESPONSE"
fi

echo ""
echo "✅ All tests passed! Claude API is working correctly."
echo ""
echo "Stopping server..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo "Done!"

