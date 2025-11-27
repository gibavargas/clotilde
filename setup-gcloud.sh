#!/bin/bash
# Google Cloud Setup Script for Clotilde
# Prerequisites: gcloud CLI installed and authenticated
#
# Required environment variables:
#   OPENAI_SECRET - Name for your OpenAI API key secret (will be created)
#   API_SECRET    - Name for your service API key secret (will be created)
#
# Example:
#   export OPENAI_SECRET=my-openai-key-abc123
#   export API_SECRET=my-api-key-xyz789
#   ./setup-gcloud.sh

set -e

# Check required environment variables
if [ -z "$OPENAI_SECRET" ] || [ -z "$API_SECRET" ]; then
    echo "ERROR: Required environment variables not set."
    echo ""
    echo "Please set the following environment variables:"
    echo "  export OPENAI_SECRET=your-unique-openai-secret-name"
    echo "  export API_SECRET=your-unique-api-secret-name"
    echo ""
    echo "TIP: Use unique, unpredictable names for security."
    echo "     Example: openssl rand -hex 4 | xargs -I{} echo my-openai-key-{}"
    exit 1
fi

# Configuration
PROJECT_ID=${GOOGLE_CLOUD_PROJECT:-$(gcloud config get-value project)}
REGION=${REGION:-us-central1}
REPO_NAME=${REPO_NAME:-clotilde-repo}
SERVICE_NAME=${SERVICE_NAME:-clotilde}

echo "Setting up Clotilde on Google Cloud..."
echo "Project ID: $PROJECT_ID"
echo "Region: $REGION"
echo "OpenAI Secret Name: $OPENAI_SECRET"
echo "API Secret Name: $API_SECRET"
echo ""

# Set project
gcloud config set project $PROJECT_ID

# Enable required APIs
echo "Enabling required APIs..."
gcloud services enable \
    artifactregistry.googleapis.com \
    secretmanager.googleapis.com \
    run.googleapis.com

# Create Artifact Registry repository
echo "Creating Artifact Registry repository..."
if ! gcloud artifacts repositories describe $REPO_NAME --location=$REGION &>/dev/null; then
    gcloud artifacts repositories create $REPO_NAME \
        --repository-format=docker \
        --location=$REGION \
        --description="Clotilde Docker images" \
        --quiet
    echo "✓ Artifact Registry repository created"
else
    echo "✓ Artifact Registry repository already exists"
fi

# Create secrets (interactive)
echo ""
echo "Creating secrets in Secret Manager..."
echo "You will be prompted to enter values."

# OpenAI API Key
if ! gcloud secrets describe $OPENAI_SECRET &>/dev/null; then
    echo -n "Enter your OpenAI API key: "
    read -s OPENAI_KEY
    echo ""
    echo -n "$OPENAI_KEY" | gcloud secrets create $OPENAI_SECRET \
        --data-file=- \
        --replication-policy="automatic" \
        --quiet
    echo "✓ OpenAI API key secret created ($OPENAI_SECRET)"
else
    echo "✓ OpenAI API key secret already exists ($OPENAI_SECRET)"
fi

# Clotilde API Key
if ! gcloud secrets describe $API_SECRET &>/dev/null; then
    echo -n "Enter your Clotilde API key (for authenticating requests): "
    read -s CLOTILDE_KEY
    echo ""
    echo -n "$CLOTILDE_KEY" | gcloud secrets create $API_SECRET \
        --data-file=- \
        --replication-policy="automatic" \
        --quiet
    echo "✓ Clotilde API key secret created ($API_SECRET)"
else
    echo "✓ Clotilde API key secret already exists ($API_SECRET)"
fi

# Perplexity API Key (optional)
export PERPLEXITY_SECRET="my-perplexity-key-$(openssl rand -hex 4)"
if ! gcloud secrets describe $PERPLEXITY_SECRET &>/dev/null; then
    echo ""
    echo "Perplexity API Key (optional - for web search):"
    echo -n "Enter your Perplexity API key (or press Enter to skip): "
    read -s PERPLEXITY_KEY
    echo ""
    if [ ! -z "$PERPLEXITY_KEY" ]; then
        echo -n "$PERPLEXITY_KEY" | gcloud secrets create $PERPLEXITY_SECRET \
            --data-file=- \
            --replication-policy="automatic" \
            --quiet
        echo "✓ Perplexity API key secret created ($PERPLEXITY_SECRET)"
        echo "PERPLEXITY_SECRET=$PERPLEXITY_SECRET"
    else
        echo "✓ Perplexity API key skipped (Perplexity Search API will be disabled)"
        PERPLEXITY_SECRET=""
    fi
else
    echo "✓ Perplexity API key secret already exists ($PERPLEXITY_SECRET)"
fi

# Get Cloud Run service account
echo ""
echo "Configuring IAM permissions..."
SERVICE_ACCOUNT=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")-compute@developer.gserviceaccount.com

# Grant Secret Manager access
gcloud secrets add-iam-policy-binding $OPENAI_SECRET \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor" \
    --quiet

gcloud secrets add-iam-policy-binding $API_SECRET \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor" \
    --quiet

if [ ! -z "$PERPLEXITY_SECRET" ]; then
    gcloud secrets add-iam-policy-binding $PERPLEXITY_SECRET \
        --member="serviceAccount:$SERVICE_ACCOUNT" \
        --role="roles/secretmanager.secretAccessor" \
        --quiet
    echo "✓ IAM permissions configured for $OPENAI_SECRET, $API_SECRET, and $PERPLEXITY_SECRET"
else
    echo "✓ IAM permissions configured for $OPENAI_SECRET and $API_SECRET"
fi

# Configure Docker authentication
echo ""
echo "Configuring Docker authentication..."
gcloud auth configure-docker ${REGION}-docker.pkg.dev --quiet
echo "✓ Docker authentication configured"

echo ""
echo "Setup complete!"
echo ""
echo "Next steps:"
echo "1. Build and deploy: ./deploy.sh"
echo "2. Get service URL: gcloud run services describe $SERVICE_NAME --region $REGION --format='value(status.url)'"

