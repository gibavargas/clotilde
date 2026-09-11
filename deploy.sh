#!/bin/bash
# Deploy Clotilde to Cloud Run
# Prerequisites: setup-gcloud.sh has been run
#
# Required environment variables:
#   API_SECRET    - Name of your service API key secret in Secret Manager
#   At least one AI provider secret:
#     CLAUDE_SECRET, OPENROUTER_SECRET, OPENAI_SECRET, or ZAI_SECRET
#
# Optional environment variables:
#   CLAUDE_SECRET        - Name of Claude (Anthropic) API key secret in Secret Manager (RECOMMENDED)
#   OPENROUTER_SECRET    - Name of OpenRouter API key secret in Secret Manager
#   OPENAI_SECRET        - Name of OpenAI API key secret in Secret Manager (Responses fallback)
#   ZAI_SECRET           - Name of Z.AI API key secret in Secret Manager (GLM fallback)
#   ADMIN_USER           - Admin username for dashboard (enables admin dashboard)
#   ADMIN_SECRET         - Name of admin password secret in Secret Manager (enables admin dashboard)
#   HERMES_API_KEY       - Name of the Hermes API-server key secret; enables the Hermes
#                          primary path (HERMES_ENABLED=true) with provider-chain fallback
#   HERMES_ENABLED       - "true" to route chats through Hermes first (default: off)
#   CAMOUFOX_HARNESS_* - Optional external browser harness configuration (disabled by default)
#   LOG_BUFFER_SIZE      - Max log entries in memory (default: 1000)
#   MIN_INSTANCES        - Minimum warm instances (default: 0 to scale to zero when idle)
#
# Example:
#   export CLAUDE_SECRET=my-claude-key-abc123
#   export API_SECRET=my-api-key-xyz789
#   export ADMIN_USER=admin
#   export ADMIN_SECRET=my-admin-password-secret
#   ./deploy.sh

set -e

# Check required environment variables
if [ -z "$API_SECRET" ] || { [ -z "$CLAUDE_SECRET" ] && [ -z "$OPENROUTER_SECRET" ] && [ -z "$OPENAI_SECRET" ] && [ -z "$ZAI_SECRET" ]; }; then
    echo "ERROR: Required environment variables not set."
    echo ""
    echo "Please set the following environment variables:"
    echo "  export API_SECRET=your-api-secret-name"
    echo "  export CLAUDE_SECRET=your-claude-secret-name"
    echo ""
    echo "Or use another AI provider:"
    echo "  export OPENROUTER_SECRET=your-openrouter-secret-name"
    echo "  export OPENAI_SECRET=your-openai-secret-name"
    echo "  export ZAI_SECRET=your-zai-secret-name"
    echo ""
    echo "Optional (for admin dashboard):"
    echo "  export ADMIN_USER=your-admin-username"
    echo "  export ADMIN_SECRET=your-admin-password-secret-name"
    echo "  export LOG_BUFFER_SIZE=1000"
    echo ""
    echo "These should be the names of your secrets in Secret Manager."
    exit 1
fi

# Configuration
PROJECT_ID=${GOOGLE_CLOUD_PROJECT:-$(gcloud config get-value project)}
REGION=${REGION:-us-central1}
REPO_NAME=${REPO_NAME:-clotilde-repo}
SERVICE_NAME=${SERVICE_NAME:-clotilde}
IMAGE_TAG=${IMAGE_TAG:-latest}

IMAGE_NAME="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO_NAME}/${SERVICE_NAME}:${IMAGE_TAG}"

# Set defaults
LOG_BUFFER_SIZE=${LOG_BUFFER_SIZE:-1000}
MIN_INSTANCES=${MIN_INSTANCES:-0}

# Generate random salt for IP hashing if not provided
IP_HASH_SALT=${IP_HASH_SALT:-$(openssl rand -hex 32)}

# Full-content logging (questions/answers visible in Cloud Logging + admin).
# Default OFF: docs/SECURITY.md promises metadata-only logging by default.
# Set LOG_FULL_CONTENT=true explicitly when you accept the privacy trade-off.
LOG_FULL_CONTENT=${LOG_FULL_CONTENT:-false}
# PII redaction in logs: default ON (defense in depth; no-op unless full content is on).
LOG_REDACT_PII=${LOG_REDACT_PII:-true}

# Build environment variables string
# Note: PORT is automatically set by Cloud Run, don't include it
ENV_VARS="GOOGLE_CLOUD_PROJECT=${PROJECT_ID},LOG_BUFFER_SIZE=${LOG_BUFFER_SIZE},IP_HASH_SALT=${IP_HASH_SALT},LOG_FULL_CONTENT=${LOG_FULL_CONTENT},LOG_REDACT_PII=${LOG_REDACT_PII}"
if [ -n "$ADMIN_USER" ]; then
    ENV_VARS="${ENV_VARS},ADMIN_USER=${ADMIN_USER}"
fi
if [ "${HERMES_ENABLED:-}" = "true" ] && [ -n "$HERMES_API_KEY" ]; then
    ENV_VARS="${ENV_VARS},HERMES_ENABLED=true"
