# Quick Start: Bitrise Terraform Provider

This guide will help you get started with managing Bitrise applications using Terraform in under 10 minutes.

## Prerequisites

- Terraform >= 1.0 installed
- A Bitrise account with an organization
- A Bitrise Personal Access Token ([create one here](https://app.bitrise.io/me/profile#/security))
- A git repository (GitHub, GitLab, or Bitbucket)

## What You'll Build

This quickstart will walk you through:
1. Registering a Bitrise application
2. Configuring SSH keys for repository access
3. Setting up project configuration
4. Adding secrets
5. Configuring team access

## Step 1: Create Your First Configuration

Create a file named `main.tf`:

```hcl
terraform {
  required_providers {
    bitrise = {
      source = "registry.terraform.io/your-org/bitrise"
    }
  }
}

provider "bitrise" {
  endpoint = "https://api.bitrise.io"
  token    = var.bitrise_token
}

variable "bitrise_token" {
  description = "Bitrise Personal Access Token"
  type        = string
  sensitive   = true
}

variable "organization_slug" {
  description = "Your Bitrise organization slug"
  type        = string
}

# 1. Register your application
resource "bitrise_app" "my_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/myrepo"
  type              = "git"
  git_repo_slug     = "myrepo"
  git_owner         = "myorg"
  organization_slug = var.organization_slug
  is_public         = false
}

# 2. Configure SSH keys (for private repos)
resource "tls_private_key" "bitrise_ssh" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "bitrise_app_ssh" "my_app_ssh" {
  app_slug             = bitrise_app.my_app.app_slug
  auth_ssh_private_key = tls_private_key.bitrise_ssh.private_key_pem
  auth_ssh_public_key  = tls_private_key.bitrise_ssh.public_key_openssh
}

# 3. Complete app setup
resource "bitrise_app_finish" "my_app_config" {
  app_slug          = bitrise_app.my_app.app_slug
  project_type      = "ios"
  stack_id          = "osx-xcode-14.2.x"
  config            = file("${path.module}/bitrise.yml")
  mode              = "manual"
  organization_slug = var.organization_slug
}

# 4. Add a secret
resource "bitrise_app_secret" "api_key" {
  app_slug     = bitrise_app.my_app.app_slug
  name         = "API_KEY"
  value        = var.api_key
  is_protected = true
}

# 5. Configure team access
resource "bitrise_app_roles" "developers" {
  app_slug  = bitrise_app.my_app.app_slug
  role_name = "developer"
  groups    = ["dev-team"]
}

output "app_slug" {
  value       = bitrise_app.my_app.app_slug
  description = "The slug of the created app"
}
```

## Step 2: Create a Basic Bitrise Configuration

Create a file named `bitrise.yml`:

```yaml
---
format_version: '11'
default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
project_type: ios

workflows:
  primary:
    steps:
    - activate-ssh-key@4:
        run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
    - git-clone@8: {}
    - cache-pull@2: {}
    - certificate-and-profile-installer@1: {}
    - xcode-archive@4:
        inputs:
        - project_path: "$BITRISE_PROJECT_PATH"
        - scheme: "$BITRISE_SCHEME"
    - deploy-to-bitrise-io@2: {}
    - cache-push@2: {}

app:
  envs:
  - BITRISE_PROJECT_PATH: YourProject.xcworkspace
  - BITRISE_SCHEME: YourScheme
```

## Step 3: Set Your Variables

Create a `terraform.tfvars` file (add to `.gitignore`!):

```hcl
bitrise_token     = "your-bitrise-token-here"
organization_slug = "your-org-slug"
api_key          = "your-api-key"
```

Or use environment variables:

```bash
export TF_VAR_bitrise_token="your-bitrise-token"
export TF_VAR_organization_slug="your-org-slug"
export TF_VAR_api_key="your-api-key"
```

## Step 4: Initialize and Apply

```bash
# Initialize Terraform
terraform init

# Preview what will be created
terraform plan

# Create all resources
terraform apply
```

Type `yes` when prompted.

## Step 5: Verify

1. Go to https://app.bitrise.io
2. Find your newly created app
3. Check the Secrets tab for your API_KEY
4. Check the Team tab for access configuration

## Quick Start Examples by Use Case

## Quick Start Examples by Use Case

### Just Managing Secrets (Existing App)

If you already have a Bitrise app and just want to manage secrets:

```hcl
provider "bitrise" {
  endpoint = "https://api.bitrise.io"
  token    = var.bitrise_token
}

resource "bitrise_app_secret" "my_secret" {
  app_slug     = "existing-app-slug"
  name         = "MY_SECRET"
  value        = "secret-value"
  is_protected = true
}
```

### Android App Setup

```hcl
resource "bitrise_app" "android_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/android-app"
  type              = "git"
  git_repo_slug     = "android-app"
  git_owner         = "myorg"
  organization_slug = var.organization_slug
}

resource "bitrise_app_finish" "android_config" {
  app_slug          = bitrise_app.android_app.app_slug
  project_type      = "android"
  stack_id          = "linux-docker-android-22.04"
  config            = file("bitrise-android.yml")
  mode              = "manual"
  organization_slug = var.organization_slug
  
  envs = {
    GRADLE_BUILD_TOOL_VERSION = "7.4"
    ANDROID_SDK_VERSION       = "33"
  }
}
```

### React Native App Setup

```hcl
resource "bitrise_app" "rn_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/rn-app"
  type              = "git"
  git_repo_slug     = "rn-app"
  git_owner         = "myorg"
  organization_slug = var.organization_slug
}

resource "bitrise_app_finish" "rn_config" {
  app_slug          = bitrise_app.rn_app.app_slug
  project_type      = "react-native"
  stack_id          = "osx-xcode-14.2.x"
  config            = file("bitrise-rn.yml")
  mode              = "manual"
  organization_slug = var.organization_slug
  
  envs = {
    NODE_VERSION = "18"
    YARN_VERSION = "1.22"
  }
}
```

## Common Patterns

### Multiple Secrets

## Common Patterns

### Multiple Secrets

```hcl
locals {
  secrets = {
    "AWS_ACCESS_KEY_ID"     = var.aws_access_key
    "AWS_SECRET_ACCESS_KEY" = var.aws_secret_key
    "DATABASE_URL"          = var.database_url
  }
}

resource "bitrise_app_secret" "secrets" {
  for_each = local.secrets

  app_slug     = bitrise_app.my_app.app_slug
  name         = each.key
  value        = each.value
  is_protected = true
}
```

### Using Existing SSH Keys

```hcl
resource "bitrise_app_ssh" "from_files" {
  app_slug             = bitrise_app.my_app.app_slug
  auth_ssh_private_key = file("~/.ssh/id_rsa")
  auth_ssh_public_key  = file("~/.ssh/id_rsa.pub")
}
```

### Environment-Specific Configuration

```hcl
variable "environment" {
  type    = string
  default = "development"
}

resource "bitrise_app_secret" "api_url" {
  app_slug = bitrise_app.my_app.app_slug
  name     = "API_URL"
  value    = var.environment == "production" ? "https://api.prod.com" : "https://api.dev.com"
}

resource "bitrise_app_secret" "build_type" {
  app_slug = bitrise_app.my_app.app_slug
  name     = "BUILD_TYPE"
  value    = upper(var.environment)
}
```

## Available Resources

The provider includes these resources:

- **bitrise_app** - Register applications
- **bitrise_app_ssh** - Configure SSH keys
- **bitrise_app_finish** - Complete app setup
- **bitrise_app_secret** - Manage secrets
- **bitrise_app_bitrise_yml** - Manage YAML configuration
- **bitrise_app_roles** - Manage team access

## Next Steps

1. **Explore Examples**: Check the `examples/` directory for complete examples
   - `examples/resources/bitrise_app/` - App registration
   - `examples/resources/bitrise_app_ssh/` - SSH configuration
   - `examples/resources/bitrise_app_finish/` - App setup with multiple project types
   - `examples/resources/bitrise_app_secret/` - Secret management patterns

2. **Read Full Documentation**: 
   - Resource docs: `docs/resources/`
   - Data source docs: `docs/data-sources/`
   - Implementation summary: `IMPLEMENTATION_SUMMARY.md`

3. **Import Existing Resources**:
   ```bash
   # Import an existing app
   terraform import bitrise_app.existing your-app-slug
   
   # Import an existing secret
   terraform import bitrise_app_secret.existing your-app-slug/SECRET_NAME
   
   # Import role configuration
   terraform import bitrise_app_roles.existing your-app-slug/admin
   ```

## Troubleshooting

### "Unauthorized" Error
- Verify your token is correct
- Check that your token has admin access to the organization
- Ensure the token hasn't expired

### App Registration Fails
- Verify the git repository URL is correct and accessible
- Check that your Bitrise account has access to the repository
- Ensure the organization slug is correct

### SSH Key Configuration Issues
- Verify the private and public keys match
- Check that the keys are in the correct format (PEM for private, OpenSSH for public)
- For provider registration, ensure your Bitrise account has OAuth access to the git provider

### App Finish Fails
- Verify the stack_id is valid for your project type
- Check that the bitrise.yml file exists and is valid YAML
- Ensure the project_type matches your repository structure

### Secret Not Showing in Bitrise
- Wait a few seconds and refresh
- Check the app slug is correct
- Verify in Terraform state: `terraform show`

### Changes Detected Every Plan (Protected Secrets)
- This is expected for protected secrets
- Terraform cannot read the value to detect drift
- Either accept this or use non-protected secrets

## Important Security Notes

⚠️ **Never commit secrets to version control!**

- Use environment variables: `TF_VAR_*`
- Use `.tfvars` files (add to `.gitignore`)
- Use a secrets management solution (Vault, AWS Secrets Manager, etc.)
- Use Terraform Cloud/Enterprise workspace variables

⚠️ **SSH Key Security**

- Generate dedicated SSH keys for Bitrise (don't reuse personal keys)
- Use the `tls_private_key` resource to manage keys within Terraform
- Private keys are marked as sensitive and won't appear in logs
- Rotate keys regularly

⚠️ **Protected Secrets**

- Use `is_protected = true` for production credentials
- Once protected, values cannot be retrieved via API
- Terraform won't be able to detect drift for protected secrets

⚠️ **Pull Request Exposure**

- Only enable `is_exposed_for_pull_requests` if necessary
- Use separate, limited-privilege secrets for PRs
- Never expose production credentials to PRs

## Getting Help

- **Full Documentation**: `docs/resources/` and `docs/data-sources/`
- **Examples**: `examples/resources/`
- **Testing Guide**: `TESTING.md`
- **Implementation Summary**: `IMPLEMENTATION_SUMMARY.md`
- **Bitrise API Docs**: https://api-docs.bitrise.io/

## Complete Production Example

Here's a production-ready example with a complete workflow:

## Complete Production Example

Here's a production-ready example with a complete workflow:

```hcl
terraform {
  required_providers {
    bitrise = {
      source = "registry.terraform.io/your-org/bitrise"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
}

provider "bitrise" {
  endpoint = "https://api.bitrise.io"
  token    = var.bitrise_token
}

variable "bitrise_token" {
  type      = string
  sensitive = true
}

variable "organization_slug" {
  type = string
}

variable "environment" {
  type    = string
  default = "production"
}

# 1. Register the application
resource "bitrise_app" "mobile_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/mobile-app"
  type              = "git"
  git_repo_slug     = "mobile-app"
  git_owner         = "myorg"
  organization_slug = var.organization_slug
  is_public         = false
}

# 2. Generate and configure SSH keys
resource "tls_private_key" "bitrise_ssh" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "bitrise_app_ssh" "mobile_app_ssh" {
  app_slug                              = bitrise_app.mobile_app.app_slug
  auth_ssh_private_key                  = tls_private_key.bitrise_ssh.private_key_pem
  auth_ssh_public_key                   = tls_private_key.bitrise_ssh.public_key_openssh
  is_register_key_into_provider_service = true
}

# 3. Complete app setup for iOS
resource "bitrise_app_finish" "ios_config" {
  app_slug          = bitrise_app.mobile_app.app_slug
  project_type      = "ios"
  stack_id          = "osx-xcode-14.2.x"
  config            = file("${path.module}/bitrise-ios.yml")
  mode              = "manual"
  organization_slug = var.organization_slug
  
  envs = {
    FASTLANE_XCODE_VERSION = "14.2"
    ENVIRONMENT            = var.environment
  }
}

# 4. Add secrets
locals {
  secrets = {
    # Protected production secrets
    AWS_ACCESS_KEY_ID     = { value = var.aws_access_key_id, protected = true, pr_exposed = false }
    AWS_SECRET_ACCESS_KEY = { value = var.aws_secret_access_key, protected = true, pr_exposed = false }
    SIGNING_KEY           = { value = var.signing_key, protected = true, pr_exposed = false }
    
    # Configuration secrets
    API_URL               = { value = var.api_url, protected = false, pr_exposed = true }
    BUILD_NUMBER_PREFIX   = { value = upper(var.environment), protected = false, pr_exposed = true }
    
    # Notification webhook (safe for PRs)
    SLACK_WEBHOOK_URL     = { value = var.slack_webhook_url, protected = false, pr_exposed = true }
  }
}

resource "bitrise_app_secret" "secrets" {
  for_each = local.secrets

  app_slug                      = bitrise_app.mobile_app.app_slug
  name                          = each.key
  value                         = each.value.value
  is_protected                  = each.value.protected
  is_exposed_for_pull_requests = each.value.pr_exposed
}

# 5. Configure team access
resource "bitrise_app_roles" "admin_access" {
  app_slug  = bitrise_app.mobile_app.app_slug
  role_name = "admin"
  groups    = ["platform-team"]
}

resource "bitrise_app_roles" "developer_access" {
  app_slug  = bitrise_app.mobile_app.app_slug
  role_name = "developer"
  groups    = ["mobile-dev-team", "qa-team"]
}

# Outputs
output "app_slug" {
  value       = bitrise_app.mobile_app.app_slug
  description = "The slug of the created Bitrise app"
}

output "ssh_public_key" {
  value       = tls_private_key.bitrise_ssh.public_key_openssh
  description = "The SSH public key (for reference)"
}

output "secrets_created" {
  value       = [for s in bitrise_app_secret.secrets : s.id]
  description = "List of created secrets"
}
```

Save this as `main.tf`, create your `bitrise-ios.yml`, set your variables, and run:

```bash
terraform init
terraform apply \
  -var="organization_slug=your-org-slug" \
  -var="environment=production"
```

Happy Terraforming! 🚀
