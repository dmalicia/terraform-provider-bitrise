# terraform {
#   required_providers {
#     bitrise = {
#       source = "hashicorp.com/dmalicia/bitrise"
#     }
#   }
# }

provider "bitrise" {
  token    = "token-here"
  endpoint = "https://api.bitrise.io"
}

resource "bitrise_app" "app_resource" {
  repo                = "github"
  is_public           = false
  organization_slug   = "orgslug"
  repo_url            = "git@github.com:yourorg/fictional-winner.git"
  type                = "git"
  git_repo_slug       = "example-repository"
  git_owner           = "api_demo"
    # Capture the app_slug in the state file or variable
  lifecycle {
    create_before_destroy = true
    ignore_changes        = [app_slug]
  }
}

resource "bitrise_app_finish" "app_name_finish" {
  app_slug          = bitrise_app.app_resource.app_slug
  project_type      = "ios"
  stack_id          = "osx-xcode-13.2.x"
  config            = "default-ios-config"
  mode              = "manual"
  envs = {
    env1 = "val1"
    env2 = "val2"
  }
  organization_slug  = "orgslug"
  depends_on = [bitrise_app_ssh.ssh_resource]
}

resource "bitrise_app_ssh" "ssh_resource" {
  app_slug                            = bitrise_app.app_resource.app_slug
  auth_ssh_private_key                = <<EOT
your-private-key-here
EOT
  auth_ssh_public_key                 = "you-pub-key-here"
  is_register_key_into_provider_service = false
  depends_on = [bitrise_app.app_resource]
}






