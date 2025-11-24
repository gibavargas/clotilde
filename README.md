# Clotilde CarPlay Assistant

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A voice-activated CarPlay assistant powered by GPT-5 with web search capabilities. Built with Go for minimal resource usage and deployed on Google Cloud Run with Artifact Registry.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Features

- 🚗 CarPlay integration via Apple Shortcuts
- 🧠 GPT-5 with web browsing for real-time information
- 🇧🇷 Brazilian Portuguese responses (default)
- 🔒 Security-first design with API key authentication, rate limiting, and input validation
- 💰 Free tier optimized (Google Cloud Artifact Registry + Cloud Run)
- 🐳 Minimal Docker image (~10-15MB)

## Architecture

- **Backend**: Go HTTP server with minimal dependencies
- **Deployment**: Google Cloud Run (256MB RAM, 1 CPU)
- **Container Registry**: Google Cloud Artifact Registry (free tier)
- **Secrets**: Google Secret Manager
- **Client**: Apple Shortcut on iPhone/CarPlay

## Prerequisites

- Google Cloud CLI (gcloud) installed and configured
- Go 1.21+ installed
- Docker installed (for local testing)
- OpenAI API key
- Apple iPhone with Shortcuts app

## Setup

### 1. Google Cloud Setup

#### Create Artifact Registry Repository

```bash
# Set your project ID
export PROJECT_ID=your-project-id
export REGION=us-central1
export REPO_NAME=clotilde-repo

# Create Artifact Registry repository
gcloud artifacts repositories create $REPO_NAME \
    --repository-format=docker \
    --location=$REGION \
    --description="Clotilde Docker images"
```

#### Create Secret Manager Secrets

```bash
# Create OpenAI API key secret
echo -n "your-openai-api-key" | gcloud secrets create openai-api-key \
    --data-file=- \
    --replication-policy="automatic"

# Create API key for authenticating requests
echo -n "your-secure-api-key-here" | gcloud secrets create clotilde-api-key \
    --data-file=- \
    --replication-policy="automatic"
```

#### Grant Cloud Run Service Account Access

```bash
# Get the Cloud Run service account email
export SERVICE_ACCOUNT=$(gcloud iam service-accounts list --filter="displayName:Compute Engine default service account" --format="value(email)")

# Grant Secret Manager access
gcloud secrets add-iam-policy-binding openai-api-key \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/secretmanager.secretAccessor"

gcloud secrets add-iam-policy-binding clotilde-api-key \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/secretmanager.secretAccessor"
```

### 2. Build and Deploy

#### Option A: Using Cloud Build (Recommended)

```bash
# Submit build
gcloud builds submit --config=cloudbuild.yaml \
    --substitutions=_REGION=$REGION,_REPO_NAME=$REPO_NAME,_SERVICE_NAME=clotilde
```

#### Option B: Manual Build and Deploy

```bash
# Build Docker image
docker build -t $REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/clotilde:latest .

# Configure Docker authentication
gcloud auth configure-docker $REGION-docker.pkg.dev

# Push to Artifact Registry
docker push $REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/clotilde:latest

# Deploy to Cloud Run
gcloud run deploy clotilde \
    --image $REGION-docker.pkg.dev/$PROJECT_ID/$REPO_NAME/clotilde:latest \
    --region $REGION \
    --platform managed \
    --allow-unauthenticated \
    --memory 256Mi \
    --cpu 1 \
    --min-instances 0 \
    --max-instances 10 \
    --timeout 30 \
    --set-env-vars GOOGLE_CLOUD_PROJECT=$PROJECT_ID,PORT=8080 \
    --set-secrets OPENAI_KEY_SECRET_NAME=openai-api-key:latest,API_KEY_SECRET_NAME=clotilde-api-key:latest
```

### 3. Get Your Service URL

```bash
gcloud run services describe clotilde --region $REGION --format="value(status.url)"
```

Save this URL - you'll need it for the Apple Shortcut.

### 4. Environment Variables

For local development, create a `.env` file from the template:

```bash
cp .env.example .env
# Edit .env with your actual values
```

**Important**: Never commit `.env` files to git. They are already in `.gitignore`.

