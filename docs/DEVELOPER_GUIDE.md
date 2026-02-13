# Bitrise App Secret Resource - Developer Quick Reference

## Resource Overview

The `bitrise_app_secret` resource manages environment variables (secrets) for Bitrise applications using the Bitrise API v0.1.

## API Endpoints Used

| Operation | Method | Endpoint | Status Code |
|-----------|--------|----------|-------------|
| Create | POST | `/v0.1/apps/{app-slug}/secrets` | 201 Created |
| Read | GET | `/v0.1/apps/{app-slug}/secrets/{secret-name}` | 200 OK |
| Update | PATCH | `/v0.1/apps/{app-slug}/secrets/{secret-name}` | 200 OK |
| Delete | DELETE | `/v0.1/apps/{app-slug}/secrets/{secret-name}` | 204 No Content |

## Schema Reference

### Required Fields
- `app_slug` (string, ForceNew) - Bitrise application identifier
- `name` (string, ForceNew) - Secret name/key
- `value` (string, Sensitive) - Secret value

### Optional Fields
- `is_protected` (bool, default: false) - Makes value unreadable via API
- `is_exposed_for_pull_requests` (bool, default: false) - Exposes secret to PR builds
- `expand_in_step_inputs` (bool, default: true) - Enables variable expansion

### Computed Fields
- `id` (string) - Format: `{app_slug}/{name}`

## Request/Response Structures

### Create Request Body
```json
{
  "name": "SECRET_NAME",
  "value": "secret-value",
  "is_protected": false,
  "is_exposed_for_pull_requests": false,
  "expand_in_step_inputs": true
}
```

### Update Request Body
```json
{
  "value": "new-value",
  "is_protected": false,
  "is_exposed_for_pull_requests": false,
  "expand_in_step_inputs": true
}
```

### Response Body
```json
{
  "id": "secret-id",
  "name": "SECRET_NAME",
  "value": "secret-value",  // Only if not protected
  "is_protected": false,
  "is_exposed_for_pull_requests": false,
  "expand_in_step_inputs": true
}
```

## Important Implementation Notes

### Protected Secrets
- When `is_protected = true`, the API does NOT return the `value` field in GET requests
- Terraform cannot detect drift in protected secret values
- Reading a protected secret only updates the flags, not the value
- Keep the current state value when a protected secret is read

### Resource ID Format
- Import/Export ID: `{app_slug}/{secret_name}`
- Example: `my-app-slug/API_KEY`

### ForceNew Attributes
- `app_slug` - Changing app requires new resource
- `name` - Changing name requires new resource (API doesn't support rename)

### State Management
```go
// During Read operation for protected secrets:
if !secretResp.IsProtected && secretResp.Value != "" {
    data.Value = types.StringValue(secretResp.Value)
}
// Otherwise, keep the existing state value
```

## Error Handling

| Scenario | HTTP Status | Action |
|----------|-------------|--------|
| Secret not found | 404 | Remove from state (Read), return error (Update/Delete) |
| Already deleted | 404 | No-op (Delete) |
| Unauthorized | 401 | Return error with message |
| Invalid request | 400 | Return error with API message |

## Testing Considerations

### Unit Tests
- Test protected vs non-protected secret handling
- Test state management for missing values
- Test import ID parsing

### Acceptance Tests
- Create, read, update, delete lifecycle
- Protected secret behavior
- PR exposure flag
- Variable expansion flag
- Import functionality
- ForceNew behavior for app_slug and name changes

## Code Snippets

### Creating HTTP Request
```go
url := fmt.Sprintf("%s/v0.1/apps/%s/secrets", r.endpoint, appSlug)
httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadJSON)))
httpReq.Header.Set("Content-Type", "application/json")
```

### Handling Protected Secrets in Read
```go
if !secretResp.IsProtected && secretResp.Value != "" {
    data.Value = types.StringValue(secretResp.Value)
}
// Keep existing state value for protected secrets
```

### Import ID Parsing
```go
parts := strings.Split(req.ID, "/")
if len(parts) != 2 {
    return error
}
app_slug := parts[0]
secret_name := parts[1]
```

## Common Patterns

### Update with Pointers
```go
isProtected := data.IsProtected.ValueBool()
secretReq := SecretUpdateRequest{
    Value: data.Value.ValueString(),
    IsProtected: &isProtected,  // Use pointer for optional fields
}
```

### Logging Best Practices
```go
tflog.Debug(ctx, "Creating secret", map[string]interface{}{
    "app_slug": appSlug,
    "name": secretName,
    // Never log the value!
})
```

## Security Best Practices

1. **Never log secret values** - Even in debug mode
2. **Mark value as Sensitive** in schema
3. **Handle protected secrets correctly** - Don't overwrite state value
4. **Validate input** - Check app_slug and name formats
5. **Clear error messages** - Don't expose sensitive data in errors

## Related Files

- Implementation: `internal/provider/app_secrets_resource.go`
- Documentation: `docs/resources/bitrise_app_secret.md`
- Examples: `examples/resources/bitrise_app_secret/`
- Tests: `internal/provider/app_secrets_resource_test.go` (to be created)

## API Documentation References

- [Bitrise API Secrets Endpoints](https://api-docs.bitrise.io/#/secrets)
- [Managing Secrets with the API](https://docs.bitrise.io/en/bitrise-ci/api/managing-secrets-with-the-api.html)
- [Step Input Properties](https://devcenter.bitrise.io/en/references/steps-reference/step-inputs-reference.html#step-input-properties)
