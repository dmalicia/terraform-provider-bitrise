# Bitrise App Finish Resource Example

This example demonstrates how to complete the setup of Bitrise applications with project-specific configurations.

## Prerequisites

- A Bitrise organization slug
- Bitrise API token
- Bitrise configuration files (bitrise.yml) for each project type

## Usage

1. Set your variables:
   ```bash
   export TF_VAR_bitrise_token="your-bitrise-api-token"
   export TF_VAR_organization_slug="your-org-slug"
   ```

2. Choose which example to apply:
   ```bash
   # For iOS app
   terraform apply -target=bitrise_app.ios_app -target=bitrise_app_finish.ios_config
   
   # For Android app
   terraform apply -target=bitrise_app.android_app -target=bitrise_app_finish.android_config
   
   # For React Native app
   terraform apply -target=bitrise_app.react_native_app -target=bitrise_app_finish.react_native_config
   
   # For all apps
   terraform apply
   ```

## Examples Included

### 1. iOS App Configuration
Complete iOS app setup with Xcode stack:
- **Project Type**: `ios`
- **Stack**: `osx-xcode-14.2.x`
- **Environment Variables**: Xcode version, Team ID
- **Config**: iOS-specific bitrise.yml with certificate installation, CocoaPods, and archiving

### 2. Android App Configuration
Android app with Gradle build:
- **Project Type**: `android`
- **Stack**: `linux-docker-android-22.04`
- **Environment Variables**: Gradle version, Android SDK version, Java version
- **Config**: Android-specific bitrise.yml with Gradle build steps

### 3. React Native App Configuration
React Native app targeting iOS:
- **Project Type**: `react-native`
- **Stack**: `osx-xcode-14.2.x`
- **Environment Variables**: Node version, Yarn version, RN version
- **Config**: React Native bitrise.yml with npm install, tests, and linting

### 4. Flutter App Configuration
Flutter app for cross-platform development:
- **Project Type**: `flutter`
- **Stack**: `linux-docker-android-22.04`
- **Environment Variables**: Flutter and Dart versions
- **Config**: Flutter bitrise.yml with analyzer, tests, and build steps

### 5. Minimal Configuration
Simple configuration for other project types:
- **Project Type**: `other`
- **Stack**: `linux-docker-android-22.04`
- **Config**: Basic bitrise.yml with git clone and custom script step

## Bitrise Configuration Files

The example includes sample `bitrise.yml` files for each project type:

- `bitrise-ios.yml` - iOS project configuration
- `bitrise-android.yml` - Android project configuration
- `bitrise-rn.yml` - React Native project configuration
- `bitrise-flutter.yml` - Flutter project configuration
- `bitrise-minimal.yml` - Minimal generic configuration

### Customizing bitrise.yml

Each bitrise.yml file contains:
- **workflows**: Build workflows and their steps
- **app.envs**: Application-level environment variables
- **steps**: Individual build steps (git clone, build, test, deploy, etc.)

Modify these files to match your project's specific needs.

## Common Stack IDs

### macOS Stacks (for iOS/macOS builds)
- `osx-xcode-14.0.x` - Xcode 14.0
- `osx-xcode-14.1.x` - Xcode 14.1
- `osx-xcode-14.2.x` - Xcode 14.2
- `osx-xcode-15.0.x` - Xcode 15.0

### Linux Stacks (for Android/cross-platform builds)
- `linux-docker-android-20.04` - Ubuntu 20.04 with Android SDK
- `linux-docker-android-22.04` - Ubuntu 22.04 with Android SDK

## Project Types

- `ios` - iOS/macOS applications
- `android` - Android applications
- `react-native` - React Native applications
- `flutter` - Flutter applications
- `xamarin` - Xamarin applications
- `cordova` - Cordova/PhoneGap applications
- `ionic` - Ionic applications
- `fastlane` - Fastlane configurations
- `other` - Other project types

## Environment Variables

The `envs` parameter allows you to set build-time environment variables:

```terraform
envs = {
  NODE_VERSION = "18"
  CUSTOM_VAR   = "value"
}
```

These variables are available across all workflows in your bitrise.yml.

## Complete Workflow

A typical complete setup includes:

1. **Create App** (`bitrise_app`)
2. **Configure SSH Keys** (`bitrise_app_ssh`) - if needed for private repos
3. **Finish Setup** (`bitrise_app_finish`) - configure project type and build settings
4. **Add Secrets** (`bitrise_app_secret`) - add sensitive environment variables
5. **Configure Roles** (`bitrise_app_roles`) - manage team access

## Outputs

- `ios_app_slug` - The slug for the iOS app
- `android_app_slug` - The slug for the Android app

## Notes

- The `config` parameter must contain valid bitrise.yml content
- Stack ID must be compatible with your project type (e.g., iOS requires macOS stacks)
- The `mode` is typically set to `manual` for Terraform-managed configurations
- Environment variables set in `envs` supplement those in the bitrise.yml file
- Make sure the bitrise.yml file paths are correct relative to the Terraform configuration

## Troubleshooting

### Invalid Stack ID
If you receive an error about an invalid stack, check the [Bitrise stack documentation](https://devcenter.bitrise.io/en/infrastructure/build-stacks.html) for the latest available stacks.

### Invalid bitrise.yml
Validate your bitrise.yml file syntax before applying:
```bash
bitrise validate -c bitrise-ios.yml
```

### Permission Errors
Ensure your Bitrise API token has the necessary permissions to create and configure apps in the specified organization.
