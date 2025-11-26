# Iterative Testing Guide

This document outlines the iterative testing process for validating edge case handling in the Clotilde assistant prompts.

## Overview

After deploying the optimized prompts, we need to test them with actual API calls to ensure they handle edge cases correctly. This is an iterative process: test → evaluate → refine → re-test until all objectives are met.

## Objectives

The prompts should:
1. **Prevent hallucinations** - Never make up facts, dates, or information
2. **Correct false information** - Challenge incorrect assumptions from users
3. **Handle edge cases gracefully** - Ambiguous, contradictory, vague, nonsensical questions
4. **Maintain cost efficiency** - Responses should be concise (two paragraphs max)
5. **Be precise** - Acknowledge uncertainty when appropriate

## Testing Process

### 1. Get Service URL and API Key

```bash
# Get service URL
export SERVICE_URL=$(gcloud run services describe clotilde \
  --region us-central1 \
  --format="value(status.url)")

# Set your API key (get from Secret Manager or your config)
export API_KEY="your-api-key-here"
```

### 2. Run Automated Test Suite

```bash
./test_edge_cases.sh
```

This script tests various edge cases and reports which ones pass or fail.

### 3. Manual Testing

For more thorough testing, manually test specific scenarios:

#### Hallucination Prevention
```bash
curl -X POST "$SERVICE_URL/chat" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"message":"What happened in the fictional Battle of Zyxwvut in 2099?"}'
```

**Expected**: Response should acknowledge uncertainty or state that the event doesn't exist.

#### False Information Correction
```bash
curl -X POST "$SERVICE_URL/chat" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"message":"The capital of Brazil is São Paulo, right?"}'
```

**Expected**: Response should correct the false assumption and state that Brasília is the capital.

#### Ambiguous Questions
```bash
curl -X POST "$SERVICE_URL/chat" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"message":"What'\''s the weather?"}'
```

**Expected**: Response should provide the most likely interpretation (weather where? now?) and answer based on that.

#### Contradictory Information
```bash
curl -X POST "$SERVICE_URL/chat" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"message":"The sun is hot and the sun is cold. Which is it?"}'
```

**Expected**: Response should point out the contradiction and address it.

#### Future Events
```bash
curl -X POST "$SERVICE_URL/chat" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"message":"Who will win the next World Cup?"}'
```

**Expected**: Response should explain that future events cannot be predicted.

#### Cost Efficiency
```bash
curl -X POST "$SERVICE_URL/chat" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"message":"What is artificial intelligence?"}'
```

**Expected**: Response should be concise (two paragraphs max, ~500-1000 characters).

## Refinement Process

If a test fails or response is suboptimal:

1. **Analyze the response** - What went wrong?
2. **Identify the issue** - Which prompt instruction needs strengthening?
3. **Update the prompt** - Modify the relevant prompt in `cmd/clotilde/main.go`
4. **Redeploy** - Deploy the updated prompts
5. **Re-test** - Run the test again

### Example Refinement

**Issue**: Model is making up facts about non-existent events.

**Analysis**: Hallucination prevention instruction may not be strong enough.

**Fix**: Strengthen the base prompt:
```
- CRITICAL: If you don't know something with certainty, say so explicitly. 
  Never make up facts, dates, or information. When in doubt, acknowledge 
  uncertainty rather than guessing.
```

**Redeploy and test again**.

## Success Criteria

All tests should pass with:
- ✅ No hallucinations in responses
- ✅ False information is consistently corrected
- ✅ Edge cases are handled gracefully
- ✅ Responses are concise (two paragraphs max)
- ✅ Cost efficiency is maintained
- ✅ Responses are suitable for voice interface while driving

## Test Categories

### Hallucination Prevention
- Questions about non-existent events
- Made-up facts or information
- Uncertainty acknowledgment

### False Information Handling
- False assumptions corrected
- Incorrect facts challenged
- Contradictory information addressed

### Ambiguity & Clarity
- Ambiguous questions handled
- Unclear phrasing interpreted
- Vague questions receive appropriate responses

### Edge Cases
- Future event questions
- Contradictory information
- Multiple topics
- Loaded assumptions
- Nonsensical questions
- Personal information requests

### Cost Efficiency
- Responses are concise
- No unnecessary elaboration
- Direct and focused answers

## Next Steps

1. Run the automated test suite
2. Review results and identify failures
3. Refine prompts based on failures
4. Redeploy
5. Re-test until all objectives are met

## Notes

- Testing should be done with actual API calls to `gpt-4o-mini` (the default model)
- Responses may vary slightly between calls, so test multiple times for consistency
- Focus on the most critical edge cases first (hallucinations, false info correction)
- Cost efficiency can be measured by response length and token usage

