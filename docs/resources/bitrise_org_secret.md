# bitrise_org_secret Resource

Manages a workspace (organization) secret in Bitrise. Workspace secrets are shared by every app in the workspace. An app-level secret with the same name (`bitrise_app_secret`) overrides the workspace one.

## Example Usage

```terraform
resource "bitrise_org_secret" "github_token" {
  org_slug                     = "your-workspace-slug"
  name                         = "GITHUB_TOKEN"
  value                        = var.github_token
  is_protected                 = true
  is_exposed_for_pull_requests = true
  expand_in_step_inputs        = false
}
```

## Argument Reference

* `org_slug` - (Required, ForceNew) The slug of the Bitrise workspace (organization).
* `name` - (Required, ForceNew) The name (key) of the secret.
* `value` - (Required, Sensitive) The value of the secret.
* `is_protected` - (Optional) If `true`, the value cannot be retrieved via the API or the UI. Default: `false`. Bitrise does not allow unprotecting a secret, so changing `true` to `false` destroys and recreates it.
* `is_exposed_for_pull_requests` - (Optional) If `true`, the secret is available to pull request builds. Default: `false`.
* `expand_in_step_inputs` - (Optional) If `true`, variable expansion is enabled for this secret in step inputs. Default: `true`.

## Attribute Reference

* `id` - `org_slug/name`.
* `scope` - The scope reported by Bitrise (e.g. `workspace`).

## Import

```shell
terraform import bitrise_org_secret.github_token your-workspace-slug/GITHUB_TOKEN
```

Protected values are never returned by the API, so after importing a protected secret the next `apply` writes the configured `value`. Drift on protected values cannot be detected.

## API Documentation

- POST `/v0.1/organizations/{org-slug}/secrets`
- GET `/v0.1/organizations/{org-slug}/secrets/{secret-name}`
- PATCH `/v0.1/organizations/{org-slug}/secrets/{secret-name}`
- DELETE `/v0.1/organizations/{org-slug}/secrets/{secret-name}`
