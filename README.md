# Clotilde CarPlay Assistant

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A voice-activated CarPlay assistant powered by GPT-5 with web search capabilities. Built with Go for minimal resource usage and deployed on Google Cloud Run with Artifact Registry.

## Features

- 🚗 CarPlay integration via Apple Shortcuts
- 🧠 OpenAI Responses API with web search for real-time information
- 🔍 **Perplexity AI Search API integration** - Alternative web search provider with toggle control
- 🇧🇷 Brazilian Portuguese responses (default)
- 🔒 Security-first design with API key authentication, rate limiting, and input validation
- 💰 Free tier optimized (Google Cloud Artifact Registry + Cloud Run)
- 🐳 Minimal Docker image (~14.9MB)
- 📊 Admin dashboard for monitoring logs and usage statistics
- ⚙️ **Dynamic configuration**: Change system prompt and models without redeployment
- 🔍 Request tracing with unique request IDs

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

**Important**: Use unique, unpredictable secret names for security.

```bash
# Generate unique secret names (recommended for security)
export OPENAI_SECRET="my-openai-key-$(openssl rand -hex 4)"
export API_SECRET="my-api-key-$(openssl rand -hex 4)"

# Create OpenAI API key secret
echo -n "your-openai-api-key" | gcloud secrets create $OPENAI_SECRET \
    --data-file=- \
    --replication-policy="automatic"

# Create API key for authenticating requests
echo -n "your-secure-api-key-here" | gcloud secrets create $API_SECRET \
    --data-file=- \
    --replication-policy="automatic"

# Create Perplexity API key secret (optional - for web search)
export PERPLEXITY_SECRET="my-perplexity-key-$(openssl rand -hex 4)"
echo -n "your-perplexity-api-key" | gcloud secrets create $PERPLEXITY_SECRET \
    --data-file=- \
    --replication-policy="automatic"

# Save your secret names securely (you'll need them for deployment)
echo "OPENAI_SECRET=$OPENAI_SECRET"
echo "API_SECRET=$API_SECRET"
echo "PERPLEXITY_SECRET=$PERPLEXITY_SECRET"
```

#### Grant Cloud Run Service Account Access

```bash
# Get the Cloud Run service account email
export SERVICE_ACCOUNT=$(gcloud iam service-accounts list --filter="displayName:Compute Engine default service account" --format="value(email)")

# Grant Secret Manager access (use your secret names from above)
gcloud secrets add-iam-policy-binding $OPENAI_SECRET \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/secretmanager.secretAccessor"

gcloud secrets add-iam-policy-binding $API_SECRET \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/secretmanager.secretAccessor"

# Grant Secret Manager access for Perplexity API key (if created)
if [ ! -z "$PERPLEXITY_SECRET" ]; then
    gcloud secrets add-iam-policy-binding $PERPLEXITY_SECRET \
        --member="serviceAccount:${SERVICE_ACCOUNT}" \
        --role="roles/secretmanager.secretAccessor"
fi
```

### 2. Build and Deploy

#### Option A: Using deploy.sh (Recommended)

```bash
# Set required environment variables
export OPENAI_SECRET=your-openai-secret-name
export API_SECRET=your-api-secret-name

# Optional: Enable admin dashboard
export ADMIN_USER=admin
export ADMIN_SECRET=your-admin-password-secret-name
export LOG_BUFFER_SIZE=1000

# Deploy
chmod +x deploy.sh
./deploy.sh
```

#### Option B: Using Cloud Build (Deprecated)

> **Note**: Cloud Build is deprecated in favor of the local `deploy.sh` script. This option is kept for reference but is not recommended for new deployments.

