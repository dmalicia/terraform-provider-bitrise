# Bitrise App Bitrise.yml Resource Examples

This directory contains examples for managing Bitrise application workflow configurations using the `bitrise_app_bitrise_yml` resource.

## Basic Usage

The simplest form uses inline YAML content:

```terraform
resource "bitrise_app_bitrise_yml" "basic" {
  app_slug    = "your-app-slug"
  yml_content = <<-EOT
    format_version: 11
    default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
    
    workflows:
      primary:
        steps:
        - git-clone@8: {}
        - script@1:
            title: Hello World
            inputs:
            - content: echo "Hello World!"
  EOT
}
```

## Using File Templates

For better organization, you can store your bitrise.yml in a separate file:

```terraform
resource "bitrise_app_bitrise_yml" "from_file" {
  app_slug    = var.app_slug
  yml_content = file("${path.module}/bitrise-template.yml")
}
```

## Using Template Files with Variables

For dynamic configurations, use `templatefile()`:

```terraform
resource "bitrise_app_bitrise_yml" "from_template" {
  app_slug = var.app_slug
  yml_content = templatefile("${path.module}/bitrise-template.yml", {
    app_name      = "my-app"
    slack_webhook = var.slack_webhook
  })
}
```

Then in your `bitrise-template.yml`:

```yaml
format_version: 11
default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git

workflows:
  primary:
    steps:
    - git-clone@8: {}
    - slack@3:
        inputs:
        - webhook_url: ${slack_webhook}
        - text: "Build started for ${app_name}"
```

## Importing Existing Configuration

You can import existing bitrise.yml configurations:

```bash
terraform import bitrise_app_bitrise_yml.example <app-slug>
```
