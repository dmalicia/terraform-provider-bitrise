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
