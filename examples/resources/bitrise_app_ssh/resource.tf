terraform {
  required_providers {
    bitrise = {
      source = "local/provider/bitrise"
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
  description = "Bitrise API token"
  type        = string
  sensitive   = true
}

variable "app_slug" {
  description = "The Bitrise app slug"
  type        = string
}

# Example 1: Using existing SSH keys from files
resource "bitrise_app_ssh" "from_files" {
  app_slug             = var.app_slug
  auth_ssh_private_key = file("~/.ssh/id_rsa")
  auth_ssh_public_key  = file("~/.ssh/id_rsa.pub")
}

# Example 2: Generate new SSH keys using tls_private_key
resource "tls_private_key" "bitrise_ssh" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "bitrise_app_ssh" "generated" {
  app_slug             = var.app_slug
  auth_ssh_private_key = tls_private_key.bitrise_ssh.private_key_pem
  auth_ssh_public_key  = tls_private_key.bitrise_ssh.public_key_openssh
}

# Example 3: Register public key with git provider
resource "bitrise_app_ssh" "with_provider_registration" {
  app_slug                              = var.app_slug
  auth_ssh_private_key                  = tls_private_key.bitrise_ssh.private_key_pem
  auth_ssh_public_key                   = tls_private_key.bitrise_ssh.public_key_openssh
  is_register_key_into_provider_service = true
}

# Example 4: Complete workflow with app creation
resource "bitrise_app" "my_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/myrepo"
  type              = "git"
  git_repo_slug     = "myrepo"
  git_owner         = "myorg"
  organization_slug = "my-bitrise-org"
}

resource "tls_private_key" "app_ssh" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "bitrise_app_ssh" "app_ssh_config" {
  app_slug             = bitrise_app.my_app.app_slug
  auth_ssh_private_key = tls_private_key.app_ssh.private_key_pem
  auth_ssh_public_key  = tls_private_key.app_ssh.public_key_openssh
}

# Output the public key (safe to output, unlike private key)
output "ssh_public_key" {
  value       = tls_private_key.bitrise_ssh.public_key_openssh
  description = "The generated SSH public key"
}
