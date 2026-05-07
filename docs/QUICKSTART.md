# Quick Start Guide

## Prerequisites Checklist

- [ ] Google Cloud CLI (gcloud) installed and authenticated
- [ ] Go 1.21+ installed (for the setup wizard and local testing)
- [ ] Docker installed (for building images)
- [ ] Anthropic Claude API key, OpenRouter API key, or OpenAI API key
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

The same wizard can enable Claude Haiku direct API, OpenRouter, OpenAI Responses fallback, Perplexity search, an optional `config_api` handoff secret for agent workflows, and the admin dashboard through the corresponding sections in `setup.json`.

### 2. Non-Interactive Agent Setup

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

`setup-gcloud.sh` and `deploy.sh` remain supported for manual deployments. In `deploy.sh`, `API_SECRET` is required and at least one upstream provider secret must be set: `CLAUDE_SECRET`, `OPENROUTER_SECRET`, or `OPENAI_SECRET`.

Inside Cloud Run, these mount as `API_KEY_SECRET_NAME`, `CLAUDE_KEY_SECRET_NAME`, `OPENROUTER_KEY_SECRET_NAME`, and `OPENAI_KEY_SECRET_NAME`. Optional Perplexity search is configured with `PERPLEXITY_SECRET_NAME`. The `config_api` handoff secret is available through the setup wizard, not the manual `deploy.sh` path.

## Testing

Test the API directly:

```bash
export SERVICE_URL="https://your-service-url.run.app"
export API_KEY="your-api-key"

curl -X POST $SERVICE_URL/chat \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"message":"Qual é a temperatura em São Paulo agora?"}'
```

## Next Steps

- Read [README.md](../README.md) for detailed documentation
- Review [PROVIDERS.md](PROVIDERS.md) for Claude/OpenRouter/OpenAI setup choices
- Review [SECURITY.md](SECURITY.md) for security best practices
- Configure monitoring alerts in Google Cloud Console

## ⚡ Pro Tip: Runtime Configuration

**No redeployment needed!** After setup, you can change AI models, prompts, and settings instantly:

```bash
# Keep CarPlay latency low with Claude Haiku
curl -X POST $SERVICE_URL/api/config \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "standard_model": "claude-haiku-4-5-20251001",
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
