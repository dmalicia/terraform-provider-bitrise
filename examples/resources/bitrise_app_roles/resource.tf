resource "bitrise_app_roles" "admin_groups" {
  app_slug  = "your-app-slug-here"
  role_name = "admin"
  groups    = ["admin-group-slug", "another-admin-group"]
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
