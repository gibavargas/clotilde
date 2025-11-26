#!/bin/bash
# Migration script to create new secrets with unique names
# This script helps migrate from hardcoded secret names to configurable ones
# Run this script to create new secrets and update your deployment

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
echo "Secret Migration Script"
echo "=========================================="
echo "Project ID: $PROJECT_ID"
echo "Region: $REGION"
echo ""
echo "This script will:"
echo "1. Create new secrets with unique names"
echo "2. Copy values from old secrets (if they exist)"
echo "3. Grant IAM permissions"
echo "4. Show you how to deploy with new secrets"
echo ""
echo "New secret names:"
echo "  OpenAI: $OPENAI_SECRET_NEW"
echo "  API Key: $API_SECRET_NEW"
echo ""
read -p "Press Enter to continue or Ctrl+C to cancel..."

# Check if old secrets exist
OLD_OPENAI="openai-api-key"
OLD_API="clotilde-api-key"

# Create new OpenAI secret
echo ""
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
    echo "  Old secret not found. You'll need to provide the value manually."
    echo -n "  Enter your OpenAI API key: "
    read -s OPENAI_KEY
    echo ""
    echo -n "$OPENAI_KEY" | gcloud secrets create $OPENAI_SECRET_NEW \
        --data-file=- \
        --replication-policy="automatic" \
        --project=$PROJECT_ID \
        --quiet
    echo "  ✓ Created with new value"
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
    echo "  Old secret not found. You'll need to provide the value manually."
    echo -n "  Enter your API key: "
    read -s API_KEY
    echo ""
    echo -n "$API_KEY" | gcloud secrets create $API_SECRET_NEW \
        --data-file=- \
        --replication-policy="automatic" \
        --project=$PROJECT_ID \
            --quiet
    echo "  ✓ Created with new value"
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
echo "Next steps:"
echo ""
echo "1. Deploy with new secrets using Cloud Build:"
echo "   gcloud builds submit --config=cloudbuild.yaml \\"
echo "     --substitutions=_OPENAI_SECRET=$OPENAI_SECRET_NEW,_API_SECRET=$API_SECRET_NEW"
echo ""
echo "2. Or use deploy.sh:"
echo "   export OPENAI_SECRET=$OPENAI_SECRET_NEW"
echo "   export API_SECRET=$API_SECRET_NEW"
echo "   ./deploy.sh"
echo ""
echo "3. After verifying deployment works, delete old secrets:"
echo "   gcloud secrets delete $OLD_OPENAI --project=$PROJECT_ID"
echo "   gcloud secrets delete $OLD_API --project=$PROJECT_ID"
echo ""
echo "4. Save these secret names for future deployments:"
echo "   export OPENAI_SECRET=$OPENAI_SECRET_NEW"
echo "   export API_SECRET=$API_SECRET_NEW"
echo ""

