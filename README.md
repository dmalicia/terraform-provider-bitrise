# Terraform Provider README

This repository serves as a template for creating a Terraform provider using the Terraform Plugin Framework. It provides a starting point for building Terraform providers and includes the following components:

- A resource and a data source implementation (internal/provider/)
- Examples demonstrating usage (examples/)
- Generated documentation (docs/)
- Miscellaneous meta files

## Getting Started

Before you begin, ensure that you meet the following requirements:

- Terraform >= 1.0
- Go >= 1.19

### Building the Provider

1. Clone this repository to your local machine.
2. Navigate to the repository directory.
3. Build the provider using the following Go install command:

   sh
   go install

### Adding Dependencies

This provider uses Go modules. To add a new dependency (github.com/author/dependency) to your Terraform provider:

sh
go get github.com/author/dependency
go mod tidy

Remember to commit the changes to go.mod and go.sum.

### Using the Provider

Replace this section with usage instructions specific to your provider.

## Provider Implementation Overview

This provider is built for the Bitrise API, enabling infrastructure-as-code management of Bitrise applications and their configurations. Here's a brief overview of the key components:

### Provider Components

- **BitriseProvider**: The main provider struct that implements the provider.Provider interface. It handles metadata, schema, and resource configuration.

- **ClientProvider**: An interface for getting an HTTP client. It appears to be part of your provider's dependency injection mechanism.

- **authenticatedTransport**: An HTTP transport implementation that adds an authorization header to outgoing requests.

### Resources

- **bitrise_app**: Manages Bitrise applications (create, read, delete) - Register apps from various git providers
- **bitrise_app_ssh**: Manages SSH keys for Bitrise applications - Configure secure repository access
- **bitrise_app_finish**: Completes the application registration process - Set project type, stack, and build configuration
- **bitrise_app_secret**: Manages secrets (environment variables) for Bitrise applications - Full CRUD with protection options
- **bitrise_app_bitrise_yml**: Manages Bitrise YAML configuration for applications
- **bitrise_app_roles**: Manages team role assignments for applications - Control access and permissions

### Data Sources

- **bitrise_app_roles**: Retrieve role assignments for an application
- **bitrise_org_groups**: Retrieve organization groups for access management

### Example Usage

```hcl
provider "bitrise" {
  endpoint = "https://api.bitrise.io"
  token    = var.bitrise_token
}

# Complete application setup workflow

# 1. Register the application
resource "bitrise_app" "my_app" {
  repo              = "github"
  repo_url          = "https://github.com/myorg/myrepo"
  type              = "git"
  git_repo_slug     = "myrepo"
  git_owner         = "myorg"
  organization_slug = "my-bitrise-org"
}

# 2. Configure SSH keys (for private repos)
resource "tls_private_key" "bitrise_ssh" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "bitrise_app_ssh" "my_app_ssh" {
  app_slug             = bitrise_app.my_app.app_slug
  auth_ssh_private_key = tls_private_key.bitrise_ssh.private_key_pem
  auth_ssh_public_key  = tls_private_key.bitrise_ssh.public_key_openssh
}

# 3. Complete app setup with project configuration
resource "bitrise_app_finish" "my_app_config" {
  app_slug          = bitrise_app.my_app.app_slug
  project_type      = "ios"
  stack_id          = "osx-xcode-14.2.x"
  config            = file("bitrise.yml")
  mode              = "manual"
  organization_slug = "my-bitrise-org"
}

# 4. Add secrets
resource "bitrise_app_secret" "api_key" {
  app_slug     = bitrise_app.my_app.app_slug
  name         = "API_KEY"
  value        = var.api_key
  is_protected = true
}

# 5. Configure team access
resource "bitrise_app_roles" "admin_groups" {
  app_slug  = bitrise_app.my_app.app_slug
  role_name = "admin"
  groups    = ["admin-team"]
}
```

For detailed usage examples, see the [examples](./examples/) directory.

## Developing the Provider

If you wish to contribute to or modify the provider, follow these steps:

1. Ensure that you have Go installed on your machine (see Requirements above).
2. Compile the provider by running the following command:

   sh
   go install

   This command will build the provider and place the binary in the $GOPATH/bin directory.

3. Generate or update documentation using the command:

   sh
   go generate

4. Run the full suite of acceptance tests by executing:

   sh
   make testacc

   Note: Acceptance tests create real resources, which may incur costs.

## Further Resources

For more detailed information on creating Terraform providers, you can explore tutorials and guides on the HashiCorp Developer platform. Additionally, consult the official Terraform documentation to learn about Terraform Plugin Framework-specific details and best practices.

When your provider is ready, consider publishing it on the Terraform Registry so that others can benefit from and use it.

### Some notes for reference...

If you want to locally prove the provider without running any scripts...

```shell
go build -ldflags="-X main.version=0.0.1 -X main.commit=n/a"
mv terraform-provider-bitrise ~/.terraform.d/plugins/terraform.local/local/bitrise/0.0.1/darwin_arm64/terraform-provider-bitrise_v0.0.1
```

and in your providers.tf

```shell
    bitrise = {
      source = "terraform.local/local/bitrise"
      version = "0.0.1"
    }
```