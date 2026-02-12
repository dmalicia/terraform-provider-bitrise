# bitrise_app_secret Resource

Manages secrets (environment variables) for a Bitrise application. Secrets are securely stored and can be used in your build workflows.

## Example Usage

```terraform
# Create a simple secret
resource "bitrise_app_secret" "api_key" {
  app_slug = "your-app-slug"
  name     = "API_KEY"
  value    = "your-secret-api-key"
}

# Create a protected secret (value cannot be retrieved via API)
resource "bitrise_app_secret" "production_key" {
  app_slug     = "your-app-slug"
  name         = "PRODUCTION_KEY"
  value        = "super-secret-key"
  is_protected = true
}

# Create a secret exposed to pull requests
resource "bitrise_app_secret" "pr_token" {
  app_slug                      = "your-app-slug"
  name                          = "PR_TOKEN"
  value                         = "pr-access-token"
  is_exposed_for_pull_requests = true
}

# Create a secret with custom expansion settings
resource "bitrise_app_secret" "build_config" {
  app_slug               = "your-app-slug"
  name                   = "BUILD_CONFIG"
  value                  = "$ENV_VAR/path"
  expand_in_step_inputs = false  # Disable variable expansion
}
```

## Argument Reference

The following arguments are supported:

* `app_slug` - (Required, ForceNew) The slug of the Bitrise app. Changing this forces a new resource to be created.
* `name` - (Required, ForceNew) The name (key) of the secret. Changing this forces a new resource to be created.
* `value` - (Required, Sensitive) The value of the secret. This is marked as sensitive and will not appear in logs.
* `is_protected` - (Optional) If `true`, the secret value cannot be retrieved via the API. Default: `false`. **Warning:** Once a secret is protected, you cannot retrieve its value through Terraform.
* `is_exposed_for_pull_requests` - (Optional) If `true`, the secret will be available for pull request builds. Default: `false`.
* `expand_in_step_inputs` - (Optional) If `true`, variable expansion will be enabled for this secret in step inputs. Default: `true`. See [Bitrise documentation](https://devcenter.bitrise.io/en/references/steps-reference/step-inputs-reference.html#step-input-properties) for details.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the secret in the format `app_slug/secret_name`.

## Import

Secrets can be imported using the format `app_slug/secret_name`:

```shell
terraform import bitrise_app_secret.api_key your-app-slug/API_KEY
```

**Note:** When importing a protected secret, Terraform won't be able to retrieve the value from the API. You'll need to manually set the `value` in your configuration to match the actual secret value to avoid unwanted updates.

## API Documentation

This resource uses the following Bitrise API endpoints:

- POST `/v0.1/apps/{app-slug}/secrets` - Create a new secret
- GET `/v0.1/apps/{app-slug}/secrets/{secret-name}` - Read a secret
- PATCH `/v0.1/apps/{app-slug}/secrets/{secret-name}` - Update a secret
- DELETE `/v0.1/apps/{app-slug}/secrets/{secret-name}` - Delete a secret

For more information, see the [Bitrise API documentation](https://docs.bitrise.io/en/bitrise-ci/api/managing-secrets-with-the-api.html).
