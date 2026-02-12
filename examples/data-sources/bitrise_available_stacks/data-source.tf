data "bitrise_available_stacks" "all" {
}

output "all_stacks" {
  description = "All available Bitrise stack IDs"
  value       = data.bitrise_available_stacks.all.stack_keys
}

# Example: Filter stacks by OS type
locals {
  # Get all macOS stacks
  osx_stacks = [
    for stack in data.bitrise_available_stacks.all.stack_keys :
    stack if can(regex("^osx-", stack))
  ]

  # Get all Linux stacks
  linux_stacks = [
    for stack in data.bitrise_available_stacks.all.stack_keys :
    stack if can(regex("^(linux-|ubuntu-)", stack))
  ]

  # Get all Xcode stacks
  xcode_stacks = [
    for stack in data.bitrise_available_stacks.all.stack_keys :
    stack if can(regex("xcode", stack))
  ]
}

output "osx_stacks" {
  description = "All macOS stacks"
  value       = local.osx_stacks
}

output "linux_stacks" {
  description = "All Linux stacks"
  value       = local.linux_stacks
}

output "xcode_stacks" {
  description = "All Xcode stacks"
  value       = local.xcode_stacks
}