```bash
# Submit build (use your secret names from the setup step)
gcloud builds submit --config=cloudbuild.yaml \
    --substitutions=_REGION=$REGION,_REPO_NAME=$REPO_NAME,_SERVICE_NAME=clotilde,_OPENAI_SECRET=$OPENAI_SECRET,_API_SECRET=$API_SECRET
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

#### Admin Dashboard (Optional)

To enable the admin dashboard for monitoring logs and statistics:
- `ADMIN_USER`: Admin username for Basic Auth
- `ADMIN_PASSWORD`: Admin password for Basic Auth (use a strong password)
- `LOG_BUFFER_SIZE`: Maximum log entries to keep in memory (default: 1000)

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

### Perplexity Search API Integration

Clotilde supports Perplexity AI Search API as an alternative to OpenAI's native web_search tool. When enabled, Perplexity provides web search results that are formatted and included in the system prompt for the OpenAI model.

#### Features

- **Toggle Control**: Enable/disable Perplexity via admin dashboard or API (enabled by default)
- **Automatic Fallback**: Falls back to OpenAI web_search if Perplexity fails
- **Language Filtering**: Automatically filters results by language (Portuguese for Brazilian queries)
- **Result Formatting**: Search results are formatted and explained in the system prompt

#### Setup

1. **Create Perplexity API Key Secret**:
   ```bash
   export PERPLEXITY_SECRET="my-perplexity-key-$(openssl rand -hex 4)"
   echo -n "pplx-YOUR_API_KEY_HERE" | gcloud secrets create $PERPLEXITY_SECRET \
       --data-file=- \
       --replication-policy="automatic"
   ```

2. **Grant Access**:
   ```bash
   gcloud secrets add-iam-policy-binding $PERPLEXITY_SECRET \
       --member="serviceAccount:${SERVICE_ACCOUNT}" \
       --role="roles/secretmanager.secretAccessor"
   ```

3. **Set Environment Variable** (in `deploy.sh` or Cloud Run):
   ```bash
   export PERPLEXITY_SECRET_NAME=$PERPLEXITY_SECRET
   ```

4. **Configure via Admin Dashboard**:
   - Navigate to `/admin/` in your browser
   - Toggle "Enable Perplexity Search API for Web Search" on/off
   - Changes take effect immediately

#### How It Works

When Perplexity is enabled and a web search query is detected:
1. Perplexity Search API is called with the user's query
2. Results are formatted with titles, URLs, and snippets
3. Formatted results are appended to the system prompt with explanation
4. OpenAI model uses these results to generate the response
5. If Perplexity fails, automatically falls back to OpenAI's web_search tool

### Configuration API

The `/api/config` endpoint allows you to read and update system prompts and model configuration programmatically using your API key (same authentication as `/chat`). This is an alternative to the admin dashboard for programmatic access.

#### Get Current Configuration

```
GET /api/config
```

**Headers:**
```
X-API-Key: your-api-key
```

**Response:**
```json
{
  "base_system_prompt": "Você é \"Clotilde\"...",
  "category_prompts": {
    "web_search": "...",
    "complex": "...",
    "factual": "...",
    "mathematical": "...",
    "creative": "..."
  },
  "standard_model": "gpt-4.1-mini",
  "premium_model": "gpt-4.1-mini",
  "perplexity_enabled": true,
  "category_models": {}
}
```

#### Update Configuration

```
POST /api/config
```

**Headers:**
```
Content-Type: application/json
X-API-Key: your-api-key
```

**Request Body:**
```json
{
  "base_system_prompt": "Você é \"Clotilde\"...",
  "category_prompts": {
    "web_search": "Custom prompt for web search...",
    "complex": "Custom prompt for complex queries..."
  },
  "standard_model": "gpt-4o-mini",
  "premium_model": "gpt-4.1",
  "perplexity_enabled": true,
  "category_models": {
    "web_search": "gpt-4o"
  }
}
```

**Response (Success):**
```json
{
  "base_system_prompt": "...",
  "category_prompts": {...},
  "standard_model": "gpt-4o-mini",
  "premium_model": "gpt-4.1",
  "perplexity_enabled": true,
  "category_models": {...}
}
```

**Example: Toggle Perplexity Search API**

To enable Perplexity:
```bash
curl -X POST https://your-service-url.run.app/api/config \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "perplexity_enabled": true
  }'
```

To disable Perplexity (use OpenAI web_search instead):
```bash
curl -X POST https://your-service-url.run.app/api/config \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "perplexity_enabled": false
  }'
