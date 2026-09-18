# Clotilde — CarPlay voice relay to Hermes

Shortcut (CarPlay) → Cloud Run (`/chat`, X-API-Key) → Hermes gateway
(`deepseek-v4.1-flash`) → spoken answer. That is the whole system.

```
Apple Shortcut ──POST /chat──▶ Cloud Run ──POST /v1/chat/completions──▶ Hermes gateway (Oracle box)
                                 │  │                                        model: deepseek-v4.1-flash
                                 │  └─ fresh-data? run SERP harness (≤9s),   (loopback :8642 behind a TLS
                                 │     attach sanitized evidence to prompt    funnel; bearer-authenticated)
                                 └─ URL-strip answer, log, return JSON
```

## What it does

1. **Auth + sanitize**: X-API-Key, prompt-injection sanitizer, IP/key rate
   limits, request-size cap (kept from the old stack — small, tested).
2. **Fresh-data gate**: a ~30-keyword heuristic decides if the question is
   about anything that rots (price/score/weather/news/"hoje"/"agora"). If
   yes, the Go SERP harness (Bing → DDG → Mojeek) runs in parallel and the
   sanitized evidence is attached to the system prompt, so Hermes answers
   grounded **in one round trip**. If no, the prompt goes straight out.
3. **Relay**: single POST to the gateway, 25s budget inherited from the
   Shortcut's ~30s. No fallback chain — Hermes is the only provider.
4. **Voice-safe output**: URLs/domains stripped, 1-2 sentence answers.

## Files

- `cmd/clotilde/main.go` — the whole server (~800 lines: relay, SERP gate,
  evidence sanitizer, metrics page)
- `cmd/clotilde/prompts.go` — the ONE persona prompt (PT-BR, MODO RÁDIO)
- `internal/` — auth, ratelimit, validator, promptinjection, logging,
  clientip, queryrewrite, websearch (all small, all tested)
- `deploy.sh` — build + deploy (env-driven, see header)

## Deploy

```bash
export API_SECRET=<service-key-secret-name>
export HERMES_SECRET=<hermes-bearer-secret-name>
./deploy.sh
```

`--set-env-vars`/`--set-secrets` REPLACE the whole set: to add/remove a
secret, re-run deploy with the full list you want.

## Security posture (verified 2026-09-18)

- Hermes gateway `:8642` binds loopback; direct ports (8642/9900) are
  filtered from the internet (verified externally). Internet path is the
  Tailscale Funnel TLS endpoint, bearer-authenticated (`API_SERVER_KEY` on
  the box); unauthenticated POSTs get 401.
- Cloud Run: `/health` anonymous, `/chat` requires X-API-Key,
  `/metrics` requires an internal bearer token.
- Secrets live in Google Secret Manager; IP hashes are salted
  (`IP_HASH_SALT`); logs are metadata-only by default.

## Changing the model

One line in `cmd/clotilde/main.go`:

```go
const hermesModel = "deepseek-v4.1-flash"
```

The gateway resolves model ids per request from its provider registry — no
gateway-side config change needed.

## Testing

```bash
go build ./... && go test ./...
```

Local smoke: `HERMES_KEY_SECRET_NAME=<direct-value> API_KEY_SECRET_NAME=<direct-value> go run ./cmd/clotilde`
