# Security Documentation

## Overview

Clotilde CarPlay Assistant implements multiple layers of security to protect user prompts, prevent exploitation, and ensure secure operation in production.

## Threat Model

### Assets to Protect

1. **User Prompts**: Voice input from CarPlay users
2. **API Keys**: OpenAI API key and service authentication key
3. **Service Availability**: Protection against DDoS and abuse
4. **User Privacy**: No data retention or logging of sensitive information

### Threat Vectors

1. **Unauthorized Access**: Unauthorized API usage
2. **Data Exfiltration**: Theft of API keys or user prompts
3. **Service Abuse**: Rate limit bypass, DDoS attacks
4. **Container Exploitation**: Container escape, privilege escalation
5. **Man-in-the-Middle**: Interception of requests/responses

## Security Measures

### 1. Authentication

**Implementation**: API key validation via `X-API-Key` header

- **Location**: `internal/auth/auth.go`
- **Method**: Constant-time comparison to prevent timing attacks
- **Storage**: Google Secret Manager (not environment variables)
- **Validation**: Every request (except `/health`) requires valid API key

**Configuration**:
- Secret name: `clotilde-api-key` (configurable via `API_KEY_SECRET_NAME`)
- Retrieved at startup from Secret Manager
- Never logged or exposed in error messages

### 2. Rate Limiting

**Implementation**: Per-IP and per-API-key rate limiting

- **Location**: `internal/ratelimit/ratelimit.go`
- **Limits**:
  - 10 requests per minute per API key
  - 100 requests per hour per IP address
- **Storage**: In-memory map with automatic cleanup
- **Scope**: Per-API-key (preferred) or per-IP (fallback)

**Protection Against**:
- DDoS attacks
- API key abuse
- Resource exhaustion

### 3. Input Validation

**Implementation**: Request size and content validation

- **Location**: `internal/validator/validator.go`
- **Limits**:
  - Request body: 5KB maximum
  - Message length: 1000 characters maximum
- **Validation**: JSON structure, required fields, length checks

**Protection Against**:
- Buffer overflow attempts
- Resource exhaustion via large payloads
- Malformed request attacks

### 4. Secrets Management

**Implementation**: Google Secret Manager

- **Secrets Stored**:
  - `openai-api-key`: OpenAI API key
  - `clotilde-api-key`: Service authentication key
- **Access**: Cloud Run service account with least privilege
- **Rotation**: Supported via Secret Manager versioning
- **Never**: Stored in environment variables, code, or logs

**IAM Configuration**:
```bash
# Service account needs this role:
roles/secretmanager.secretAccessor
```

### 5. Container Security

**Implementation**: Minimal, hardened Docker image

- **Base Image**: Alpine Linux (~5MB)
- **User**: Non-root (UID 1000)
- **Filesystem**: Read-only where possible
- **Packages**: Minimal (only CA certificates)
- **Binary**: Statically compiled, stripped symbols

**Dockerfile Security**:
- Multi-stage build (no build tools in final image)
- Non-root user creation
- Minimal base image
- No unnecessary packages

### 6. Network Security

**Implementation**: HTTPS/TLS enforcement and CORS

- **HTTPS**: Enforced by Cloud Run (all traffic encrypted)
- **CORS**: Restricted to Apple Shortcuts origin (configurable)
- **Headers**: Security headers via Cloud Run
- **No HTTP**: Cloud Run only accepts HTTPS

### 7. Secure Logging

**Implementation**: Metadata-only logging

**What is Logged**:
- Request timestamp
- IP address hash (not actual IP)
- API key hash (not actual key)
- Message length (not content)
- Response length (not content)

**What is NOT Logged**:
- Full user prompts
- API keys
- Complete request/response bodies
- Sensitive headers

**Example Log Entry**:
```
Request received: IP=ip_12345, MessageLength=42
Response generated: Length=156
```

### 8. Prompt Privacy

**Protection Measures**:
- No logging of full prompts
- No data retention (stateless service)
- HTTPS-only transmission
- Secrets never in logs or error messages
- No database or persistent storage

**Data Flow**:
1. User speaks → Apple Shortcut
2. Shortcut → HTTPS POST to Cloud Run
3. Cloud Run → OpenAI API (HTTPS)
4. Response → User via Shortcut
5. No storage, no logging of content

### 9. DDoS Protection

**Layers**:
1. **Cloud Run**: Built-in DDoS protection
2. **Application**: Rate limiting (10/min, 100/hour)
3. **Input Validation**: Request size limits
4. **Auto-scaling**: Cloud Run scales to handle traffic

**Configuration**:
- Min instances: 0 (cost optimization)
- Max instances: 10 (prevent runaway costs)
- Timeout: 30 seconds

### 10. Monitoring and Alerting

**Recommended Alerts** (configure in Cloud Monitoring):

1. **High Error Rate**:
   - Alert if error rate > 5% over 5 minutes

2. **Unusual Traffic**:
   - Alert if requests/minute > 1000

3. **Failed Authentication**:
   - Alert if auth failures > 50 in 5 minutes

4. **Rate Limit Hits**:
   - Alert if rate limit errors > 100 in 5 minutes

## Security Best Practices

### For Developers

1. **Never commit secrets**: Use Secret Manager
2. **Review dependencies**: Keep Go modules updated
3. **Test security**: Verify rate limiting and auth work
4. **Monitor logs**: Check for suspicious patterns
5. **Rotate keys**: Regularly rotate API keys

### For Deployment

1. **Least privilege**: Service account only needs Secret Manager access
2. **Network isolation**: Consider VPC connector for additional isolation
3. **Resource limits**: Keep memory/CPU at minimum needed
4. **Version pinning**: Pin Secret Manager secret versions
5. **Regular updates**: Update base images and dependencies

### For API Key Management

1. **Generate strong keys**: Use cryptographically secure random keys
2. **Store securely**: Only in Secret Manager
3. **Rotate regularly**: Update keys every 90 days
4. **Monitor usage**: Check Cloud Logging for unusual patterns
5. **Revoke compromised keys**: Immediately if suspected breach

## Incident Response

### If API Key is Compromised

1. Immediately rotate the key in Secret Manager
2. Update the Apple Shortcut with new key
3. Review Cloud Logging for unauthorized access
4. Check for unusual traffic patterns

### If Service is Under Attack

1. Check Cloud Run metrics for traffic spikes
2. Review rate limiting logs
3. Consider temporarily reducing max instances
4. Enable Cloud Armor if attacks persist

### If Container is Exploited

1. Immediately stop the Cloud Run service
2. Review container logs
3. Rebuild and redeploy with updated base image
4. Review Secret Manager access logs

## Compliance Notes

- **No PII Storage**: Service does not store personal information
- **No Data Retention**: Stateless service, no persistent storage
- **HTTPS Only**: All traffic encrypted in transit
- **Minimal Logging**: Only metadata, no sensitive data

## Security Checklist

Before deploying to production:

- [ ] Secrets created in Secret Manager
- [ ] Service account has correct IAM roles
- [ ] API key is strong and unique
- [ ] CORS configured appropriately
- [ ] Rate limits set appropriately
- [ ] Monitoring alerts configured
- [ ] Container runs as non-root
- [ ] No secrets in code or environment variables
- [ ] HTTPS enforced
- [ ] Logging verified (no sensitive data)

## Contact

For security issues, please contact the project maintainer directly.

