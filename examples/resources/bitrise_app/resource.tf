terraform {
  required_providers {
    bitrise = {
      source = "local/provider/bitrise"
    }
  }
}

provider "bitrise" {
  endpoint = "https://api.bitrise.io"
  token    = var.bitrise_token
}

# Register a GitHub application
resource "bitrise_app" "my_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/myrepo"
  type              = "git"
  git_repo_slug     = "myrepo"
  git_owner         = "myorg"
  organization_slug = "my-bitrise-org"
  is_public         = false
}

# Register a public open-source application
resource "bitrise_app" "public_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/public-repo"
  type              = "git"
  git_repo_slug     = "public-repo"
  git_owner         = "myorg"
  organization_slug = "my-bitrise-org"
  is_public         = true
}

# Output the app slug for use in other resources
output "app_slug" {
  value       = bitrise_app.my_app.app_slug
  description = "The slug of the created Bitrise app"
}

output "app_id" {
  value       = bitrise_app.my_app.id
  description = "The ID of the created Bitrise app"
}