fi
if [ -n "$CAMOUFOX_HARNESS_ENABLED" ]; then
    ENV_VARS="${ENV_VARS},CAMOUFOX_HARNESS_ENABLED=${CAMOUFOX_HARNESS_ENABLED}"
fi
if [ -n "$CAMOUFOX_HARNESS_CMD" ]; then
    ENV_VARS="${ENV_VARS},CAMOUFOX_HARNESS_CMD=${CAMOUFOX_HARNESS_CMD}"
fi
if [ -n "$CAMOUFOX_HARNESS_ARGS" ]; then
    ENV_VARS="${ENV_VARS},CAMOUFOX_HARNESS_ARGS=${CAMOUFOX_HARNESS_ARGS}"
fi
if [ -n "$CAMOUFOX_HARNESS_MODE" ]; then
    ENV_VARS="${ENV_VARS},CAMOUFOX_HARNESS_MODE=${CAMOUFOX_HARNESS_MODE}"
fi
if [ -n "$CAMOUFOX_HARNESS_MAX_PAGES" ]; then
    ENV_VARS="${ENV_VARS},CAMOUFOX_HARNESS_MAX_PAGES=${CAMOUFOX_HARNESS_MAX_PAGES}"
fi
if [ -n "$CAMOUFOX_HARNESS_TIMEOUT_SECONDS" ]; then
    ENV_VARS="${ENV_VARS},CAMOUFOX_HARNESS_TIMEOUT_SECONDS=${CAMOUFOX_HARNESS_TIMEOUT_SECONDS}"
fi

# Build secrets string
SECRETS="API_KEY_SECRET_NAME=${API_SECRET}:latest"
if [ -n "$OPENAI_SECRET" ]; then
    SECRETS="${SECRETS},OPENAI_KEY_SECRET_NAME=${OPENAI_SECRET}:latest"
fi
if [ -n "$ADMIN_SECRET" ]; then
    SECRETS="${SECRETS},ADMIN_PASSWORD=${ADMIN_SECRET}:latest"
fi
if [ -n "$CLAUDE_SECRET" ]; then
    SECRETS="${SECRETS},CLAUDE_KEY_SECRET_NAME=${CLAUDE_SECRET}:latest"
fi
if [ -n "$OPENROUTER_SECRET" ]; then
    SECRETS="${SECRETS},OPENROUTER_KEY_SECRET_NAME=${OPENROUTER_SECRET}:latest"
fi
if [ -n "$ZAI_SECRET" ]; then
    SECRETS="${SECRETS},ZAI_KEY_SECRET_NAME=${ZAI_SECRET}:latest"
fi
if [ -n "$CEREBRAS_SECRET" ]; then
    SECRETS="${SECRETS},CEREBRAS_KEY_SECRET_NAME=${CEREBRAS_SECRET}:latest"
fi
if [ -n "$HERMES_API_KEY" ]; then
    SECRETS="${SECRETS},HERMES_API_KEY=${HERMES_API_KEY}:latest"
fi

echo "Building and deploying Clotilde..."
echo "Project ID: $PROJECT_ID"
echo "Region: $REGION"
echo "Image: $IMAGE_NAME"
echo "API Secret: $API_SECRET"
if [ -n "$OPENAI_SECRET" ]; then
    echo "OpenAI Secret: $OPENAI_SECRET"
fi
if [ -n "$ADMIN_USER" ]; then
    echo "Admin User: $ADMIN_USER"
    echo "Admin Secret: $ADMIN_SECRET"
fi
if [ -n "$CLAUDE_SECRET" ]; then
    echo "Claude Secret: $CLAUDE_SECRET (recommended for fast CarPlay responses)"
fi
if [ -n "$OPENROUTER_SECRET" ]; then
    echo "OpenRouter Secret: $OPENROUTER_SECRET"
fi
if [ -n "$ZAI_SECRET" ]; then
    echo "Z.AI Secret: $ZAI_SECRET"
fi
echo "Log Buffer Size: $LOG_BUFFER_SIZE"
echo "Min Instances: $MIN_INSTANCES"
echo ""

# Build and push a single-platform linux/amd64 image. --provenance/--sbom off:
# buildx defaults wrap the image in an OCI index with attestation manifests,
# which Cloud Run rejects ("must support amd64/linux").
echo "Building and pushing Docker image (linux/amd64)..."
docker buildx build --platform linux/amd64 --provenance=false --sbom=false \
    -t $IMAGE_NAME --push .

# Deploy to Cloud Run
echo "Deploying to Cloud Run..."
gcloud run deploy $SERVICE_NAME \
    --image $IMAGE_NAME \
    --region $REGION \
    --platform managed \
    --allow-unauthenticated \
    --memory 256Mi \
    --cpu 1 \
    --cpu-boost \
    --min-instances $MIN_INSTANCES \
    --max-instances 10 \
    --timeout 30 \
    --set-env-vars "$ENV_VARS" \
    --set-secrets "$SECRETS" \
    --quiet

# Get service URL
SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --region $REGION --format="value(status.url)")

echo ""
echo "✓ Deployment complete!"
echo ""
echo "Service URL: $SERVICE_URL"
echo ""
echo "Test the service:"
echo "curl -X POST $SERVICE_URL/chat \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-API-Key: YOUR_API_KEY' \\"
echo "  -d '{\"message\":\"Hello\"}'"
