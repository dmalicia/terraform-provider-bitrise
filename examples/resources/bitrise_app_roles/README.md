# Bitrise App Roles Example

This example demonstrates how to manage groups assigned to specific role types for a Bitrise application.

## Usage

1. Update the `app_slug` value with your actual Bitrise app slug
2. Update the `role_name` with the role type you want to manage (admin, manager, member, or platform_engineer)
3. Modify the `groups` list with the group slugs you want to assign to this role
4. Run `terraform init` to initialize the provider
5. Run `terraform plan` to see what changes will be made
6. Run `terraform apply` to apply the role group configuration

## Notes

- This resource replaces **all groups** for the specified role, so make sure to include all groups you want assigned
- Each resource manages one role type - create multiple resources to manage different role types
- Available role types:
  - `admin` - Administrative access
  - `manager` - Manager access (equivalent to developer)
  - `member` - Member access (equivalent to tester/qa)
  - `platform_engineer` - Platform engineer access
- Group slugs can be found in your Bitrise organization's group management section
