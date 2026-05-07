# Provider Configuration

Clotilde is designed around low-latency CarPlay use. Configure at least one model provider.

## Recommended Order

1. `CLAUDE_KEY_SECRET_NAME`: Anthropic API key for direct Claude Haiku 4.5. Runtime model: `claude-haiku-4-5-20251001`.
2. `OPENROUTER_KEY_SECRET_NAME`: OpenRouter API key for hosted provider routing. Runtime model: `openrouter/anthropic/claude-haiku-4.5`.
3. `OPENAI_KEY_SECRET_NAME`: OpenAI API key for Responses API fallback and native `web_search`.

The runtime accepts direct key values in local development. In Cloud Run, the same environment variable names are mounted from Secret Manager by `deploy.sh` or the setup wizard.

## OpenAI OAuth

OpenAI API calls from this backend are server-to-server bearer requests. Use an OpenAI API key for `OPENAI_KEY_SECRET_NAME`.

OAuth can still sit in front of Clotilde if you expose the service to a GPT Action or another user-facing integration. That OAuth flow authenticates the user to your action or app; it does not replace Clotilde's upstream OpenAI API key or Clotilde's own `X-API-Key` service authentication.

## Web Search

For time-sensitive requests, Clotilde tries provider-appropriate search:

- Direct Claude: Claude native `web_search_20250305`
- Perplexity: optional formatted search results injected into the prompt
- OpenAI: Responses API `web_search`
- OpenRouter: `openrouter:web_search` server tool

If a configured provider cannot perform search, Clotilde falls back to another configured search-capable provider before returning an error.

## Runtime Models

Use `/api/config` or `/admin/` to switch models without redeploying.

```json
{
  "standard_model": "claude-haiku-4-5-20251001",
  "premium_model": "claude-haiku-4-5-20251001"
}
```

OpenRouter models use the `openrouter/<provider>/<model>` prefix:

```json
{
  "standard_model": "openrouter/anthropic/claude-haiku-4.5",
  "premium_model": "openrouter/anthropic/claude-haiku-4.5"
}
```
