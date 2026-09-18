# Provider Configuration

Clotilde is designed around low-latency CarPlay use. Configure at least one model provider.

## Recommended Order

1. `HERMES_API_KEY` (+ `HERMES_ENABLED=true`): optional Hermes primary path. When mounted and enabled, every chat is tried against the Hermes agent server (OpenAI-compatible, model alias `clotilde`) first; any failure falls through to the provider chain below. On the gateway side, the `clotilde` alias is a `model_routes` entry: since 2026-09-18 it routes to `deepseek-v4.1-flash` via the `opencode-go` provider (previously `qwen-3.8-27b` via Cerebras, dropped after quota exhaustion). The agent loop returns only the final answer, so reasoning-model internals never reach the Shortcut client, and the gateway-side SOUL persona currently carries the no-refusal/web-search behavior until the equivalent OPERADOR blocks in `cmd/clotilde/prompts.go` are deployed from this repo.
2. `CEREBRAS_KEY_SECRET_NAME`: Cerebras API key for the fast lane. Runtime models: `qwen-3.8-27b` (default standard model), `gemma-4-31b`, and `gpt-oss-120b`. These models have no server-side search tool and are grounded by the Go-native web search harness.
3. `CLAUDE_KEY_SECRET_NAME`: Anthropic API key for direct Claude Haiku 4.5. Runtime model: `claude-haiku-4-5-20251001`. First in the ordered fallback chain.
4. `OPENROUTER_KEY_SECRET_NAME`: OpenRouter API key for hosted provider routing. Runtime model: `openrouter/anthropic/claude-haiku-4.5`.
5. `OPENAI_KEY_SECRET_NAME`: OpenAI API key for Responses API fallback and native `web_search`.
6. `ZAI_KEY_SECRET_NAME`: Z.AI API key for GLM chat-completions fallback. Runtime model: `zai/glm-4.7-flash`.

Per request, dispatch is: Hermes (if enabled) → routed model → ordered fallback chain (Claude → OpenRouter → Z.AI → OpenAI), so an upstream outage degrades the answer instead of failing the request.

The runtime accepts direct key values in local development. In Cloud Run, the same environment variable names are mounted from Secret Manager by `deploy.sh` or the setup wizard (`HERMES_API_KEY` is gated behind `HERMES_ENABLED=true` in `deploy.sh`).

## OpenAI OAuth

OpenAI API calls from this backend are server-to-server bearer requests. Use an OpenAI API key for `OPENAI_KEY_SECRET_NAME`.

OAuth can still sit in front of Clotilde if you expose the service to a GPT Action or another user-facing integration. That OAuth flow authenticates the user to your action or app; it does not replace Clotilde's upstream OpenAI API key or Clotilde's own `X-API-Key` service authentication.

## Web Search

For time-sensitive requests, Clotilde grounds answers through, in order of preference:

- **Go-native harness** (`internal/websearch`): SERP scraping (Bing first, DuckDuckGo HTML/lite and Mojeek as fallbacks) plus concurrent page extraction under a shared 9-second budget. Used by default and always for Cerebras models. Queries are rewritten first by `internal/queryrewrite`.
- Direct Claude: Claude native `web_search_20250305`
- OpenAI: Responses API `web_search`
- Z.AI: chat-completions `web_search` tool
- OpenRouter: `openrouter:web_search` server tool

By default every request is grounded. Set `SEARCH_ONLY_CATEGORIES=true` to search only for `web_search`/`factual` categories (faster simple queries). If a configured provider cannot perform search, Clotilde falls back to another configured search-capable provider before returning an error. The optional Camoufox harness can render public result pages when native search fails or returns blocked/JavaScript-gated snippets.

## Runtime Models

Use `/api/config` (writes require the admin Basic credential) or `/admin/` to switch models without redeploying.

```json
{
  "standard_model": "qwen-3.8-27b",
  "premium_model": "claude-haiku-4-5-20251001"
}
```

Cerebras models (no prefix): `qwen-3.8-27b`, `gemma-4-31b`, `gpt-oss-120b`.

OpenRouter models use the `openrouter/<provider>/<model>` prefix:

```json
{
  "standard_model": "openrouter/anthropic/claude-haiku-4.5",
  "premium_model": "openrouter/anthropic/claude-haiku-4.5"
}
```

Z.AI models use the `zai/<model>` prefix, for example `zai/glm-4.7-flash`.
