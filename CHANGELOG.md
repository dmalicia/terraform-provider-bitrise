## 2.3.0 (Unreleased)

BUG FIXES:

* `bitrise_app`, `bitrise_app_finish`, `bitrise_app_ssh`: Report API and network errors on create instead of failing with a generic "Missing Resource State After Create" error
* `bitrise_app`: Send updates to the app's own endpoint; the slug was unknown during update, so every in-place change failed with a 404
* `bitrise_app`: Import now sets `app_slug`, so the imported app is read from the API and a missing app is reported instead of importing an empty resource
* `bitrise_app`: Omitting `is_public` no longer causes a perpetual diff
* `bitrise_app_ssh`: Omitting `is_register_key_into_provider_service` no longer fails with a value conversion error
* `bitrise_app_finish`, `bitrise_app_ssh`: Fix import, which failed with a value conversion or state write error; both are imported by app slug
* `bitrise_app_ssh`: Stop writing the private key to a `testtfkey` file in the working directory and to debug logs
* `bitrise_app_ssh`: Mark `auth_ssh_private_key` as sensitive so the private key is no longer printed in plan and apply output (the documentation already said it was)
* `bitrise_app_secret`: Send `false` flags on create; `expand_in_step_inputs = false` was dropped, so the API default applied and every plan showed a diff
* `data.bitrise_available_stacks`: Return `stack_keys` sorted instead of in random order
* Time out API requests after 60 seconds instead of waiting indefinitely for an unresponsive API
* Reject API responses larger than 1 MiB and truncate error response bodies in diagnostics to 4 KiB instead of reading them without limit

NOTES:

* Build with Go 1.26.11 (Go 1.24 is no longer supported) and update dependencies, including `golang.org/x/crypto`, `golang.org/x/net`, `golang.org/x/text`, `google.golang.org/grpc` and `github.com/cloudflare/circl`, to fix known vulnerabilities

## 2.2.0 (November 24, 2026)

FEATURES:

* **New Resource:** `bitrise_org_secret` - Manage workspace (organization) secrets shared by all apps in a Bitrise workspace

## 2.1.0 (February 13, 2026)

FEATURES:

**Resources:**
* **New Resource:** `bitrise_app_secret` - Manage application secrets (environment variables) with support for protected secrets, pull request exposure, and variable expansion settings
* **New Resource:** `bitrise_app_bitrise_yml` - Manage bitrise.yml workflow configuration files with support for inline YAML, file templates, and dynamic template variables
* **New Resource:** `bitrise_app_roles` - Manage team role assignments and access control for applications

**Data Sources:**
* **New Data Source:** `bitrise_app_roles` - Retrieve role assignments for an application
* **New Data Source:** `bitrise_org_groups` - Retrieve organization groups for access management
* **New Data Source:** `bitrise_available_stacks` - Retrieve the build stacks available to an application

IMPROVEMENTS:

* provider: Complete provider implementation with all core Bitrise app management resources
* provider: Security-first design with sensitive data handling and protected secret support
* docs: Comprehensive documentation for all resources with detailed examples and API references
* examples: Complete workflow examples demonstrating app registration, SSH configuration, and setup finalization
* examples: Production-ready examples for iOS, Android, React Native, and Flutter applications
* examples: Security best practices for secret management and SSH key handling
* examples: Multiple bitrise.yml templates for different project types
* testing: Comprehensive testing guide with manual test scenarios for all resources

DOCUMENTATION:

* Added complete resource documentation for all 6 resources
* Added data source documentation for 3 data sources
* Added IMPLEMENTATION_SUMMARY.md with complete provider overview
* Updated README.md with full resource listing and complete workflow examples
* Added detailed examples with README files for each resource
* Included sample bitrise.yml templates for multiple project types

## 2.0.0 (January 5, 2026)

IMPROVEMENTS:

* provider: Upgrade to Go 1.24 and current `terraform-plugin-framework` and `terraform-plugin-log` releases
* `bitrise_app`, `bitrise_app_finish`: Structured logging with `tflog` instead of printing to stdout
* `bitrise_app_finish`: Read, update, delete and import handle state and report errors through diagnostics

## 1.0.0 (September 4, 2023)

FEATURES:

* **New Resource:** `bitrise_app` - Register and manage Bitrise applications from various git providers (GitHub, GitLab, Bitbucket)
* **New Resource:** `bitrise_app_ssh` - Manage SSH keys for Bitrise applications with secure key handling and optional provider registration
* **New Resource:** `bitrise_app_finish` - Complete application setup with project type configuration, stack selection, and build configuration
