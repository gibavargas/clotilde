#!/bin/bash
# Deploy Clotilde to Cloud Run
# Prerequisites: setup-gcloud.sh has been run
#
# Required environment variables:
#   OPENAI_SECRET - Name of your OpenAI API key secret in Secret Manager
#   API_SECRET    - Name of your service API key secret in Secret Manager
#
# Example:
#   export OPENAI_SECRET=my-openai-key-abc123
#   export API_SECRET=my-api-key-xyz789
#   ./deploy.sh

set -e

# Check required environment variables
if [ -z "$OPENAI_SECRET" ] || [ -z "$API_SECRET" ]; then
    echo "ERROR: Required environment variables not set."
    echo ""
    echo "Please set the following environment variables:"
    echo "  export OPENAI_SECRET=your-openai-secret-name"
    echo "  export API_SECRET=your-api-secret-name"
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

echo "Building and deploying Clotilde..."
echo "Project ID: $PROJECT_ID"
echo "Region: $REGION"
echo "Image: $IMAGE_NAME"
echo "OpenAI Secret: $OPENAI_SECRET"
echo "API Secret: $API_SECRET"
echo ""

# Build Docker image
echo "Building Docker image..."
docker build -t $IMAGE_NAME .

# Push to Artifact Registry
echo "Pushing to Artifact Registry..."
docker push $IMAGE_NAME

# Deploy to Cloud Run
echo "Deploying to Cloud Run..."
gcloud run deploy $SERVICE_NAME \
    --image $IMAGE_NAME \
    --region $REGION \
    --platform managed \
    --allow-unauthenticated \
    --memory 256Mi \
    --cpu 1 \
    --min-instances 0 \
    --max-instances 10 \
    --timeout 30 \
    --set-env-vars GOOGLE_CLOUD_PROJECT=$PROJECT_ID,PORT=8080 \
    --set-secrets OPENAI_KEY_SECRET_NAME=${OPENAI_SECRET}:latest,API_KEY_SECRET_NAME=${API_SECRET}:latest \
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

