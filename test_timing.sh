#!/bin/bash
# Timing test script to validate response times for CarPlay compatibility
# Apple Shortcuts has ~30s timeout, so all responses must complete within 25s
#
# Usage:
#   LOCAL:  ./test_timing.sh http://localhost:8080 YOUR_API_KEY
#   PROD:   ./test_timing.sh https://your-service.run.app YOUR_API_KEY
#
# Requirements:
#   - curl
#   - jq (optional, for pretty output)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check arguments
if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <BASE_URL> <API_KEY>"
    echo "Example: $0 http://localhost:8080 my-api-key"
    exit 1
fi

BASE_URL="$1"
API_KEY="$2"
CHAT_URL="${BASE_URL}/chat"

# Timeout threshold (Apple Shortcuts limit is ~30s, we aim for 25s max)
MAX_TIME=25

echo "=================================================="
echo "Clotilde Timing Test - CarPlay Compatibility"
echo "=================================================="
echo "Target: $CHAT_URL"
echo "Max allowed time: ${MAX_TIME}s (Apple Shortcuts limit: ~30s)"
echo ""

# Test cases: various complexity levels
declare -a TEST_CASES=(
    "Simple|Qual a capital do Brasil?"
    "Simple|Quanto é 5 vezes 7?"
    "Simple|Que horas são?"
    "Factual|Quem foi Santos Dumont?"
    "Factual|O que é a teoria da relatividade?"
    "Complex|Explique a crise climática e suas causas principais"
    "Complex|Quais os prós e contras de carros elétricos versus combustão?"
    "Complex|Compare democracia e autoritarismo"
    "Creative|Me sugira 3 drinks para uma festa"
    "Creative|Que música brasileira combina com um pôr do sol na praia?"
    "Math|Quanto é 1547 dividido por 23?"
    "Math|Se eu dirijo a 80km/h, quanto tempo levo para percorrer 200km?"
    "WebSearch|Qual o placar do último jogo do Flamengo?"
    "WebSearch|Como está o clima em São Paulo hoje?"
)

PASSED=0
FAILED=0
TOTAL=${#TEST_CASES[@]}

echo "Running $TOTAL test cases..."
echo ""

for test_case in "${TEST_CASES[@]}"; do
    # Parse category and question
    IFS='|' read -r category question <<< "$test_case"
    
    echo -n "[$category] \"${question:0:50}...\" "
    
    # Measure time
    START=$(date +%s.%N)
    
    # Make request
    RESPONSE=$(curl -s -w "\n%{http_code}\n%{time_total}" \
        -X POST "$CHAT_URL" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: $API_KEY" \
        -d "{\"message\":\"$question\"}" \
        --max-time $((MAX_TIME + 5)) 2>&1) || true
    
    END=$(date +%s.%N)
    
    # Parse response
    HTTP_CODE=$(echo "$RESPONSE" | tail -n2 | head -n1)
    TIME_TOTAL=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | head -n -2)
    
    # Check if timed out
    if [[ "$RESPONSE" == *"timed out"* ]] || [[ "$RESPONSE" == *"Operation timed out"* ]]; then
        echo -e "${RED}TIMEOUT (>${MAX_TIME}s)${NC}"
        ((FAILED++))
        continue
    fi
    
    # Check HTTP status
    if [ "$HTTP_CODE" != "200" ]; then
        echo -e "${RED}FAILED (HTTP $HTTP_CODE)${NC}"
        ((FAILED++))
        continue
    fi
    
    # Check time
    TIME_INT=$(echo "$TIME_TOTAL" | cut -d'.' -f1)
    if [ -z "$TIME_INT" ]; then
        TIME_INT=0
    fi
    
    if (( $(echo "$TIME_TOTAL > $MAX_TIME" | bc -l) )); then
        echo -e "${RED}SLOW (${TIME_TOTAL}s > ${MAX_TIME}s)${NC}"
        ((FAILED++))
    elif (( $(echo "$TIME_TOTAL > 15" | bc -l) )); then
        echo -e "${YELLOW}WARNING (${TIME_TOTAL}s)${NC}"
        ((PASSED++))
    else
        echo -e "${GREEN}OK (${TIME_TOTAL}s)${NC}"
        ((PASSED++))
    fi
    
    # Show response preview
    if command -v jq &> /dev/null; then
        RESP_TEXT=$(echo "$BODY" | jq -r '.response // .error // "No response"' 2>/dev/null | head -c 100)
    else
        RESP_TEXT=$(echo "$BODY" | head -c 100)
    fi
    echo "    → ${RESP_TEXT}..."
    echo ""
    
    # Small delay between requests to avoid rate limiting
    sleep 1
done

echo "=================================================="
echo "Results: $PASSED/$TOTAL passed"
echo ""

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}⚠ $FAILED tests exceeded time limit or failed${NC}"
    echo ""
    echo "Recommendations:"
    echo "1. Ensure Claude 4.5 Haiku is configured (fastest model)"
    echo "2. Check network latency to API providers"
    echo "3. Consider disabling web search for faster responses"
    echo "4. Review system prompts for complexity"
    exit 1
else
    echo -e "${GREEN}✓ All tests passed within ${MAX_TIME}s limit${NC}"
    echo ""
    echo "CarPlay compatibility: VERIFIED"
    exit 0
fi

