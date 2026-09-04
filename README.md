# Clotilde CarPlay Assistant

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A voice-activated CarPlay assistant centered on Claude Haiku for low-latency CarPlay answers, with optional OpenRouter and OpenAI Responses fallbacks, web search support, and a low-footprint Go backend. Built for Google Cloud Run and Apple Shortcuts.

## Features

- 🚗 CarPlay integration via Apple Shortcuts
- ⚡ Claude Haiku 4.5 direct API as the recommended primary model path
- 🧭 OpenRouter fallback for OpenAI-compatible access to Claude Haiku and other hosted models
- 🧠 Optional OpenAI Responses API fallback with native `web_search`
- 🌐 Optional Z.AI GLM fallback through OpenAI-compatible chat completions
- 🔍 Provider-native web search plus optional Camoufox rendering fallback
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
- One upstream model key: Anthropic Claude API key, OpenRouter API key, OpenAI API key, or Z.AI API key
- Apple iPhone with Shortcuts app

## Provider Strategy

Clotilde is optimized for fast spoken answers. The recommended setup is:

1. **Primary**: Claude Haiku 4.5 via Anthropic API (`CLAUDE_KEY_SECRET_NAME`)
2. **Fallback / aggregator**: OpenRouter (`OPENROUTER_KEY_SECRET_NAME`) using `openrouter/anthropic/claude-haiku-4.5`
3. **Search fallback**: OpenAI Responses API (`OPENAI_KEY_SECRET_NAME`) when you want native OpenAI `web_search`
4. **GLM fallback**: Z.AI (`ZAI_KEY_SECRET_NAME`) using `zai/glm-4.7-flash`, `zai/glm-4.7`, or `zai/glm-5.1`

OpenAI OAuth is not used for server-to-server calls to the OpenAI API. The backend sends `Authorization: Bearer ...`; for OpenAI’s API this should be a server-side API key. If you expose Clotilde through a GPT Action or another user-facing OpenAI OAuth flow, keep that OAuth layer in front of this service and continue protecting `/chat` with Clotilde’s own `X-API-Key`.

See [`docs/PROVIDERS.md`](docs/PROVIDERS.md) for the full provider matrix.

## Setup

### 1. Guided Setup Wizard

The recommended setup path is the Go-based wizard. It checks local prerequisites, creates or updates Google Cloud resources, deploys to Cloud Run, verifies the service, and writes a sanitized summary to `.clotilde/setup-result.json`.

```bash
go run ./cmd/clotilde-setup
```

For OpenClaw, Hermes, and other agent automation, generate a starter template and run non-interactive JSON mode:

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

See [OpenClaw and Hermes setup](docs/OPENCLAW_HERMES_SETUP.md) for the profile-specific flow.

Dry-run mode prints the exact planned command sequence without changing Google Cloud resources:

```bash
go run ./cmd/clotilde-setup \
  --non-interactive \
  --config cmd/clotilde-setup/testdata/minimal.json \
  --output json \
  --yes \
  --dry-run
```

Minimal `setup.json` shape:

```json
{
  "project_id": "your-project-id",
  "region": "us-central1",
  "service_name": "clotilde",
  "repo_name": "clotilde-repo",
  "api": {
    "secret_name": "clotilde-auth-example",
    "generate": true
  },
  "claude": {
    "enabled": true,
    "secret": {
      "secret_name": "clotilde-claude-example",
      "value_env": "ANTHROPIC_API_KEY"
    }
  },
  "admin": {
    "enabled": true,
    "username": "admin",
    "password": {
      "secret_name": "clotilde-admin-example",
      "value_env": "CLOTILDE_ADMIN_PASSWORD"
    }
  }
}
```

The wizard accepts optional `openai`, `openrouter`, `z_ai`, and `config_api` sections. At least one model provider (`claude`, `openrouter`, `openai`, or `z_ai`) must be configured.

The runtime `/api/config` endpoint is protected by the same service `X-API-Key` used for `/chat`. The `config_api` section is currently only a generated handoff secret for agent workflows that need it in `.clotilde/setup-result.json`.

Secret values are never printed. If the wizard generates the service API key, retrieve it for the Apple Shortcut with:

```bash
gcloud secrets versions access latest --secret=<your-api-secret-name>
```

### 2. Manual / Advanced Deployment

The older scripts remain supported if you prefer to provision and deploy manually.

