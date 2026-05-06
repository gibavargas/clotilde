# Quick Start Guide

## Prerequisites Checklist

- [ ] Google Cloud CLI (gcloud) installed and authenticated
- [ ] Go 1.21+ installed (for local testing)
- [ ] Docker installed (for building images)
- [ ] OpenAI API key
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

`setup-gcloud.sh` and `deploy.sh` remain supported for manual deployments. In those scripts, `OPENAI_SECRET` and `API_SECRET` are Secret Manager secret names; inside Cloud Run, `OPENAI_KEY_SECRET_NAME` and `API_KEY_SECRET_NAME` are mounted secret values consumed by the service.

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

- Read [README.md](README.md) for detailed documentation
- Review [SECURITY.md](SECURITY.md) for security best practices
- Configure monitoring alerts in Google Cloud Console

## ⚡ Pro Tip: Runtime Configuration

**No redeployment needed!** After setup, you can change AI models, prompts, and settings instantly:

```bash
# Switch to faster models for better performance
curl -X POST $SERVICE_URL/api/config \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "standard_model": "gpt-4.1-mini",
    "premium_model": "gpt-4.1"
  }'
```

See [Runtime Configuration](#runtime-configuration-no-redeployment-needed) in README.md for full details.

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
