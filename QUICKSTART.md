# Quick Start Guide

## Prerequisites Checklist

- [ ] Google Cloud CLI (gcloud) installed and authenticated
- [ ] Go 1.21+ installed (for local testing)
- [ ] Docker installed (for building images)
- [ ] OpenAI API key
- [ ] Google Cloud project with billing enabled (for Cloud Run)

## 5-Minute Setup

### 1. Run Setup Script

```bash
chmod +x setup-gcloud.sh
./setup-gcloud.sh
```

This will:
- Enable required Google Cloud APIs
- Create Artifact Registry repository
- Create secrets in Secret Manager
- Configure IAM permissions

### 2. Deploy to Cloud Run

```bash
chmod +x deploy.sh
./deploy.sh
```

Or use Cloud Build:

```bash
gcloud builds submit --config=cloudbuild.yaml
```

### 3. Get Your Service URL

```bash
gcloud run services describe clotilde --region us-central1 --format="value(status.url)"
```

### 4. Get Your API Key

```bash
gcloud secrets versions access latest --secret="clotilde-api-key"
```

### 5. Set Up Apple Shortcut

Follow the instructions in [SHORTCUT_SETUP.md](SHORTCUT_SETUP.md) to create the shortcut on your iPhone.

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

