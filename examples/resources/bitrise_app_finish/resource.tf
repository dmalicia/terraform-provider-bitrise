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

variable "bitrise_token" {
  description = "Bitrise API token"
  type        = string
  sensitive   = true
}

variable "organization_slug" {
  description = "Bitrise organization slug"
  type        = string
}

# Example 1: Complete iOS app setup
resource "bitrise_app" "ios_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/ios-app"
  type              = "git"
  git_repo_slug     = "ios-app"
  git_owner         = "myorg"
  organization_slug = var.organization_slug
}

resource "bitrise_app_finish" "ios_config" {
  app_slug          = bitrise_app.ios_app.app_slug
  project_type      = "ios"
  stack_id          = "osx-xcode-14.2.x"
  config            = file("${path.module}/bitrise-ios.yml")
  mode              = "manual"
  organization_slug = var.organization_slug
  
  envs = {
    FASTLANE_XCODE_VERSION = "14.2"
    TEAM_ID                = "YOUR_TEAM_ID"
  }
}

# Example 2: Android app setup
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
  config            = file("${path.module}/bitrise-android.yml")
  mode              = "manual"
  organization_slug = var.organization_slug
  
  envs = {
    GRADLE_BUILD_TOOL_VERSION = "7.4"
    ANDROID_SDK_VERSION       = "33"
    JAVA_VERSION              = "11"
  }
}

# Example 3: React Native app setup
resource "bitrise_app" "react_native_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/rn-app"
  type              = "git"
  git_repo_slug     = "rn-app"
  git_owner         = "myorg"
  organization_slug = var.organization_slug
}

resource "bitrise_app_finish" "react_native_config" {
  app_slug          = bitrise_app.react_native_app.app_slug
  project_type      = "react-native"
  stack_id          = "osx-xcode-14.2.x"
  config            = file("${path.module}/bitrise-rn.yml")
  mode              = "manual"
  organization_slug = var.organization_slug
  
  envs = {
    NODE_VERSION = "18"
    YARN_VERSION = "1.22"
    RN_VERSION   = "0.71"
  }
}

# Example 4: Flutter app setup
resource "bitrise_app" "flutter_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/flutter-app"
  type              = "git"
  git_repo_slug     = "flutter-app"
  git_owner         = "myorg"
  organization_slug = var.organization_slug
}

resource "bitrise_app_finish" "flutter_config" {
  app_slug          = bitrise_app.flutter_app.app_slug
  project_type      = "flutter"
  stack_id          = "linux-docker-android-22.04"
  config            = file("${path.module}/bitrise-flutter.yml")
  mode              = "manual"
  organization_slug = var.organization_slug
  
  envs = {
    FLUTTER_VERSION = "3.10.0"
    DART_VERSION    = "3.0.0"
  }
}

# Example 5: Minimal configuration
resource "bitrise_app" "minimal_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/minimal-app"
  type              = "git"
  git_repo_slug     = "minimal-app"
  git_owner         = "myorg"
  organization_slug = var.organization_slug
}

resource "bitrise_app_finish" "minimal_config" {
  app_slug          = bitrise_app.minimal_app.app_slug
  project_type      = "other"
  stack_id          = "linux-docker-android-22.04"
  config            = file("${path.module}/bitrise-minimal.yml")
  mode              = "manual"
  organization_slug = var.organization_slug
}

# Outputs
output "ios_app_slug" {
  value       = bitrise_app.ios_app.app_slug
  description = "iOS app slug"
}

output "android_app_slug" {
  value       = bitrise_app.android_app.app_slug
  description = "Android app slug"
}
