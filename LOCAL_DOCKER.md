# Local Docker Development Guide

This guide explains how to run Clotilde locally using Docker without exposing secret information in the public GitHub repository.

## Prerequisites

- Docker installed
- Google Cloud CLI configured (for Secret Manager access, optional)
- Environment variables set (see below)

## Quick Start

### Option 1: Using Direct Secret Values (Recommended for Local Testing)

This method uses environment variables with the actual secret values. These are never committed to git.

1. **Create a local `.env` file** (already in `.gitignore`):
   ```bash
   cat > .env << EOF
   OPENAI_KEY_SECRET_NAME=sk-your-actual-openai-key-here
   API_KEY_SECRET_NAME=your-actual-api-key-here
   GOOGLE_CLOUD_PROJECT=your-project-id
   PORT=8080
   EOF
   ```

2. **Build and run with Docker**:
   ```bash
   # Build the image
   docker build -t clotilde:local .
   
   # Run with environment variables from .env file
   docker run --env-file .env -p 8080:8080 clotilde:local
   ```

3. **Test the service**:
   ```bash
   curl -X POST http://localhost:8080/chat \
     -H "Content-Type: application/json" \
     -H "X-API-Key: your-actual-api-key-here" \
     -d '{"message":"teste"}'
   ```

### Option 2: Using Secret Manager (Production-like)

This method uses Secret Manager, requiring GCP authentication and secret names.

1. **Set environment variables**:
   ```bash
   export OPENAI_SECRET_NAME=your-openai-secret-name
   export API_SECRET_NAME=your-api-secret-name
   export GOOGLE_CLOUD_PROJECT=your-project-id
   export PORT=8080
   ```

2. **Authenticate with GCP**:
   ```bash
   gcloud auth application-default login
   ```

3. **Build and run**:
   ```bash
   # Build the image
   docker build -t clotilde:local .
   
   # Run with environment variables
   docker run \
     -e OPENAI_SECRET_NAME=$OPENAI_SECRET_NAME \
     -e API_SECRET_NAME=$API_SECRET_NAME \
     -e GOOGLE_CLOUD_PROJECT=$GOOGLE_CLOUD_PROJECT \
     -e PORT=8080 \
     -v ~/.config/gcloud:/root/.config/gcloud:ro \
     -p 8080:8080 \
     clotilde:local
   ```

   **Note**: The `-v ~/.config/gcloud:/root/.config/gcloud:ro` mounts your GCP credentials into the container (read-only).

## Environment Variables

### Required for Local Development

| Variable | Description | Example |
|----------|-------------|---------|
| `OPENAI_KEY_SECRET_NAME` | Direct OpenAI API key value (preferred) | `sk-...` |
| `API_KEY_SECRET_NAME` | Direct API key value (preferred) | `abc123...` |
| `PORT` | Server port | `8080` |

### Alternative (Secret Manager Lookup)

If `OPENAI_KEY_SECRET_NAME` and `API_KEY_SECRET_NAME` are not set, the app will try Secret Manager:

| Variable | Description | Example |
|----------|-------------|---------|
| `OPENAI_SECRET_NAME` | Name of OpenAI secret in Secret Manager | `clotilde-oai-abc123` |
| `API_SECRET_NAME` | Name of API key secret in Secret Manager | `clotilde-auth-xyz789` |
| `GOOGLE_CLOUD_PROJECT` | GCP project ID | `my-project-id` |

**Priority**: Direct values (`OPENAI_KEY_SECRET_NAME`) take precedence over Secret Manager lookup.

## Docker Compose (Optional)

Create a `docker-compose.yml` for easier local development:

```yaml
version: '3.8'

services:
  clotilde:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    environment:
      - PORT=8080
    # Uncomment if using Secret Manager:
    # volumes:
    #   - ~/.config/gcloud:/root/.config/gcloud:ro
```

Run with:
```bash
docker-compose up
```

## Security Notes

1. **Never commit `.env` files**: They're in `.gitignore` for a reason
2. **Use direct values for local dev**: Faster, no GCP dependency
3. **Use Secret Manager for production-like testing**: Tests the actual deployment flow
4. **Rotate secrets regularly**: Update your local `.env` when secrets change

## Troubleshooting

### "OPENAI_SECRET_NAME environment variable not set"

**Solution**: Either:
- Set `OPENAI_KEY_SECRET_NAME` with the direct value (preferred for local)
- Set `OPENAI_SECRET_NAME` and `GOOGLE_CLOUD_PROJECT` for Secret Manager lookup

### "Failed to create secret manager client"

**Solution**: 
- Run `gcloud auth application-default login`
- Ensure GCP credentials are mounted in Docker (see Option 2)

### "Failed to get OpenAI API key"

**Solution**:
- Check secret name is correct
- Verify IAM permissions: `gcloud secrets get-iam-policy YOUR_SECRET_NAME`
- Ensure `GOOGLE_CLOUD_PROJECT` is set correctly

### Port already in use

**Solution**: Change the port:
```bash
export PORT=8081
docker run -e PORT=$PORT -p 8081:8081 ...
```

## Next Steps

- See [README.md](README.md) for full documentation
- See [SECURITY.md](SECURITY.md) for security best practices
- See [agents.md](agents.md) for secret name configuration details

