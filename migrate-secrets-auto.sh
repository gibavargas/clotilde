#!/bin/bash
# Non-interactive migration script to create new secrets with unique names
# This script automatically copies values from old secrets

set -e

# Configuration
PROJECT_ID=${GOOGLE_CLOUD_PROJECT:-$(gcloud config get-value project)}
REGION=${REGION:-us-central1}
SERVICE_NAME=${SERVICE_NAME:-clotilde}

# Generate unique suffixes for secret names
SUFFIX=$(openssl rand -hex 4)
OPENAI_SECRET_NEW="clotilde-oai-${SUFFIX}"
API_SECRET_NEW="clotilde-auth-${SUFFIX}"

echo "=========================================="
echo "Secret Migration Script (Auto)"
echo "=========================================="
echo "Project ID: $PROJECT_ID"
echo "Region: $REGION"
echo ""
echo "New secret names:"
echo "  OpenAI: $OPENAI_SECRET_NEW"
echo "  API Key: $API_SECRET_NEW"
echo ""

# Old secret names
OLD_OPENAI="openai-api-key"
OLD_API="clotilde-api-key"

# Create new OpenAI secret
echo "Creating OpenAI secret: $OPENAI_SECRET_NEW"
if gcloud secrets describe $OLD_OPENAI --project=$PROJECT_ID &>/dev/null; then
    echo "  Copying value from old secret: $OLD_OPENAI"
    gcloud secrets versions access latest --secret=$OLD_OPENAI --project=$PROJECT_ID | \
        gcloud secrets create $OPENAI_SECRET_NEW \
            --data-file=- \
            --replication-policy="automatic" \
            --project=$PROJECT_ID \
            --quiet
    echo "  ✓ Created from old secret"
else
    echo "  ✗ ERROR: Old secret $OLD_OPENAI not found"
    exit 1
fi

# Create new API key secret
echo ""
echo "Creating API key secret: $API_SECRET_NEW"
if gcloud secrets describe $OLD_API --project=$PROJECT_ID &>/dev/null; then
    echo "  Copying value from old secret: $OLD_API"
    gcloud secrets versions access latest --secret=$OLD_API --project=$PROJECT_ID | \
        gcloud secrets create $API_SECRET_NEW \
            --data-file=- \
            --replication-policy="automatic" \
            --project=$PROJECT_ID \
            --quiet
    echo "  ✓ Created from old secret"
else
    echo "  ✗ ERROR: Old secret $OLD_API not found"
    exit 1
fi

# Grant IAM permissions
echo ""
echo "Granting IAM permissions..."
SERVICE_ACCOUNT=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")-compute@developer.gserviceaccount.com

gcloud secrets add-iam-policy-binding $OPENAI_SECRET_NEW \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor" \
    --project=$PROJECT_ID \
    --quiet
echo "  ✓ Granted access to $OPENAI_SECRET_NEW"

gcloud secrets add-iam-policy-binding $API_SECRET_NEW \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor" \
    --project=$PROJECT_ID \
    --quiet
echo "  ✓ Granted access to $API_SECRET_NEW"

echo ""
echo "=========================================="
echo "Migration Complete!"
echo "=========================================="
echo ""
echo "New secret names:"
echo "  OPENAI_SECRET=$OPENAI_SECRET_NEW"
echo "  API_SECRET=$API_SECRET_NEW"
echo ""
echo "Export these for deployment:"
echo "  export OPENAI_SECRET=$OPENAI_SECRET_NEW"
echo "  export API_SECRET=$API_SECRET_NEW"
echo ""

