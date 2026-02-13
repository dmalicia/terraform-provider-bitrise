## 0.1.0 (Unreleased)

FEATURES:

**Resources:**
* **New Resource:** `bitrise_app` - Register and manage Bitrise applications from various git providers (GitHub, GitLab, Bitbucket)
* **New Resource:** `bitrise_app_ssh` - Manage SSH keys for Bitrise applications with secure key handling and optional provider registration
* **New Resource:** `bitrise_app_finish` - Complete application setup with project type configuration, stack selection, and build configuration
* **New Resource:** `bitrise_app_secret` - Manage application secrets (environment variables) with support for protected secrets, pull request exposure, and variable expansion settings
* **New Resource:** `bitrise_app_bitrise_yml` - Manage bitrise.yml workflow configuration files with support for inline YAML, file templates, and dynamic template variables
* **New Resource:** `bitrise_app_roles` - Manage team role assignments and access control for applications

**Data Sources:**
* **New Data Source:** `bitrise_app_roles` - Retrieve role assignments for an application
* **New Data Source:** `bitrise_org_groups` - Retrieve organization groups for access management

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
* Added data source documentation for 2 data sources
* Added IMPLEMENTATION_SUMMARY.md with complete provider overview
* Updated README.md with full resource listing and complete workflow examples
* Added detailed examples with README files for each resource
* Included sample bitrise.yml templates for multiple project types