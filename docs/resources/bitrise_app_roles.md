# bitrise_app_roles Resource

Manages the groups assigned to a specific role type for a Bitrise application. This resource replaces all groups for the specified role with the provided list.

## Example Usage

```terraform
resource "bitrise_app_roles" "admin_groups" {
  app_slug  = "your-app-slug-here"
  role_name = "admin"
  groups    = ["group-slug-1", "group-slug-2"]
}

resource "bitrise_app_roles" "manager_groups" {
  app_slug  = "your-app-slug-here"
  role_name = "manager"
  groups    = ["developers-group"]
}

resource "bitrise_app_roles" "member_groups" {
  app_slug  = "your-app-slug-here"
  role_name = "member"
  groups    = ["qa-team-group", "testers-group"]
}
```

## Argument Reference

* `app_slug` - (Required) The slug of the Bitrise app. Changing this forces a new resource to be created.
* `role_name` - (Required) The role type to manage. Changing this forces a new resource to be created. Supported values:
  * `admin` - Administrative access
  * `manager` - Manager access (equivalent to developer)
  * `member` - Member access (equivalent to tester/qa)
  * `platform_engineer` - Platform engineer access
* `groups` - (Required) List of group slugs to assign to this role. This replaces all existing groups for this role.

## Attribute Reference

* `id` - Resource identifier in the format `app_slug/role_name`.

## Import

App roles can be imported using the app slug and role name separated by a forward slash:

```bash
terraform import bitrise_app_roles.admin_groups your-app-slug-here/admin
terraform import bitrise_app_roles.manager_groups your-app-slug-here/manager
```

## Notes

* This resource manages the **complete** list of roles. Any roles not specified in the configuration will be removed.
* Deleting this resource will clear all roles from the application (set to empty list).
* The role values should match those accepted by the Bitrise API for your organization's plan.
