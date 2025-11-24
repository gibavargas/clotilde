#!/bin/bash
# Deploy Clotilde to Cloud Run
# Prerequisites: setup-gcloud.sh has been run

set -e

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
    --set-secrets OPENAI_KEY_SECRET_NAME=YOUR_OPENAI_SECRET_NAME:latest,API_KEY_SECRET_NAME=YOUR_API_SECRET_NAME:latest \
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

