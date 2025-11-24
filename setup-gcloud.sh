#!/bin/bash
# Google Cloud Setup Script for Clotilde
# Prerequisites: gcloud CLI installed and authenticated

set -e

# Configuration
PROJECT_ID=${GOOGLE_CLOUD_PROJECT:-$(gcloud config get-value project)}
REGION=${REGION:-us-central1}
REPO_NAME=${REPO_NAME:-clotilde-repo}
SERVICE_NAME=${SERVICE_NAME:-clotilde}

echo "Setting up Clotilde on Google Cloud..."
echo "Project ID: $PROJECT_ID"
echo "Region: $REGION"
echo ""

# Set project
gcloud config set project $PROJECT_ID

# Enable required APIs
echo "Enabling required APIs..."
gcloud services enable \
    artifactregistry.googleapis.com \
    secretmanager.googleapis.com \
    run.googleapis.com \
    cloudbuild.googleapis.com

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
if ! gcloud secrets describe YOUR_OPENAI_SECRET_NAME &>/dev/null; then
    echo -n "Enter your OpenAI API key: "
    read -s OPENAI_KEY
    echo ""
    echo -n "$OPENAI_KEY" | gcloud secrets create YOUR_OPENAI_SECRET_NAME \
        --data-file=- \
        --replication-policy="automatic" \
        --quiet
    echo "✓ OpenAI API key secret created"
else
    echo "✓ OpenAI API key secret already exists"
fi

# Clotilde API Key
if ! gcloud secrets describe YOUR_API_SECRET_NAME &>/dev/null; then
    echo -n "Enter your Clotilde API key (for authenticating requests): "
    read -s CLOTILDE_KEY
    echo ""
    echo -n "$CLOTILDE_KEY" | gcloud secrets create YOUR_API_SECRET_NAME \
        --data-file=- \
        --replication-policy="automatic" \
        --quiet
    echo "✓ Clotilde API key secret created"
else
    echo "✓ Clotilde API key secret already exists"
fi

# Get Cloud Run service account
echo ""
echo "Configuring IAM permissions..."
SERVICE_ACCOUNT=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")-compute@developer.gserviceaccount.com

# Grant Secret Manager access
gcloud secrets add-iam-policy-binding YOUR_OPENAI_SECRET_NAME \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor" \
    --quiet

gcloud secrets add-iam-policy-binding YOUR_API_SECRET_NAME \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor" \
    --quiet

echo "✓ IAM permissions configured"

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
echo "2. Or use Cloud Build: gcloud builds submit --config=cloudbuild.yaml"
echo "3. Get service URL: gcloud run services describe $SERVICE_NAME --region $REGION --format='value(status.url)'"

