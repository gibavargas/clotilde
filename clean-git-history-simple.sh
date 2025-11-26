#!/bin/bash
# Simple script to remove old secret names from git history
# Uses git filter-branch to rewrite history

set -e

echo "=========================================="
echo "Git History Cleanup Script"
echo "=========================================="
echo ""
echo "This will remove old secret names from ALL git history:"
echo "  - openai-api-key"
echo "  - clotilde-api-key"
echo ""
echo "WARNING: This rewrites git history!"
echo "All commit hashes will change."
echo ""
read -p "Press Enter to continue or Ctrl+C to cancel..."

# Create backup branch
BACKUP_BRANCH="backup-before-cleanup-$(date +%Y%m%d-%H%M%S)"
echo ""
echo "Creating backup branch: $BACKUP_BRANCH"
git branch "$BACKUP_BRANCH"
echo "✓ Backup created at: $BACKUP_BRANCH"

# Rewrite history to replace old secret names
echo ""
echo "Rewriting git history..."
echo "This may take a few minutes..."

git filter-branch --force --tree-filter '
    # Replace in cloudbuild.yaml
    if [ -f cloudbuild.yaml ]; then
        sed -i.bak "s/openai-api-key/YOUR_OPENAI_SECRET_NAME/g" cloudbuild.yaml 2>/dev/null || true
        sed -i.bak "s/clotilde-api-key/YOUR_API_SECRET_NAME/g" cloudbuild.yaml 2>/dev/null || true
        rm -f cloudbuild.yaml.bak 2>/dev/null || true
    fi
    
    # Replace in deploy.sh
    if [ -f deploy.sh ]; then
        sed -i.bak "s/openai-api-key/YOUR_OPENAI_SECRET_NAME/g" deploy.sh 2>/dev/null || true
        sed -i.bak "s/clotilde-api-key/YOUR_API_SECRET_NAME/g" deploy.sh 2>/dev/null || true
        rm -f deploy.sh.bak 2>/dev/null || true
    fi
    
    # Replace in setup-gcloud.sh
    if [ -f setup-gcloud.sh ]; then
        sed -i.bak "s/openai-api-key/YOUR_OPENAI_SECRET_NAME/g" setup-gcloud.sh 2>/dev/null || true
        sed -i.bak "s/clotilde-api-key/YOUR_API_SECRET_NAME/g" setup-gcloud.sh 2>/dev/null || true
        rm -f setup-gcloud.sh.bak 2>/dev/null || true
    fi
' --prune-empty --tag-name-filter cat -- --all

# Clean up backup refs
echo ""
echo "Cleaning up backup refs..."
git for-each-ref --format="%(refname)" refs/original/ | xargs -n 1 git update-ref -d 2>/dev/null || true

# Expire reflog and garbage collect
echo "Running garbage collection..."
git reflog expire --expire=now --all
git gc --prune=now --aggressive

echo ""
echo "=========================================="
echo "History Cleanup Complete!"
echo "=========================================="
echo ""
echo "Backup branch: $BACKUP_BRANCH"
echo ""
echo "Verify the changes:"
echo "  git log --all --oneline"
echo "  git show HEAD:cloudbuild.yaml"
echo ""
echo "To push to remote (WARNING: rewrites remote history):"
echo "  git push origin --force --all"
echo "  git push origin --force --tags"
echo ""
echo "To restore from backup:"
echo "  git checkout $BACKUP_BRANCH"

