# bitrise_available_stacks (Data Source)

Retrieves the list of all available stack IDs from Bitrise.

This data source allows you to fetch all stack IDs that are currently available on Bitrise. Stacks are the build environments (OS and software versions) that your builds can run on.

## Example Usage

```terraform
data "bitrise_available_stacks" "all" {
}

output "all_stacks" {
  value = data.bitrise_available_stacks.all.stack_keys
}

# Filter to find specific stacks
locals {
  xcode_stacks = [
    for stack in data.bitrise_available_stacks.all.stack_keys :
    stack if can(regex("^osx-xcode", stack))
  ]
}

output "xcode_stacks" {
  value = local.xcode_stacks
}
```

## Schema

### Read-Only

- `id` (String) Data source identifier
- `stack_keys` (List of String) List of all available stack IDs (e.g., "osx-xcode-16.2.x", "ubuntu-noble-24.04-bitrise-2025-android")

## Notes

- This data source does not require any input parameters
- The list of stacks is returned as an array of stack IDs
- Stack IDs can be used when configuring workflows or triggering builds
