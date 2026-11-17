terraform {
  required_providers {
    bitrise = {
      source = "dmalicia/bitrise"
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

variable "org_slug" {
  description = "Bitrise workspace (organization) slug"
  type        = string
}

variable "github_token" {
  type      = string
  sensitive = true
}

resource "bitrise_org_secret" "github_token" {
  org_slug                     = var.org_slug
  name                         = "GITHUB_TOKEN"
  value                        = var.github_token
  is_protected                 = true
  is_exposed_for_pull_requests = true
  expand_in_step_inputs        = false
}

resource "bitrise_org_secret" "github_actor" {
  org_slug                     = var.org_slug
  name                         = "GITHUB_ACTOR"
  value                        = "my-bot"
  is_exposed_for_pull_requests = true
  expand_in_step_inputs        = false
}
