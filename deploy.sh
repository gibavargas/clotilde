#!/bin/bash
# Deploy Clotilde (Hermes relay) to Cloud Run.
#
# Required environment variables:
#   API_SECRET    - Service API key secret name (X-API-Key)
#   HERMES_SECRET - Hermes gateway bearer key secret name
#   HERMES_URL    - Gateway HTTPS endpoint (set as HERMES_CHAT_URL env var)
#
# Optional:
#   MIN_INSTANCES - Warm instances (default 0; 1 kills cold starts, costs more)
#   METRICS_SECRET- Bearer token secret for /metrics
set -e

if [ -z "$API_SECRET" ] || [ -z "$HERMES_SECRET" ] || [ -z "$HERMES_URL" ]; then
    echo "ERROR: set API_SECRET, HERMES_SECRET (secret names) and HERMES_URL (gateway endpoint)."
    exit 1
fi

PROJECT_ID=${GOOGLE_CLOUD_PROJECT:-$(gcloud config get-value project)}
REGION=${REGION:-us-central1}
REPO_NAME=${REPO_NAME:-clotilde-repo}
SERVICE_NAME=${SERVICE_NAME:-clotilde}
IMAGE_TAG=${IMAGE_TAG:-latest}
MIN_INSTANCES=${MIN_INSTANCES:-0}
LOG_BUFFER_SIZE=${LOG_BUFFER_SIZE:-1000}
LOG_FULL_CONTENT=${LOG_FULL_CONTENT:-false}
LOG_REDACT_PII=${LOG_REDACT_PII:-true}

IMAGE_NAME="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO_NAME}/${SERVICE_NAME}:${IMAGE_TAG}"

# IP_HASH_SALT lives in Secret Manager (not a plaintext env var): anyone with
# run.services.describe could otherwise read it. Create once if missing.
if ! gcloud secrets describe clotilde-ip-hash-salt --project=$PROJECT_ID >/dev/null 2>&1; then
    echo "Creating Secret Manager secret clotilde-ip-hash-salt..."
    openssl rand -hex 32 | gcloud secrets create clotilde-ip-hash-salt \
        --project=$PROJECT_ID --data-file=- --replication-policy=automatic
fi

ENV_VARS="GOOGLE_CLOUD_PROJECT=${PROJECT_ID},LOG_BUFFER_SIZE=${LOG_BUFFER_SIZE},LOG_FULL_CONTENT=${LOG_FULL_CONTENT},LOG_REDACT_PII=${LOG_REDACT_PII},HERMES_CHAT_URL=${HERMES_URL}"
SECRETS="API_KEY_SECRET_NAME=${API_SECRET}:latest,IP_HASH_SALT=clotilde-ip-hash-salt:latest,HERMES_KEY_SECRET_NAME=${HERMES_SECRET}:latest"
if [ -n "$METRICS_SECRET" ]; then
    SECRETS="${SECRETS},METRICS_TOKEN=${METRICS_SECRET}:latest"
fi

echo "Deploying: project=$PROJECT_ID region=$REGION image=$IMAGE_NAME"
echo "Secrets: API=${API_SECRET} HERMES=${HERMES_SECRET}"

# --provenance/--sbom off: buildx's OCI index with attestations is rejected
# by Cloud Run ("must support amd64/linux").
echo "Building linux/amd64 image..."
if docker buildx version >/dev/null 2>&1; then
    docker buildx build --provenance=false --sbom=false -t $IMAGE_NAME --push .
else
    docker build -t $IMAGE_NAME . && docker push $IMAGE_NAME
fi

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

SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --region $REGION --format="value(status.url)")
echo ""
echo "Deploy OK: $SERVICE_URL"
echo "Verify: curl -I ${SERVICE_URL}/health  (expect 200)"
echo "         curl -I ${SERVICE_URL}/chat   (expect 401 without X-API-Key)"
