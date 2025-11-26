#!/bin/bash
if [ -f cloudbuild.yaml ]; then
    # Use a temporary file for macOS compatibility
    sed 's/'\''openai-api-key'\''/'\''your-openai-secret-name'\''  # Override with --substitutions=_OPENAI_SECRET=actual-name/g' cloudbuild.yaml > cloudbuild.yaml.tmp
    sed 's/'\''clotilde-api-key'\''/'\''your-api-secret-name'\''        # Override with --substitutions=_API_SECRET=actual-name/g' cloudbuild.yaml.tmp > cloudbuild.yaml.new
    mv cloudbuild.yaml.new cloudbuild.yaml
    rm -f cloudbuild.yaml.tmp
fi