```

Note: You can update just the `perplexity_enabled` field without changing other settings. The API will merge your changes with the existing configuration.

**Response (Error):**
```json
{
  "error": "Error message"
}
```

**Validation:**
- Base system prompt must contain exactly one `%s` placeholder for date/time
- Maximum prompt size: 10KB per prompt
- Maximum request body size: 50KB
- Models must be valid OpenAI Responses API models
- Changes take effect immediately for all new requests

**Status Codes:**
- `200 OK`: Successful GET or POST
- `400 Bad Request`: Invalid JSON or validation errors
- `401 Unauthorized`: Missing or invalid API key
- `413 Request Entity Too Large`: Config body exceeds size limit
- `405 Method Not Allowed`: Unsupported HTTP method

## Admin Dashboard

The admin dashboard provides a web-based interface for monitoring your Clotilde instance.

### Features

- **Request Logs**: View recent requests with filtering by model, status, and date range
- **Usage Statistics**: Total requests, average response time, error rate, model usage distribution
- **Real-time Updates**: Auto-refresh every 10 seconds (configurable)
- **Request Tracing**: Each request gets a unique ID (`X-Request-ID` header) for debugging

### Setup

1. Set environment variables:
   ```bash
   export ADMIN_USER=your-username
   export ADMIN_PASSWORD=your-strong-password
   ```

2. Access the dashboard at: `https://your-service-url.run.app/admin/`

3. Log in with HTTP Basic Auth using your credentials

### API Endpoints

| Endpoint | Description | Authentication |
|----------|-------------|---------------|
| `POST /chat` | Chat endpoint for AI responses | X-API-Key |
| `GET /api/config` | Get current runtime configuration (system prompt, models) | X-API-Key |
| `POST /api/config` | Update runtime configuration without redeployment | X-API-Key |
| `GET /admin/` | Dashboard HTML page | HTTP Basic Auth |
| `GET /admin/logs` | JSON API for log entries (supports pagination and filtering) | HTTP Basic Auth |
| `GET /admin/stats` | JSON API for aggregated statistics | HTTP Basic Auth |
| `GET /admin/config` | Get current runtime configuration (system prompt, models) | HTTP Basic Auth |
| `POST /admin/config` | Update runtime configuration without redeployment | HTTP Basic Auth |
| `GET /health` | Enhanced health check with uptime, request count, and memory usage | None |

### Runtime Configuration

The admin dashboard allows you to change configuration without redeploying:

- **System Prompt**: Edit the AI's personality and behavior instructions
- **Standard Model**: Select the model for simple queries (e.g., `gpt-4.1-mini`, `gpt-4o-mini`)
- **Premium Model**: Select the model for complex queries (default: `gpt-4.1-mini`, also supports `gpt-4.1`, `o3`)

Changes take effect immediately for all new requests.

### Security

- Protected by HTTP Basic Auth (separate from API key authentication)
- Full user input and AI responses are logged for debugging and monitoring (stored in-memory buffer and Google Cloud Logging)
- Logs are protected by authentication (admin dashboard) and IAM (Cloud Logging)
- Admin credentials should be stored securely (use Secret Manager in production)
- See `docs/SECURITY.md` for detailed information about data retention and access controls

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
- **Admin Dashboard**: Protected by HTTP Basic Auth with separate credentials

**Before Sharing on GitHub**: All API keys have been replaced with placeholders (`YOUR_API_KEY`, `YOUR_SERVICE_URL`) in documentation files. Sensitive files are excluded via `.gitignore`.

See [docs/SECURITY.md](docs/SECURITY.md) for detailed security documentation.

## Resource Usage

- **Docker Image**: ~14.9MB (Alpine-based)
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

## Documentation

- [docs/QUICKSTART.md](docs/QUICKSTART.md) - Quick 5-minute setup guide
- [docs/SECURITY.md](docs/SECURITY.md) - Security documentation and best practices
- [docs/LOCAL_DOCKER.md](docs/LOCAL_DOCKER.md) - Local Docker development guide
- [docs/SHORTCUT_SETUP.md](docs/SHORTCUT_SETUP.md) - Apple Shortcut setup guide (English)
- [docs/GUIA_SHORTCUT_IPHONE.md](docs/GUIA_SHORTCUT_IPHONE.md) - Guia de configuração do Shortcut (Português)
- [docs/agents.md](docs/agents.md) - AI agent documentation for code maintainers (critical code paths, common issues)

## License

MIT License - See [LICENSE](LICENSE) file for details

