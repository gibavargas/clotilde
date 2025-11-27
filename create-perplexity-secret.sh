#!/bin/bash
# Script to create Perplexity API key secret in Google Secret Manager
# Usage: ./create-perplexity-secret.sh [secret-name]

set -e

# Get project ID
PROJECT_ID=${GOOGLE_CLOUD_PROJECT:-$(gcloud config get-value project 2>/dev/null)}
if [ -z "$PROJECT_ID" ]; then
    echo "Error: GOOGLE_CLOUD_PROJECT not set and unable to get from gcloud config"
    exit 1
fi

# Generate secret name or use provided one
if [ -z "$1" ]; then
    PERPLEXITY_SECRET="my-perplexity-key-$(openssl rand -hex 4)"
else
    PERPLEXITY_SECRET="$1"
fi

# Perplexity API key (get from environment variable or prompt user)
if [ -z "$PERPLEXITY_API_KEY" ]; then
    echo -n "Enter your Perplexity API key: "
    read -s PERPLEXITY_API_KEY
    echo ""
    if [ -z "$PERPLEXITY_API_KEY" ]; then
        echo "Error: Perplexity API key is required"
        exit 1
    fi
fi

echo "Creating Perplexity API key secret: $PERPLEXITY_SECRET"
echo "Project: $PROJECT_ID"

# Create secret
if ! gcloud secrets describe $PERPLEXITY_SECRET --project=$PROJECT_ID &>/dev/null; then
    echo -n "$PERPLEXITY_API_KEY" | gcloud secrets create $PERPLEXITY_SECRET \
        --project=$PROJECT_ID \
        --data-file=- \
        --replication-policy="automatic" \
        --quiet
    echo "✓ Perplexity API key secret created: $PERPLEXITY_SECRET"
else
    echo "⚠ Secret $PERPLEXITY_SECRET already exists. Updating..."
    echo -n "$PERPLEXITY_API_KEY" | gcloud secrets versions add $PERPLEXITY_SECRET \
        --project=$PROJECT_ID \
        --data-file=- \
        --quiet
    echo "✓ Perplexity API key secret updated: $PERPLEXITY_SECRET"
fi

# Get Cloud Run service account
SERVICE_ACCOUNT=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")-compute@developer.gserviceaccount.com

# Grant Secret Manager access
echo "Granting IAM permissions..."
gcloud secrets add-iam-policy-binding $PERPLEXITY_SECRET \
    --project=$PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor" \
    --quiet

echo "✓ IAM permissions configured"
echo ""
echo "Next steps:"
echo "1. Set environment variable for deployment:"
echo "   export PERPLEXITY_SECRET_NAME=$PERPLEXITY_SECRET"
echo ""
echo "2. Or add to your deploy.sh script:"
echo "   PERPLEXITY_SECRET_NAME=$PERPLEXITY_SECRET"
echo ""
echo "3. Deploy your service with the PERPLEXITY_SECRET_NAME environment variable set"

