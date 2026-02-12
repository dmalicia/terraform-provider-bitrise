# bitrise_app_roles Data Source

Retrieves the list of groups assigned to a specific role type for a Bitrise application.

## Example Usage

```terraform
data "bitrise_app_roles" "admin_groups" {
  app_slug  = "your-app-slug-here"
  role_name = "admin"
}

output "admin_groups" {
  value = data.bitrise_app_roles.admin_groups.groups
}

data "bitrise_app_roles" "manager_groups" {
  app_slug  = "your-app-slug-here"
  role_name = "manager"
}

output "manager_groups" {
  value = data.bitrise_app_roles.manager_groups.groups
}
```

## Argument Reference

* `app_slug` - (Required) The slug of the Bitrise app.
* `role_name` - (Required) The role type to query. Supported values:
  * `admin` - Administrative access
  * `manager` - Manager access (equivalent to developer)
  * `member` - Member access (equivalent to tester/qa)
  * `platform_engineer` - Platform engineer access

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - Data source identifier in the format `app_slug/role_name`.
* `groups` - List of group slugs assigned to this role.