The `.env` file should contain:
- `OPENAI_KEY_SECRET_NAME`: Your OpenAI API key
- `API_KEY_SECRET_NAME`: Your Clotilde service API key (get from Secret Manager)
- `GOOGLE_CLOUD_PROJECT`: Your Google Cloud project ID
- `SERVICE_URL`: Your deployed service URL (optional, for testing)

### 5. Local Development (Optional)

For local testing:

1. **Create `.env` file** (from `.env.example`):
   ```bash
   cp .env.example .env
   # Edit .env with your actual values
   ```

2. **Load environment variables and run**:
   ```bash
   # Option A: Use .env file (requires a tool like direnv or manually source)
   export $(cat .env | xargs)
   export PORT=8080
   go run cmd/clotilde/main.go
   
   # Option B: Set directly
   export OPENAI_KEY_SECRET_NAME=your-openai-key
   export API_KEY_SECRET_NAME=your-api-key
   export GOOGLE_CLOUD_PROJECT=your-project-id
   export PORT=8080
   go run cmd/clotilde/main.go
   ```

**Note**: 
- Local development can use environment variables directly or fall back to Secret Manager
- Production uses Secret Manager via Cloud Run secret mounting
- **Never commit `.env` files** - they're in `.gitignore`

## Apple Shortcut Setup

### Method 1: Import Shortcut File

1. Open the `Clotilde.shortcut` file on your iPhone
2. Configure the API key and service URL
3. Enable "Show in CarPlay" in Shortcut settings
4. Set Siri phrase: "Falar com Clotilde"

### Method 2: Manual Setup

1. Open Shortcuts app on iPhone
2. Create new shortcut named "Clotilde"
3. Add actions in this order:
   - **Dictate Text** (configure for CarPlay)
   - **Get Contents of URL**
     - Method: POST
     - URL: `https://your-service-url.run.app/chat`
     - Headers:
       - `Content-Type: application/json`
       - `X-API-Key: your-api-key-from-secret-manager`
     - Request Body: JSON
       ```json
       {
         "message": "[Dictated Text]"
       }
       ```
   - **Get Dictionary from Input** (parse JSON response)
   - **Get Value for "response"** (extract response text)
   - **Speak Text** (read the response)
4. In Shortcut settings:
   - Enable "Show in CarPlay"
   - Add Siri phrase: "Falar com Clotilde"

## API Usage

### Endpoint

```
POST /chat
```

### Headers

```
Content-Type: application/json
X-API-Key: your-api-key
```

### Request Body

```json
{
  "message": "Qual é o preço do petróleo hoje?"
}
```

### Response

```json
{
  "response": "O preço do petróleo hoje está em torno de..."
}
```

### Error Response

```json
{
  "error": "Error message"
}
```

## Security Features

- **API Key Authentication**: All requests require valid API key
- **Rate Limiting**: 10 requests/minute per API key, 100 requests/hour per IP
- **Input Validation**: Max 1000 characters per message, 5KB request size limit
- **Secrets Management**: All sensitive data in Google Secret Manager
- **Secure Logging**: No sensitive data in logs (only metadata)
- **HTTPS Only**: Enforced by Cloud Run
- **Non-root Container**: Runs as unprivileged user
- **No Secrets in Code**: All API keys and sensitive data use environment variables or Secret Manager
- **Git-Safe**: `.env` files and sensitive documentation excluded from version control

**Before Sharing on GitHub**: All API keys have been replaced with placeholders (`YOUR_API_KEY`, `YOUR_SERVICE_URL`) in documentation files. Sensitive files are excluded via `.gitignore`.

See [SECURITY.md](SECURITY.md) for detailed security documentation.

## Resource Usage

- **Docker Image**: ~10-15MB (Alpine-based)
- **Memory**: 256MB (Cloud Run minimum)
- **CPU**: 1 vCPU
- **Artifact Registry**: Free tier (0.5GB storage, 1GB egress/month)

## Troubleshooting

### Service won't start

- Check Secret Manager permissions for Cloud Run service account
- Verify secrets exist: `gcloud secrets list`
- Check logs: `gcloud run services logs read clotilde --region $REGION`

### Authentication errors

- Verify API key in Secret Manager matches the one in your Shortcut
- Check `X-API-Key` header is set correctly

### Rate limit errors

- Default: 10 requests/minute, 100 requests/hour
- Adjust in `internal/ratelimit/ratelimit.go` if needed

## License

Private project - All rights reserved

