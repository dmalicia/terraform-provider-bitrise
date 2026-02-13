# bitrise_app_finish Resource

Completes the registration and setup of a Bitrise application. This resource is used after creating an app with `bitrise_app` to finalize the configuration with project-specific settings like project type, stack, and build configuration.

## Example Usage

```terraform
# Complete app setup after registration
resource "bitrise_app" "my_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/myrepo"
  type              = "git"
  git_repo_slug     = "myrepo"
  git_owner         = "myorg"
  organization_slug = "my-bitrise-org"
}

resource "bitrise_app_finish" "my_app_config" {
  app_slug          = bitrise_app.my_app.app_slug
  project_type      = "ios"
  stack_id          = "osx-xcode-14.0.x"
  config            = file("bitrise.yml")
  mode              = "manual"
  organization_slug = "my-bitrise-org"
}

# Android app with environment variables
resource "bitrise_app_finish" "android_app" {
  app_slug          = bitrise_app.android.app_slug
  project_type      = "android"
  stack_id          = "linux-docker-android-20.04"
  config            = file("bitrise-android.yml")
  mode              = "manual"
  organization_slug = "my-bitrise-org"
  
  envs = {
    GRADLE_BUILD_TOOL_VERSION = "7.4"
    ANDROID_SDK_VERSION       = "33"
  }
}

# React Native app
resource "bitrise_app_finish" "react_native" {
  app_slug          = bitrise_app.rn_app.app_slug
  project_type      = "react-native"
  stack_id          = "osx-xcode-14.2.x"
  config            = file("bitrise-rn.yml")
  mode              = "manual"
  organization_slug = "my-bitrise-org"
  
  envs = {
    NODE_VERSION = "18"
    YARN_VERSION = "1.22"
  }
}
```

## Argument Reference

The following arguments are supported:

* `app_slug` - (Required, ForceNew) The slug of the Bitrise app to configure. This should reference the app created with `bitrise_app`. Changing this forces a new resource to be created.
* `project_type` - (Required) The type of the project (e.g., `ios`, `android`, `react-native`, `flutter`, `xamarin`, `fastlane`, `other`).
* `stack_id` - (Required) The ID of the build stack on which the builds will run. Common stacks include:
  * `osx-xcode-14.0.x` - macOS with Xcode 14.0
  * `osx-xcode-14.2.x` - macOS with Xcode 14.2
  * `linux-docker-android-20.04` - Linux with Android SDK
  * `linux-docker-android-22.04` - Linux with Android SDK (Ubuntu 22.04)
* `config` - (Required) The Bitrise build configuration (bitrise.yml content). This is typically loaded from a file using the `file()` function.
* `mode` - (Required) The configuration mode. Typically set to `manual` for manual configuration management.
* `organization_slug` - (Required) The slug of the organization that owns the app.
* `envs` - (Optional) A map of environment variables to set for the app. These will be available during builds.

## Attribute Reference

This resource does not export any additional attributes beyond the arguments.

## Common Project Types

* `ios` - iOS/macOS applications
* `android` - Android applications  
* `react-native` - React Native applications
* `flutter` - Flutter applications
* `xamarin` - Xamarin applications
* `fastlane` - Fastlane configurations
* `cordova` - Cordova/PhoneGap applications
* `ionic` - Ionic applications
* `other` - Other project types

## Common Stack IDs

### macOS Stacks (for iOS/macOS builds)
* `osx-xcode-14.0.x`
* `osx-xcode-14.1.x`
* `osx-xcode-14.2.x`
* `osx-xcode-15.0.x`

### Linux Stacks (for Android builds)
* `linux-docker-android-20.04`
* `linux-docker-android-22.04`

For the latest available stacks, refer to the [Bitrise stack documentation](https://devcenter.bitrise.io/en/infrastructure/build-stacks.html).

## Import

This resource cannot be imported because the configuration setup is a one-time operation that modifies the app's initial state.

## API Documentation

This resource uses the following Bitrise API endpoint:

- POST `/v0.1/apps/{app-slug}/finish` - Complete app registration

For more information, see the [Bitrise API documentation](https://api-docs.bitrise.io/).

## Notes

* This resource should be used immediately after creating an app with `bitrise_app`.
* The `config` parameter accepts the full content of a `bitrise.yml` file. Use the `file()` function to load it from a file.
* Changing any attribute will trigger an update operation which replaces the configuration.
* The `envs` map allows you to set build-time environment variables that will be available across all workflows.
* Make sure the `stack_id` is compatible with your `project_type`. For example, iOS projects require macOS stacks.
