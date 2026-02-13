# Bitrise App SSH Resource Example

This example demonstrates how to configure SSH keys for a Bitrise application.

## Prerequisites

- A Bitrise app slug (from `bitrise_app` resource or existing app)
- SSH key pair (can be existing or generated with Terraform)
- Bitrise API token

## Usage

### Using Existing SSH Keys

1. Set your variables:
   ```bash
   export TF_VAR_bitrise_token="your-bitrise-api-token"
   export TF_VAR_app_slug="your-app-slug"
   ```

2. Ensure your SSH keys exist at `~/.ssh/id_rsa` and `~/.ssh/id_rsa.pub`

3. Apply the configuration:
   ```bash
   terraform init
   terraform apply -target=bitrise_app_ssh.from_files
   ```

### Generating New SSH Keys

The example includes a `tls_private_key` resource that generates a new RSA-4096 SSH key pair:

```bash
terraform apply -target=tls_private_key.bitrise_ssh -target=bitrise_app_ssh.generated
```

## Examples Included

### 1. From Existing Files
Loads SSH keys from your local filesystem:
```terraform
resource "bitrise_app_ssh" "from_files" {
  app_slug             = var.app_slug
  auth_ssh_private_key = file("~/.ssh/id_rsa")
  auth_ssh_public_key  = file("~/.ssh/id_rsa.pub")
}
```

### 2. Generated Keys
Uses the TLS provider to generate new SSH keys:
```terraform
resource "tls_private_key" "bitrise_ssh" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "bitrise_app_ssh" "generated" {
  app_slug             = var.app_slug
  auth_ssh_private_key = tls_private_key.bitrise_ssh.private_key_pem
  auth_ssh_public_key  = tls_private_key.bitrise_ssh.public_key_openssh
}
```

### 3. With Provider Registration
Automatically registers the public key with your git provider:
```terraform
resource "bitrise_app_ssh" "with_provider_registration" {
  app_slug                              = var.app_slug
  is_register_key_into_provider_service = true
  # ... other attributes
}
```

### 4. Complete Workflow
Shows the full flow from app creation to SSH configuration:
```terraform
resource "bitrise_app" "my_app" { ... }
resource "tls_private_key" "app_ssh" { ... }
resource "bitrise_app_ssh" "app_ssh_config" { ... }
```

## Security Best Practices

1. **Never commit private keys**: Always use variables or file references
2. **Use generated keys**: Generate dedicated SSH keys for Bitrise rather than reusing personal keys
3. **Rotate regularly**: Periodically generate new SSH keys and update the configuration
4. **Limit scope**: Use deploy keys with read-only access when possible

## When to Use `is_register_key_into_provider_service`

Set this to `true` when:
- You want Bitrise to automatically add the public key to your Git provider (GitHub, GitLab, etc.)
- Your Bitrise account has OAuth access to the Git provider
- You want to minimize manual configuration

Set this to `false` when:
- You prefer to manually add the public key to your Git provider
- Your Bitrise account doesn't have the necessary OAuth permissions
- You're using a custom or on-premise Git server

## Outputs

- `ssh_public_key`: The generated SSH public key (safe to display)

## Notes

- The private key is marked as sensitive and won't appear in logs
- SSH keys are essential for accessing private repositories
- The provider temporarily writes the private key to a file during processing
- Make sure your Bitrise organization has access to register keys if using provider registration
