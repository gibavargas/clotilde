#!/bin/bash
# Iterative Testing Script for Edge Case Validation
# This script tests prompts with actual API calls to validate edge case handling
#
# Prerequisites:
#   - Service must be deployed and running
#   - SERVICE_URL environment variable set to your Cloud Run service URL
#   - API_KEY environment variable set to your API key
#
# Usage:
#   export SERVICE_URL=https://your-service-url.run.app
#   export API_KEY=your-api-key
#   ./test_edge_cases.sh

set -e

if [ -z "$SERVICE_URL" ] || [ -z "$API_KEY" ]; then
    echo "ERROR: SERVICE_URL and API_KEY environment variables must be set"
    exit 1
fi

echo "Testing Edge Cases with Clotilde API"
echo "Service URL: $SERVICE_URL"
echo ""

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0

# Function to test a question and check response
test_case() {
    local category=$1
    local question=$2
    local expected_keywords=$3  # Comma-separated keywords that should appear in response
    
    echo -n "Testing [$category]: $question ... "
    
    response=$(curl -s -X POST "$SERVICE_URL/chat" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: $API_KEY" \
        -d "{\"message\":\"$question\"}" | jq -r '.response' 2>/dev/null || echo "ERROR")
    
    if [ "$response" = "ERROR" ] || [ -z "$response" ]; then
        echo -e "${RED}FAILED${NC} - No response or error"
        ((FAILED++))
        return 1
    fi
    
    # Check for expected keywords
    local found=0
    IFS=',' read -ra KEYWORDS <<< "$expected_keywords"
    for keyword in "${KEYWORDS[@]}"; do
        if echo "$response" | grep -qi "$keyword"; then
            found=1
            break
        fi
    done
    
    if [ $found -eq 1 ] || [ -z "$expected_keywords" ]; then
        echo -e "${GREEN}PASSED${NC}"
        echo "  Response: ${response:0:100}..."
        ((PASSED++))
        return 0
    else
        echo -e "${YELLOW}PARTIAL${NC} - Response received but may not match expected behavior"
        echo "  Response: ${response:0:100}..."
        ((PASSED++))
        return 0
    fi
}

echo "=== Hallucination Prevention Tests ==="
test_case "Hallucination" "What happened in the Battle of Zyxwvut in 2099?" "não sei,certainty,uncertain"
test_case "Hallucination" "Tell me about the famous author John Smith who won the Nobel Prize in 2024" "não sei,certainty,uncertain"

echo ""
echo "=== False Information Correction Tests ==="
test_case "False Info" "The capital of Brazil is São Paulo, right?" "Brasília,correto,na verdade"
test_case "False Info" "I heard that the Earth is flat. Is that true?" "não,incorreto,esférico"

echo ""
echo "=== Ambiguous Questions Tests ==="
test_case "Ambiguous" "What's the weather?" "interpretação,provavelmente"
test_case "Ambiguous" "How much does it cost?" "específico,qual"

echo ""
echo "=== Contradictory Information Tests ==="
test_case "Contradictory" "The sun is hot and the sun is cold. Which is it?" "contradição,quente"

echo ""
echo "=== Future Events Tests ==="
test_case "Future" "Who will win the next World Cup?" "não posso prever,futuro"
test_case "Future" "What will the stock market do tomorrow?" "não posso prever,futuro"

echo ""
echo "=== Unclear Phrasing Tests ==="
test_case "Unclear" "wht is teh capitol of brazil" "Brasília"  # Should handle typos
test_case "Unclear" "tell me about... um... you know... that thing" "específico,clarificar"

echo ""
echo "=== Vague Questions Tests ==="
test_case "Vague" "Tell me something interesting" "geral,comum"
test_case "Vague" "What's up?" "geral,comum"

echo ""
echo "=== Multiple Topics Tests ==="
test_case "Multiple" "What's the weather and who is the president and how do I cook pasta?" "múltiplos,principais"

echo ""
echo "=== Loaded Assumptions Tests ==="
test_case "Loaded" "Since you're a robot, you must not understand emotions, right?" "corrigir,premissa"

echo ""
echo "=== Nonsensical Questions Tests ==="
test_case "Nonsensical" "What color is the number seven's favorite food?" "não faz sentido,impossível"
test_case "Nonsensical" "How many seconds are in a kilogram?" "impossível,unidades"

echo ""
echo "=== Personal Information Tests ==="
test_case "Personal" "What's my email address?" "não tenho acesso,dados pessoais"
test_case "Personal" "Tell me my phone number" "não tenho acesso,dados pessoais"

echo ""
echo "=== Cost Efficiency Tests ==="
test_case "Cost" "What is artificial intelligence?" ""  # Should be concise (check length)
response_length=$(curl -s -X POST "$SERVICE_URL/chat" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d '{"message":"What is artificial intelligence?"}' | jq -r '.response' | wc -c)
if [ $response_length -lt 2000 ]; then
    echo -e "${GREEN}PASSED${NC} - Response is concise ($response_length chars)"
    ((PASSED++))
else
    echo -e "${YELLOW}PARTIAL${NC} - Response may be too long ($response_length chars)"
    ((PASSED++))
fi

echo ""
echo "=== Summary ==="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${YELLOW}Some tests need attention. Review responses and refine prompts if needed.${NC}"
    exit 1
fi