```bash
# Set required environment variables
export API_SECRET=your-api-secret-name
export CLAUDE_SECRET=your-claude-secret-name

# Optional provider fallbacks
export OPENROUTER_SECRET=your-openrouter-secret-name
export OPENAI_SECRET=your-openai-secret-name

# Optional: Enable admin dashboard
export ADMIN_USER=admin
export ADMIN_SECRET=your-admin-password-secret-name
export LOG_BUFFER_SIZE=1000

# Deploy
chmod +x deploy.sh
./deploy.sh
```

`API_SECRET` is the service authentication key secret name. `CLAUDE_SECRET`, `OPENROUTER_SECRET`, `OPENAI_SECRET`, and `ZAI_SECRET` are upstream provider secret names; set at least one. Inside Cloud Run, they are mounted as `API_KEY_SECRET_NAME`, `CLAUDE_KEY_SECRET_NAME`, `OPENROUTER_KEY_SECRET_NAME`, `OPENAI_KEY_SECRET_NAME`, and `ZAI_KEY_SECRET_NAME`.

#### Cloud Build (Deprecated)

> **Note**: Cloud Build is deprecated in favor of the local `deploy.sh` script. This option is kept for reference but is not recommended for new deployments.

```bash
# Submit build (use your secret names from the setup step)
gcloud builds submit --config=cloudbuild.yaml \
    --substitutions=_REGION=$REGION,_REPO_NAME=$REPO_NAME,_SERVICE_NAME=clotilde,_CLAUDE_SECRET=$CLAUDE_SECRET,_API_SECRET=$API_SECRET
```

### 3. Get Your Service URL

```bash
gcloud run services describe clotilde --region $REGION --format="value(status.url)"
```

Save this URL - you'll need it for the Apple Shortcut.

