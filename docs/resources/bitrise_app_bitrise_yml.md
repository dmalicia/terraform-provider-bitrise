# bitrise_app_bitrise_yml Resource

Manages the `bitrise.yml` configuration file for a Bitrise application. This resource allows you to programmatically create and update the workflow configuration for your Bitrise apps using Terraform.

## Example Usage

```terraform
# Basic inline YAML configuration
resource "bitrise_app_bitrise_yml" "basic" {
  app_slug    = "your-app-slug"
  yml_content = <<-EOT
    format_version: 11
    default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git
    
    workflows:
      primary:
        steps:
        - activate-ssh-key@4:
            run_if: '{{getenv "SSH_RSA_PRIVATE_KEY" | ne ""}}'
        - git-clone@8: {}
        - script@1:
            title: Do anything with Script step
            inputs:
            - content: |
                #!/usr/bin/env bash
                set -ex
                echo "Hello World!"
  EOT
}

# Load bitrise.yml from a file
resource "bitrise_app_bitrise_yml" "from_file" {
  app_slug    = "your-app-slug"
  yml_content = file("${path.module}/bitrise-template.yml")
}

# Use templatefile() for dynamic configurations
resource "bitrise_app_bitrise_yml" "from_template" {
  app_slug = "your-app-slug"
  yml_content = templatefile("${path.module}/bitrise-template.yml", {
    app_name        = "my-app"
    slack_webhook   = var.slack_webhook
    deploy_workflow = "deploy-production"
  })
}
```

## Argument Reference

The following arguments are supported:

* `app_slug` - (Required, ForceNew) The slug of the Bitrise app. Changing this forces a new resource to be created.
* `yml_content` - (Required) The content of the bitrise.yml file. This should be a valid YAML configuration for Bitrise workflows. You can use inline YAML, the `file()` function, or the `templatefile()` function to provide the content.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the resource (same as `app_slug`).

## Using File Templates

### Basic File Loading

You can store your bitrise.yml configuration in a separate file and load it using Terraform's `file()` function:

```terraform
resource "bitrise_app_bitrise_yml" "app" {
  app_slug    = var.app_slug
  yml_content = file("${path.module}/bitrise.yml")
}
```

### Dynamic Templates

For more complex scenarios where you need to inject variables into your configuration, use the `templatefile()` function:

**Terraform configuration:**
```terraform
resource "bitrise_app_bitrise_yml" "app" {
  app_slug = var.app_slug
  yml_content = templatefile("${path.module}/bitrise-template.yml", {
    environment     = var.environment
    slack_webhook   = var.slack_webhook
    docker_image    = var.docker_image
  })
}
```

**Template file (bitrise-template.yml):**
```yaml
format_version: 11
default_step_lib_source: https://github.com/bitrise-io/bitrise-steplib.git

workflows:
  primary:
    steps:
    - script@1:
        title: Build for ${environment}
        inputs:
        - content: |
            docker build -t ${docker_image} .
    - slack@3:
        inputs:
        - webhook_url: ${slack_webhook}
```

## Import

Bitrise.yml configurations can be imported using the app slug:

```shell
terraform import bitrise_app_bitrise_yml.app your-app-slug
```

When importing, Terraform will read the current bitrise.yml configuration from Bitrise and store it in the state.

## Important Notes

### Deletion Behavior

When a `bitrise_app_bitrise_yml` resource is destroyed (via `terraform destroy` or removing it from configuration), **the bitrise.yml file remains in your Bitrise app**. The resource is only removed from Terraform state. This is a safety measure to prevent accidental deletion of workflow configurations.

If you need to completely remove or reset the configuration, you must do so manually through the Bitrise web interface or API.

### YAML Formatting

Ensure your YAML content is properly formatted and valid according to Bitrise's requirements. Invalid YAML will cause the API request to fail. You can validate your configuration using the [Bitrise CLI](https://www.bitrise.io/cli) locally before applying.

### Format Version

Always specify a `format_version` in your bitrise.yml. Bitrise recommends using the latest format version (currently 11). See the [Configuration Format Version](https://devcenter.bitrise.io/en/references/bitrise-yml-reference/configuration-format-version.html) documentation for details.

## API Documentation

This resource uses the following Bitrise API endpoints:

- POST `/v0.1/apps/{app-slug}/bitrise.yml` - Create or update bitrise.yml
- GET `/v0.1/apps/{app-slug}/bitrise.yml` - Read bitrise.yml

For more information, see the [Bitrise API documentation](https://api-docs.bitrise.io/).

## Best Practices

1. **Version Control**: Store your bitrise.yml template files in version control alongside your Terraform configuration
2. **Validation**: Use the Bitrise CLI to validate your YAML before applying: `bitrise validate -c bitrise.yml`
3. **Modularization**: For complex workflows, consider using templatefile() with environment-specific variable files
4. **Testing**: Test changes in a development app before applying to production
5. **Documentation**: Add comments in your YAML to explain complex workflow logic

## See Also

- [Bitrise.yml Reference](https://devcenter.bitrise.io/en/references/bitrise-yml-reference.html)
- [Bitrise Steps](https://www.bitrise.io/integrations/steps)
- [Workflow Editor](https://devcenter.bitrise.io/en/steps-and-workflows/introduction-to-workflows.html)
