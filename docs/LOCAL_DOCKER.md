# Local Docker Development Guide

This guide explains how to run Clotilde locally using Docker without exposing secret information in the public GitHub repository.

## Prerequisites

- Docker installed
- Google Cloud CLI configured (for Secret Manager access, optional)
- Environment variables set (see below)

## Quick Start

### Option 1: Using Direct Values (Recommended for Local Testing)

This method uses environment variables with the actual API key values. The env var names follow the production convention, but the values are the raw keys. These are never committed to git.

1. **Create a local `.env` file** (already in `.gitignore`):
   ```bash
   cat > .env << EOF
   CLAUDE_KEY_SECRET_NAME=sk-ant-your-actual-claude-key-here
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
   export CLAUDE_SECRET_NAME=your-claude-secret-name
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
     -e CLAUDE_SECRET_NAME=$CLAUDE_SECRET_NAME \
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
| `API_KEY_SECRET_NAME` | Direct service API key value (local mode) | `abc123...` |
| `PORT` | Server port | `8080` |

Set at least one direct model provider key:

| Variable | Description | Example |
|----------|-------------|---------|
| `CLAUDE_KEY_SECRET_NAME` | Direct Anthropic API key value (recommended) | `sk-ant-...` |
| `OPENROUTER_KEY_SECRET_NAME` | Direct OpenRouter API key value | `sk-or-v1-...` |
| `OPENAI_KEY_SECRET_NAME` | Direct OpenAI API key value | `sk-...` |

### Alternative (Secret Manager Lookup)

If direct key values are not set, the app will try Secret Manager:

| Variable | Description | Example |
|----------|-------------|---------|
| `API_SECRET_NAME` | Name of API key secret in Secret Manager | `clotilde-auth-xyz789` |
| `CLAUDE_SECRET_NAME` | Name of Claude secret in Secret Manager | `clotilde-claude-abc123` |
| `OPENROUTER_SECRET_NAME` | Name of OpenRouter secret in Secret Manager | `clotilde-openrouter-abc123` |
| `OPENAI_SECRET_NAME` | Name of OpenAI secret in Secret Manager | `clotilde-oai-abc123` |
| `GOOGLE_CLOUD_PROJECT` | GCP project ID | `my-project-id` |

**Priority**: Direct values such as `CLAUDE_KEY_SECRET_NAME` take precedence over Secret Manager lookup. Configure at least one model provider: Claude, OpenRouter, or OpenAI.

Optional search provider variables follow the same pattern:

| Variable | Description | Example |
|----------|-------------|---------|
| `PERPLEXITY_KEY_SECRET_NAME` | Direct Perplexity API key value in local mode | `pplx-...` |
| `PERPLEXITY_SECRET_NAME` | Perplexity secret name in Secret Manager | `clotilde-perplexity-xyz789` |

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

### "No AI provider configured"

**Solution**: Either:
- Set `CLAUDE_KEY_SECRET_NAME` with the direct Anthropic value (preferred for local)
- Set `OPENROUTER_KEY_SECRET_NAME` or `OPENAI_KEY_SECRET_NAME` as a fallback provider
- Set `CLAUDE_SECRET_NAME`, `OPENROUTER_SECRET_NAME`, or `OPENAI_SECRET_NAME` with `GOOGLE_CLOUD_PROJECT` for Secret Manager lookup

### "Failed to create secret manager client"

**Solution**: 
- Run `gcloud auth application-default login`
- Ensure GCP credentials are mounted in Docker (see Option 2)

### "Failed to get provider API key"

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

- See [README.md](../README.md) for full documentation
- See [SECURITY.md](SECURITY.md) for security best practices
- See [agents.md](agents.md) for secret name configuration details
