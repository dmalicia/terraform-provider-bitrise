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

variable "app_slug" {
  description = "Bitrise Application Slug"
  type        = string
}

# Example 1: Basic secret
resource "bitrise_app_secret" "api_key" {
  app_slug = var.app_slug
  name     = "API_KEY"
  value    = "my-api-key-value"
}

# Example 2: Protected secret (value cannot be retrieved)
resource "bitrise_app_secret" "production_password" {
  app_slug     = var.app_slug
  name         = "PROD_PASSWORD"
  value        = "super-secret-password"
  is_protected = true
}

# Example 3: Secret for pull requests
resource "bitrise_app_secret" "github_token" {
  app_slug                      = var.app_slug
  name                          = "GITHUB_PR_TOKEN"
  value                         = var.github_token
  is_exposed_for_pull_requests = true
}

# Example 4: Secret with all options
resource "bitrise_app_secret" "database_url" {
  app_slug                      = var.app_slug
  name                          = "DATABASE_URL"
  value                         = "postgresql://user:pass@host:5432/db"
  is_protected                  = true
  is_exposed_for_pull_requests = false
  expand_in_step_inputs        = true
}

# Example 5: Multiple secrets using for_each
variable "app_secrets" {
  description = "Map of secrets to create"
  type = map(object({
    value                         = string
    is_protected                  = optional(bool, false)
    is_exposed_for_pull_requests = optional(bool, false)
    expand_in_step_inputs        = optional(bool, true)
  }))
  sensitive = true
  default = {
    "AWS_ACCESS_KEY_ID" = {
      value        = "AKIAIOSFODNN7EXAMPLE"
      is_protected = true
    }
    "AWS_SECRET_ACCESS_KEY" = {
      value        = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
      is_protected = true
    }
    "SLACK_WEBHOOK_URL" = {
      value                         = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXX"
      is_protected                  = false
      is_exposed_for_pull_requests = false
    }
  }
}

resource "bitrise_app_secret" "app_secrets" {
  for_each = var.app_secrets

  app_slug                      = var.app_slug
  name                          = each.key
  value                         = each.value.value
  is_protected                  = each.value.is_protected
  is_exposed_for_pull_requests = each.value.is_exposed_for_pull_requests
  expand_in_step_inputs        = each.value.expand_in_step_inputs
}

# Outputs
output "secret_ids" {
  description = "IDs of created secrets"
  value       = { for k, v in bitrise_app_secret.app_secrets : k => v.id }
}
