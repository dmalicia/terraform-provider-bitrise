data "bitrise_org_groups" "example" {
  org_slug = "my-organization"
}

output "all_groups" {
  value = data.bitrise_org_groups.example.groups
}

output "group_names" {
  value = [for group in data.bitrise_org_groups.example.groups : group.name]
}
