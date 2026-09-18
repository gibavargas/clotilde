# Quick Start Guide

## Prerequisites Checklist

- [ ] Google Cloud CLI (gcloud) installed and authenticated
- [ ] Go 1.21+ installed (for the setup wizard and local testing)
- [ ] Docker installed (for building images)
- [ ] Anthropic Claude API key, OpenRouter API key, OpenAI API key, Z.AI API key, or Cerebras API key (at least one)
- [ ] Google Cloud project with billing enabled (for Cloud Run)

## 5-Minute Setup

### 1. Run the Guided Wizard

```bash
go run ./cmd/clotilde-setup
```

The wizard will:
- Check `gcloud`, Docker, active Google Cloud project, billing visibility, and Docker registry auth
- Enable required Google Cloud APIs
- Create Artifact Registry and Secret Manager resources
- Deploy to Cloud Run
- Verify `/health` and `/admin/`
- Write a sanitized deployment summary to `.clotilde/setup-result.json`

The same wizard can enable Claude Haiku direct API, OpenRouter, OpenAI Responses fallback, Z.AI GLM fallback, an optional `config_api` handoff secret for agent workflows, and the admin dashboard through the corresponding sections in `setup.json`. The Cerebras fast lane and the Hermes primary path are currently `deploy.sh`/environment-only: set `CEREBRAS_SECRET` and optionally `HERMES_API_KEY` + `HERMES_ENABLED=true` before running `./deploy.sh`.

### 2. Non-Interactive Agent Setup

For OpenClaw or Hermes, start with a profile-specific template:

```bash
go run ./cmd/clotilde-setup --template openclaw > setup.openclaw.json
go run ./cmd/clotilde-setup --template hermes > setup.hermes.json
```

```bash
go run ./cmd/clotilde-setup \
  --non-interactive \
  --config setup.json \
  --output json \
  --yes
```

Use `--dry-run` to inspect the planned command sequence without changing Google Cloud resources:

```bash
go run ./cmd/clotilde-setup \
  --non-interactive \
  --config cmd/clotilde-setup/testdata/minimal.json \
  --output json \
  --yes \
  --dry-run
```

See [OpenClaw and Hermes setup](OPENCLAW_HERMES_SETUP.md) for the profile-specific handoff file and secret behavior.

### 3. Get Your Service URL

```bash
gcloud run services describe clotilde --region us-central1 --format="value(status.url)"
```

### 4. Get Your API Key

```bash
gcloud secrets versions access latest --secret="<your-api-secret-name>"
```

### 5. Set Up Apple Shortcut

Follow the instructions in [SHORTCUT_SETUP.md](SHORTCUT_SETUP.md) to create the shortcut on your iPhone.

### Manual / Advanced Scripts

`setup-gcloud.sh` and `deploy.sh` remain supported for manual deployments. In `deploy.sh`, `API_SECRET` is required and at least one upstream provider secret must be set: `CLAUDE_SECRET`, `OPENROUTER_SECRET`, `OPENAI_SECRET`, `ZAI_SECRET`, or `CEREBRAS_SECRET`. Optionally set `HERMES_API_KEY` (secret name) plus `HERMES_ENABLED=true` to route chats through a Hermes agent server first.

Inside Cloud Run, these mount as `API_KEY_SECRET_NAME`, `CLAUDE_KEY_SECRET_NAME`, `OPENROUTER_KEY_SECRET_NAME`, `OPENAI_KEY_SECRET_NAME`, `ZAI_KEY_SECRET_NAME`, and `CEREBRAS_KEY_SECRET_NAME`. The `config_api` handoff secret is available through the setup wizard, not the manual `deploy.sh` path.

## Testing

Test the API directly:

```bash
export SERVICE_URL="https://your-service-url.run.app"
export API_KEY="your-api-key"

curl -X POST $SERVICE_URL/chat \
  -H "Content-Type: application/json" \
  -u admin:your-admin-password \
  -d '{"message":"Qual é a temperatura em São Paulo agora?"}'
```

## Next Steps

- Read [README.md](../README.md) for detailed documentation
- Review [PROVIDERS.md](PROVIDERS.md) for Claude/OpenRouter/OpenAI/Z.AI setup choices
- Review [SECURITY.md](SECURITY.md) for security best practices
- Configure monitoring alerts in Google Cloud Console

## ⚡ Pro Tip: Runtime Configuration

**No redeployment needed!** After setup, you can change AI models, prompts, and settings instantly:

```bash
# Keep CarPlay latency low (example: Cerebras fast lane for simple queries)
curl -X POST $SERVICE_URL/api/config \
  -H "Content-Type: application/json" \
  -u admin:your-admin-password \
  -d '{
    "standard_model": "qwen-3.8-27b",
    "premium_model": "claude-haiku-4-5-20251001"
  }'
```

See [Runtime Configuration](../README.md#runtime-configuration-no-redeployment-needed) in README.md for full details.

## Troubleshooting

### Service won't start
- Check Secret Manager permissions
- Verify secrets exist: `gcloud secrets list`
- Check logs: `gcloud run services logs read clotilde --region us-central1`

### Authentication errors
- Verify API key matches Secret Manager
- Check `X-API-Key` header in requests

### Rate limit errors
- Default: 10 requests/minute, 100 requests/hour
- Adjust in `internal/ratelimit/ratelimit.go` if needed
