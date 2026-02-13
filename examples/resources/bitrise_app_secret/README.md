# Bitrise App Secret Resource Examples

This directory contains examples of using the `bitrise_app_secret` resource to manage secrets in your Bitrise applications.

## Prerequisites

- Terraform >= 1.0
- A Bitrise Personal Access Token
- A Bitrise application slug

## Usage

1. Set your Bitrise token:
```bash
export TF_VAR_bitrise_token="your-bitrise-token"
export TF_VAR_app_slug="your-app-slug"
```

2. Initialize Terraform:
```bash
terraform init
```

3. Review the plan:
```bash
terraform plan
```

4. Apply the configuration:
```bash
terraform apply
```

## Examples Included

### 1. Basic Secret
Creates a simple secret with just a name and value.

### 2. Protected Secret
Creates a secret that cannot be retrieved via the API once created. Useful for highly sensitive values.

### 3. Pull Request Secret
Creates a secret that is exposed to pull request builds. Be cautious with this setting.

### 4. Full Configuration
Demonstrates all available options for secret configuration.

### 5. Multiple Secrets with for_each
Shows how to manage multiple secrets efficiently using Terraform's `for_each` meta-argument.

## Important Notes

### Protected Secrets
When a secret is marked as `is_protected = true`:
- The value cannot be retrieved via the API
- Terraform won't be able to detect drift in the value
- Reimporting the secret will require manually setting the value
- Consider using this for production credentials

### Pull Request Exposure
When `is_exposed_for_pull_requests = true`:
- The secret will be available in PR builds
- This can be a security risk if the repository accepts PRs from forks
- Only enable for non-sensitive values or trusted repositories

### Variable Expansion
The `expand_in_step_inputs` setting controls whether environment variable expansion occurs:
- `true` (default): `$OTHER_VAR` will be expanded to its value
- `false`: `$OTHER_VAR` will be treated as literal text

## Importing Existing Secrets

To import existing Bitrise secrets:

```bash
terraform import bitrise_app_secret.api_key your-app-slug/API_KEY
```

Note: For protected secrets, you'll need to manually set the value in your Terraform configuration after import.

## Security Best Practices

1. **Never commit secrets to version control**
   - Use environment variables or secret management tools
   - Consider using Terraform Cloud/Enterprise workspaces with encrypted variables

2. **Use protected secrets for sensitive data**
   ```hcl
   resource "bitrise_app_secret" "prod_key" {
     app_slug     = var.app_slug
     name         = "PRODUCTION_KEY"
     value        = var.production_key
     is_protected = true
   }
   ```

3. **Minimize PR exposure**
   - Only expose secrets to PRs when absolutely necessary
   - Use separate, limited-privilege secrets for PR builds

4. **Rotate secrets regularly**
   - Update secret values periodically
   - Use Terraform to ensure consistent rotation across environments

## Troubleshooting

### Secret not updating
- Check if the secret is protected - protected secrets may require recreation
- Verify your Bitrise token has the correct permissions

### Import fails
- Ensure the format is exactly `app_slug/SECRET_NAME`
- Verify the secret exists in Bitrise
- Check that your token has read access to secrets

### Value drift not detected
- This is expected for protected secrets
- Consider recreating the secret if you suspect the value is wrong

## Related Resources

- [Bitrise API Documentation](https://docs.bitrise.io/en/bitrise-ci/api/managing-secrets-with-the-api.html)
- [Terraform Provider Documentation](../../docs/resources/bitrise_app_secret.md)
