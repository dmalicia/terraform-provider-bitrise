# Bitrise Available Stacks Data Source Example

This example demonstrates how to use the `bitrise_available_stacks` data source to retrieve all available stack IDs from Bitrise.

## What are Stacks?

Stacks are the build environments (operating system and pre-installed software) that Bitrise provides for running your builds. Each stack has a unique ID that identifies the specific OS version, Xcode version (for macOS stacks), or other configuration.

## Usage

```bash
terraform init
terraform plan
terraform apply
```

This will output:
- All available stack IDs
- Filtered lists of macOS, Linux, and Xcode stacks

## Common Use Cases

1. **List all stacks**: Get a complete list of available build environments
2. **Filter by OS**: Find all macOS or Linux stacks
3. **Filter by software**: Find stacks with specific Xcode versions
4. **Validation**: Check if a specific stack ID exists before using it in your configuration
