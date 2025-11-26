#!/bin/bash
# Script to remove old secret names from git history
# WARNING: This rewrites git history. Use with caution!

set -e

echo "=========================================="
echo "Git History Cleanup Script"
echo "=========================================="
echo ""
echo "This script will:"
echo "1. Create a backup branch"
echo "2. Remove old secret names from git history"
echo "3. Show you how to force push (if needed)"
echo ""
echo "WARNING: This rewrites git history!"
echo "All commits will have new hashes."
echo ""
read -p "Press Enter to continue or Ctrl+C to cancel..."

# Create backup branch
BACKUP_BRANCH="backup-before-history-cleanup-$(date +%Y%m%d-%H%M%S)"
echo ""
echo "Creating backup branch: $BACKUP_BRANCH"
git branch "$BACKUP_BRANCH"
echo "✓ Backup branch created"

# Old secret names to remove
OLD_NAMES=("openai-api-key" "clotilde-api-key")

# Use git filter-branch to remove old secret names
echo ""
echo "Rewriting git history to remove old secret names..."
echo "This may take a few minutes..."

# Remove from cloudbuild.yaml
git filter-branch --force --index-filter \
    'git checkout --ignore-skip-worktree-bits HEAD -- cloudbuild.yaml && \
     if [ -f cloudbuild.yaml ]; then
         sed -i.bak "s/openai-api-key/YOUR_OPENAI_SECRET_NAME/g" cloudbuild.yaml
         sed -i.bak "s/clotilde-api-key/YOUR_API_SECRET_NAME/g" cloudbuild.yaml
         rm -f cloudbuild.yaml.bak
         git add cloudbuild.yaml
     fi' \
    --prune-empty --tag-name-filter cat -- --all

# Remove from deploy.sh (if it exists in history)
git filter-branch --force --index-filter \
    'git checkout --ignore-skip-worktree-bits HEAD -- deploy.sh && \
     if [ -f deploy.sh ]; then
         sed -i.bak "s/openai-api-key/YOUR_OPENAI_SECRET_NAME/g" deploy.sh
         sed -i.bak "s/clotilde-api-key/YOUR_API_SECRET_NAME/g" deploy.sh
         rm -f deploy.sh.bak
         git add deploy.sh
     fi' \
    --prune-empty --tag-name-filter cat -- --all

# Remove from setup-gcloud.sh (if it exists in history)
git filter-branch --force --index-filter \
    'git checkout --ignore-skip-worktree-bits HEAD -- setup-gcloud.sh && \
     if [ -f setup-gcloud.sh ]; then
         sed -i.bak "s/openai-api-key/YOUR_OPENAI_SECRET_NAME/g" setup-gcloud.sh
         sed -i.bak "s/clotilde-api-key/YOUR_API_SECRET_NAME/g" setup-gcloud.sh
         rm -f setup-gcloud.sh.bak
         git add setup-gcloud.sh
     fi' \
    --prune-empty --tag-name-filter cat -- --all

# Clean up backup refs
echo ""
echo "Cleaning up backup refs..."
git for-each-ref --format="%(refname)" refs/original/ | xargs -n 1 git update-ref -d

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
echo "  git show HEAD:cloudbuild.yaml | grep -i secret"
echo ""
echo "If you're satisfied, you can:"
echo "1. Delete the backup branch: git branch -D $BACKUP_BRANCH"
echo "2. Force push to remote (if needed):"
echo "   git push origin --force --all"
echo "   git push origin --force --tags"
echo ""
echo "WARNING: Force pushing rewrites remote history!"
echo "Make sure all team members are aware before force pushing."

