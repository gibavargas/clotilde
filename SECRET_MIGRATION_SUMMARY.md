# Secret Name Migration Summary

**Date**: 2025-11-24  
**Status**: ✅ Complete  
**Breaking Change**: Yes (requires migration)

## What Was Changed

### Problem
The codebase had hardcoded secret names (`openai-api-key`, `clotilde-api-key`) visible in the public GitHub repository. While this doesn't expose actual secret values, it reveals which secrets exist and makes them predictable targets.

### Solution
All secret names are now configurable via environment variables. The repository only contains placeholders, making it safe for public sharing.

## Files Modified

### Code Changes
- ✅ `cmd/clotilde/main.go`: Removed hardcoded secret names, requires `OPENAI_SECRET_NAME` and `API_SECRET_NAME` env vars
- ✅ `cloudbuild.yaml`: Added `_OPENAI_SECRET` and `_API_SECRET` substitution variables
- ✅ `deploy.sh`: Added required `OPENAI_SECRET` and `API_SECRET` environment variables
- ✅ `setup-gcloud.sh`: Updated to use configurable secret names

### New Files
- ✅ `migrate-secrets.sh`: Automated migration script
- ✅ `LOCAL_DOCKER.md`: Guide for local Docker development
- ✅ `SECRET_MIGRATION_SUMMARY.md`: This file

### Documentation Updates
- ✅ `agents.md`: Added comprehensive secret name configuration documentation
- ✅ `README.md`: Updated all examples to use placeholders
- ✅ `SECURITY.md`: Updated secret name references
- ✅ `QUICKSTART.md`: Updated secret name references
- ✅ `GUIA_SHORTCUT_IPHONE.md`: Updated secret name references
- ✅ `SHORTCUT_SETUP.md`: Updated secret name references
- ✅ `SHORTCUT_IMPORT.md`: Updated secret name references

## Migration Steps

### 1. Run Migration Script

```bash
chmod +x migrate-secrets.sh
./migrate-secrets.sh
```

This will:
- Generate unique secret names (e.g., `clotilde-oai-a1b2c3`)
- Create new secrets in Secret Manager
- Copy values from old secrets (if they exist)
- Grant IAM permissions
- Provide deployment commands

### 2. Deploy with New Secrets

**Option A: Cloud Build**
```bash
gcloud builds submit --config=cloudbuild.yaml \
  --substitutions=_OPENAI_SECRET=clotilde-oai-abc123,_API_SECRET=clotilde-auth-xyz789
```

**Option B: Deploy Script**
```bash
export OPENAI_SECRET=clotilde-oai-abc123
export API_SECRET=clotilde-auth-xyz789
./deploy.sh
```

### 3. Verify Deployment

Test the service:
```bash
curl -X POST https://your-service-url.run.app/chat \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"message":"teste"}'
```

### 4. Clean Up Old Secrets (After Verification)

```bash
gcloud secrets delete openai-api-key
gcloud secrets delete clotilde-api-key
```

## Local Development

See [LOCAL_DOCKER.md](LOCAL_DOCKER.md) for complete guide.

**Quick start**:
```bash
# Create .env file (never commit this!)
cat > .env << EOF
OPENAI_KEY_SECRET_NAME=sk-your-actual-key
API_KEY_SECRET_NAME=your-actual-api-key
GOOGLE_CLOUD_PROJECT=your-project-id
PORT=8080
EOF

# Build and run
docker build -t clotilde:local .
docker run --env-file .env -p 8080:8080 clotilde:local
```

## Environment Variables Reference

### For Cloud Run (Production)

| Variable | Source | Description |
|----------|--------|-------------|
| `OPENAI_KEY_SECRET_NAME` | Secret Manager (mounted) | Direct OpenAI API key value |
| `API_KEY_SECRET_NAME` | Secret Manager (mounted) | Direct API key value |
| `GOOGLE_CLOUD_PROJECT` | Cloud Build | GCP project ID |

### For Local Development (Secret Manager Lookup)

| Variable | Description |
|----------|-------------|
| `OPENAI_SECRET_NAME` | Name of OpenAI secret in Secret Manager |
| `API_SECRET_NAME` | Name of API key secret in Secret Manager |
| `GOOGLE_CLOUD_PROJECT` | GCP project ID |

### For Local Development (Direct Values)

| Variable | Description |
|----------|-------------|
| `OPENAI_KEY_SECRET_NAME` | Direct OpenAI API key value |
| `API_KEY_SECRET_NAME` | Direct API key value |
| `PORT` | Server port (default: 8080) |

**Priority**: Direct values (`OPENAI_KEY_SECRET_NAME`) > Secret Manager lookup

## Security Benefits

1. ✅ **No Secret Names in Public Repo**: Only placeholders visible
2. ✅ **Unpredictable Names**: Attackers can't guess secret names
3. ✅ **Explicit Configuration**: Forces conscious secret name management
4. ✅ **Git-Safe**: Repository can be public without exposing infrastructure

## Verification Checklist

- [x] Code compiles successfully
- [x] No hardcoded secret names in code
- [x] All documentation updated
- [x] Migration script created
- [x] Local development guide created
- [x] Agents.md documentation updated
- [ ] Migration script tested (manual step)
- [ ] Deployment verified with new secrets (manual step)
- [ ] Old secrets deleted (manual step)

## Next Steps

1. **Run migration script**: `./migrate-secrets.sh`
2. **Deploy with new secrets**: Use commands provided by migration script
3. **Test the service**: Verify it works correctly
4. **Delete old secrets**: After confirming everything works
5. **Save secret names**: Store them securely (not in git)

## Important Notes

⚠️ **Breaking Change**: Existing deployments will fail until secret names are configured.

✅ **Safe for Public Repo**: No secret names or values are exposed.

🔒 **Security Improvement**: Follows security best practices for secret management.

📝 **Documentation**: See [agents.md](agents.md) for detailed technical documentation.

