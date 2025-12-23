#!/bin/bash
# Test Claude API locally
# This script tests if Claude API is working with the provided key

set -e

# Get Claude API key from environment variable
CLAUDE_KEY="${CLAUDE_KEY_SECRET_NAME}"

if [ -z "$CLAUDE_KEY" ]; then
    echo "ERROR: CLAUDE_KEY_SECRET_NAME environment variable not set"
    echo "Usage: CLAUDE_KEY_SECRET_NAME='your-key' $0"
    exit 1
fi

echo "Testing Claude API key..."
echo ""

# Test 1: Direct API call to Claude
echo "Test 1: Direct Claude API call"
RESPONSE=$(curl -s -X POST "https://api.anthropic.com/v1/messages" \
  -H "Content-Type: application/json" \
  -H "x-api-key: $CLAUDE_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-haiku-4-5-20251001",
    "max_tokens": 100,
    "messages": [{"role": "user", "content": "Say hello in Portuguese"}]
  }' --max-time 20)

if echo "$RESPONSE" | grep -q "error"; then
    echo "❌ Claude API test FAILED"
    echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
    exit 1
else
    echo "✅ Claude API test PASSED"
    echo "$RESPONSE" | jq -r '.content[0].text' 2>/dev/null || echo "Response received"
fi

echo ""
echo "Claude API key is valid! ✅"

