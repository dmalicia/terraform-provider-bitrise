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

# Example 1: Basic bitrise.yml from inline content
resource "bitrise_app_bitrise_yml" "basic" {
  app_slug    = var.app_slug
  yml_content = <<-EOT
    format_version: 11
    default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
    
    workflows:
      primary:
        steps:
        - activate-ssh-key@4:
            run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
        - git-clone@8: {}
        - script@1:
            title: Do anything with Script step
            inputs:
            - content: |
                #!/usr/bin/env bash
                set -ex
                echo "Hello World!"
  EOT
}

# Example 2: Using file() function to load from a template
resource "bitrise_app_bitrise_yml" "from_file" {
  app_slug    = var.app_slug
  yml_content = file("${path.module}/bitrise-template.yml")
}

# Example 3: Using templatefile() function with variables
resource "bitrise_app_bitrise_yml" "from_template" {
  app_slug = var.app_slug
  yml_content = templatefile("${path.module}/bitrise-template.yml", {
    app_name        = "my-app"
    slack_webhook   = var.slack_webhook
    deploy_workflow = "deploy-production"
  })
}