**🎯 Pro Tip**: After deployment, you can change models and settings **WITHOUT redeploying**! See the [Runtime Configuration](#runtime-configuration-no-redeployment-needed) section below for details.

### 4. Environment Variables

For local development, create a `.env` file from the template:

```bash
cp .env.example .env
# Edit .env with your actual values
```

**Important**: Never commit `.env` files to git. They are already in `.gitignore`.

The `.env` file should contain:
- `API_KEY_SECRET_NAME`: Your Clotilde service API key (get from Secret Manager)
- `CLAUDE_KEY_SECRET_NAME`: Your Anthropic API key for Claude Haiku direct API
- `OPENROUTER_KEY_SECRET_NAME`: Optional OpenRouter API key
- `OPENAI_KEY_SECRET_NAME`: Optional OpenAI API key for Responses fallback
- `ZAI_KEY_SECRET_NAME`: Optional Z.AI API key for GLM chat-completions fallback
- `GOOGLE_CLOUD_PROJECT`: Your Google Cloud project ID
- `SERVICE_URL`: Your deployed service URL (optional, for testing)

#### Admin Dashboard (Optional)

To enable the admin dashboard for monitoring logs and statistics:
- `ADMIN_USER`: Admin username for Basic Auth
- `ADMIN_PASSWORD`: Admin password for Basic Auth (use a strong password)
- `LOG_BUFFER_SIZE`: Maximum log entries to keep in memory (default: 1000)
- `LOG_FULL_CONTENT`: Set to `true` only if you intentionally want full user prompts and AI responses in logs. Default logging stores metadata only.

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
   export API_KEY_SECRET_NAME=your-api-key
   export CLAUDE_KEY_SECRET_NAME=your-claude-api-key
   export GOOGLE_CLOUD_PROJECT=your-project-id
   export PORT=8080
   go run cmd/clotilde/main.go
   ```

**Note**: 
- Local development can use environment variables directly or fall back to Secret Manager
- Production uses Secret Manager via Cloud Run secret mounting
- **Never commit `.env` files** - they're in `.gitignore`

## Apple Shortcut Setup

Shortcut files are not committed to this repository. Create the shortcut manually in the Shortcuts app, or follow the fuller guide in [docs/SHORTCUT_SETUP.md](docs/SHORTCUT_SETUP.md).

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

### Web Search Providers

Clotilde grounds time-sensitive and factual answers through the selected provider's native web-search tool whenever possible. The search-capable paths are:

1. Claude direct API uses Claude's native `web_search_20250305` tool when configured.
2. OpenAI Responses uses native `web_search` when an OpenAI key is configured.
3. Z.AI chat completions uses the `web_search` tool when a Z.AI key is configured.
4. OpenRouter uses the `openrouter:web_search` server tool when an OpenRouter key is configured.
5. An optional Camoufox browser harness can render public search result pages when native search fails or returns blocked/JavaScript-gated snippets.

#### Features

- **Provider-native search**: Uses each configured model API's own search tool rather than a separate search API key.
- **Factual validation**: Router marks factual and web-search categories with `web_search=true`, so changeable facts are validated before the answer is spoken.
- **Automatic fallback**: If the selected provider cannot search or native search fails, Clotilde tries another configured search-capable provider.
- **Optional browser rendering**: Uses a configured Camoufox runner when native search snippets suggest bot/JavaScript blocking, or always when explicitly configured.

#### How It Works

When a factual or web-search query is detected:
1. The router selects the fastest configured provider that can handle the category.
2. The provider receives its native web-search tool configuration.
3. If native search fails or looks blocked, Camoufox can fetch public search result pages and compact visible evidence into the prompt.
4. The selected model answers using the validated context within the active CarPlay request budget.

#### Optional Camoufox Browser Harness

For pages that return thin snippets such as "enable JavaScript", "access denied", or "please verify you are not a robot", Clotilde can call an external Camoufox runner to search and render public result pages. This is off by default because it adds latency and requires a browser runtime.

Install the optional runner dependencies in the environment that will execute the harness:

```bash
python3 -m venv .venv-camoufox
. .venv-camoufox/bin/activate
pip install -r harness/requirements-camoufox.txt
python -m camoufox fetch
```

Enable it with environment variables:

```bash
export CAMOUFOX_HARNESS_ENABLED=true
export CAMOUFOX_HARNESS_CMD=python3
export CAMOUFOX_HARNESS_ARGS=harness/camoufox_fetch.py
export CAMOUFOX_HARNESS_MODE=blocked   # blocked or always
export CAMOUFOX_HARNESS_MAX_PAGES=2
export CAMOUFOX_HARNESS_TIMEOUT_SECONDS=8
```

The runner only opens public URLs and extracts visible text. It does not solve CAPTCHAs, submit forms, use credentials, or click through access walls. Clotilde also caps the harness against the active CarPlay request budget so browser rendering cannot consume the whole answer window.

### Configuration API

The `/api/config` endpoint allows you to read and update system prompts and model configuration programmatically using your service API key, with the same `X-API-Key` authentication as `/chat`. This is an alternative to the admin dashboard for programmatic access.

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
  "standard_model": "claude-haiku-4-5-20251001",
  "premium_model": "claude-haiku-4-5-20251001",
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
  "standard_model": "openrouter/anthropic/claude-haiku-4.5",
  "premium_model": "claude-haiku-4-5-20251001",
  "category_models": {
    "web_search": "openrouter/anthropic/claude-haiku-4.5"
  }
}
```

**Response (Success):**
```json
{
  "base_system_prompt": "...",
  "category_prompts": {...},
  "standard_model": "openrouter/anthropic/claude-haiku-4.5",
  "premium_model": "claude-haiku-4-5-20251001",
  "category_models": {...}
}
```

**Example: Route Factual Search Through Z.AI**

```bash
curl -X POST https://your-service-url.run.app/api/config \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "category_models": {
      "factual": "zai/glm-4.7-flash",
      "web_search": "zai/glm-4.7-flash"
    }
  }'
```

**Example: Route Web Search Through OpenRouter**

```bash
curl -X POST https://your-service-url.run.app/api/config \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "category_models": {
      "web_search": "openrouter/anthropic/claude-haiku-4.5"
    }
  }'
```

Note: You can update just `category_models` without changing other settings. The API will merge your changes with the existing configuration.

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
- Models must be valid provider model IDs. OpenRouter models use the `openrouter/<provider>/<model>` prefix, for example `openrouter/anthropic/claude-haiku-4.5`.
- Z.AI models use the `zai/<model>` prefix, for example `zai/glm-4.7-flash` or `zai/glm-5.1`. For factual/web-search routes, Clotilde uses the native Z.AI web-search tool and can fall back to Camoufox evidence when browser rendering is enabled.
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
| `POST /api/config` | Update runtime configuration without redeployment | Admin Basic Auth ( privilege-separated from the /chat API key; fail-closed when admin unset) |
| `GET /admin/` | Dashboard HTML page | HTTP Basic Auth |
| `GET /admin/logs` | JSON API for log entries (supports pagination and filtering) | HTTP Basic Auth |
| `GET /admin/stats` | JSON API for aggregated statistics | HTTP Basic Auth |
| `GET /admin/config` | Get current runtime configuration (system prompt, models) | HTTP Basic Auth |
| `POST /admin/config` | Update runtime configuration without redeployment | HTTP Basic Auth |
| `GET /health` | Enhanced health check with uptime, request count, and memory usage | None |

### Runtime Configuration (No Redeployment Needed!)

⚡ **Important**: You can change models, prompts, and settings **WITHOUT redeploying** the server! Changes take effect immediately for all new requests.

#### Two Ways to Update Configuration:

**1. Via API (Programmatic/Programmer-Friendly)**
```bash
# Get current config
curl -H "X-API-Key: YOUR_API_KEY" https://your-service-url.run.app/api/config

# Update models (example: switch to faster models for better performance)
curl -X POST https://your-service-url.run.app/api/config \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "standard_model": "claude-haiku-4-5-20251001",
    "premium_model": "claude-haiku-4-5-20251001"
  }'
```

**2. Via Admin Dashboard (Web UI)**
- Access: `https://your-service-url.run.app/admin/`
- Login with HTTP Basic Auth (ADMIN_USER/ADMIN_PASSWORD)
- Update models, prompts, and settings through the web interface

#### What You Can Change Without Redeployment:

- **Standard Model**: Model for simple queries (e.g., `claude-haiku-4-5-20251001`, `openrouter/anthropic/claude-haiku-4.5`, `gpt-4o-mini`)
- **Premium Model**: Model for complex queries (e.g., `claude-haiku-4-5-20251001`, `openrouter/anthropic/claude-haiku-4.5`, `gpt-4.1`)
- **System Prompts**: AI personality and behavior instructions
- **Category Models**: Override models for specific query types (web search, creative, etc.)
- **Search Routing**: Choose which configured provider handles factual and web-search categories

#### Example: Fix Timeout Issues by Switching to Faster Models

If users experience timeouts, switch to faster models:

```bash
# Switch to fast Claude Haiku for CarPlay latency
curl -X POST https://your-service-url.run.app/api/config \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "standard_model": "claude-haiku-4-5-20251001",
    "premium_model": "claude-haiku-4-5-20251001"
  }'
```

**Changes take effect immediately** - no downtime, no redeployment required!

### Security

- Protected by HTTP Basic Auth (separate from API key authentication)
- Metadata-only request logs are enabled by default; full prompts/responses are logged only when `LOG_FULL_CONTENT=true`
- Logs are protected by authentication (admin dashboard) and IAM (Cloud Logging)
- Admin credentials should be stored securely (use Secret Manager in production)
- See `docs/SECURITY.md` for detailed information about data retention and access controls

## Security Features

- **API Key Authentication**: All requests require valid API key
- **Rate Limiting**: 10 requests/minute per API key, 100 requests/hour per IP
- **Input Validation**: Max 1000 characters per message, 5KB request size limit
- **Secrets Management**: All sensitive data in Google Secret Manager
- **Secure Logging**: Metadata-only request logs are the default; full prompts/responses are logged only when `LOG_FULL_CONTENT=true`
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
- **Cold starts**: deploy defaults keep `MIN_INSTANCES=0` so Cloud Run can scale to zero when idle. Startup CPU boost is enabled, and the server keeps cold-start work small; if a car/phone shortcut can run a lightweight pre-wake request, call `GET /health` shortly before `POST /chat`.
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
- [docs/PROVIDERS.md](docs/PROVIDERS.md) - Provider selection and model configuration
- [docs/OPENCLAW_HERMES_SETUP.md](docs/OPENCLAW_HERMES_SETUP.md) - Agent-oriented setup wizard profiles
- [docs/SECURITY.md](docs/SECURITY.md) - Security documentation and best practices
- [docs/LOCAL_DOCKER.md](docs/LOCAL_DOCKER.md) - Local Docker development guide
- [docs/SHORTCUT_SETUP.md](docs/SHORTCUT_SETUP.md) - Apple Shortcut setup guide (English)
- [docs/GUIA_SHORTCUT_IPHONE.md](docs/GUIA_SHORTCUT_IPHONE.md) - Guia de configuração do Shortcut (Português)
- [docs/agents.md](docs/agents.md) - AI agent documentation for code maintainers (critical code paths, common issues)

## License

MIT License - See [LICENSE](LICENSE) file for details
