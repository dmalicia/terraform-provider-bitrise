# bitrise_app_ssh Resource

Manages SSH keys for a Bitrise application. This resource allows you to register SSH keys with your Bitrise app for secure repository access during builds.

## Example Usage

```terraform
# Basic SSH key registration
resource "bitrise_app_ssh" "main" {
  app_slug              = "your-app-slug"
  auth_ssh_private_key  = file("~/.ssh/id_rsa")
  auth_ssh_public_key   = file("~/.ssh/id_rsa.pub")
}

# SSH key with provider registration
resource "bitrise_app_ssh" "with_provider" {
  app_slug                              = bitrise_app.my_app.app_slug
  auth_ssh_private_key                  = var.ssh_private_key
  auth_ssh_public_key                   = var.ssh_public_key
  is_register_key_into_provider_service = true
}

# Using tls_private_key to generate SSH keys
resource "tls_private_key" "bitrise_ssh" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "bitrise_app_ssh" "generated" {
  app_slug              = bitrise_app.my_app.app_slug
  auth_ssh_private_key  = tls_private_key.bitrise_ssh.private_key_pem
  auth_ssh_public_key   = tls_private_key.bitrise_ssh.public_key_openssh
}
```

## Argument Reference

The following arguments are supported:

* `app_slug` - (Required, ForceNew) The slug of the Bitrise app. Changing this forces a new resource to be created.
* `auth_ssh_private_key` - (Required, Sensitive) The private SSH key for authentication. This key will be used to access the repository during builds.
* `auth_ssh_public_key` - (Required) The public SSH key for authentication. This corresponds to the private key and may be registered with the git provider.
* `is_register_key_into_provider_service` - (Optional) If `true`, Bitrise will automatically register the public key with your git provider service. Default: `false`.

## Attribute Reference

This resource does not export any additional attributes beyond the arguments.

## Security Considerations

* **Private Key Storage**: The private SSH key is marked as sensitive and will not appear in Terraform logs or console output.
* **Key Rotation**: To rotate SSH keys, you can update the `auth_ssh_private_key` and `auth_ssh_public_key` attributes. This will trigger an update of the SSH configuration.
* **Provider Registration**: When `is_register_key_into_provider_service` is enabled, ensure your Bitrise account has the necessary permissions to register keys with your git provider.

## Import

SSH key configurations cannot be imported because the private key value cannot be retrieved from the Bitrise API for security reasons.

## API Documentation

This resource uses the following Bitrise API endpoints:

- POST `/v0.1/apps/{app-slug}/register-ssh-key` - Register SSH keys

For more information, see the [Bitrise API documentation](https://api-docs.bitrise.io/).

## Notes

* The private SSH key is temporarily written to a local file during resource creation for processing. This file is managed securely but be aware of this behavior.
* You must have valid SSH key pairs. The public and private keys must match.
* If `is_register_key_into_provider_service` is `true`, Bitrise must have access to your git provider to register the public key.
* SSH keys are essential for private repositories and repositories that require authenticated access.
