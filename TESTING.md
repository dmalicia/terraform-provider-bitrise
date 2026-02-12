# Testing Guide for Bitrise Terraform Provider

This guide provides test scenarios and validation steps for all Bitrise Terraform provider resources.

## Prerequisites

- Valid Bitrise Personal Access Token with admin access
- Bitrise organization slug for testing
- Git repository (GitHub, GitLab, or Bitbucket) for app registration tests
- Terraform >= 1.0 installed

## Testing Strategy

### Resources to Test

1. **bitrise_app** - Application registration
2. **bitrise_app_ssh** - SSH key configuration
3. **bitrise_app_finish** - Application finalization
4. **bitrise_app_secret** - Secret management
5. **bitrise_app_bitrise_yml** - YAML configuration
6. **bitrise_app_roles** - Role management

## Manual Testing Scenarios

### Test Suite 1: Application Registration (bitrise_app)

#### Test 1.1: Basic App Registration

**Objective**: Verify basic app creation works

**Steps**:
1. Create a Terraform configuration:
```hcl
resource "bitrise_app" "test" {
  repo              = "github"
  repo_url          = "https://github.com/yourorg/testrepo"
  type              = "git"
  git_repo_slug     = "testrepo"
  git_owner         = "yourorg"
  organization_slug = "your-org-slug"
  is_public         = false
}
```

2. Run `terraform apply`
3. Verify in Bitrise UI that the app exists
4. Run `terraform plan` - should show no changes

**Expected Result**: App is created successfully, app_slug is generated

#### Test 1.2: Public App Registration

**Objective**: Verify public app registration

**Steps**:
1. Set `is_public = true` in configuration
2. Apply and verify app is marked as public in Bitrise

**Expected Result**: App is created as public

#### Test 1.3: App Deletion

**Objective**: Verify app deletion works

**Steps**:
1. Create an app
2. Remove from Terraform configuration
3. Run `terraform apply`
4. Verify app is deleted from Bitrise

**Expected Result**: App is successfully deleted

---

### Test Suite 2: SSH Key Configuration (bitrise_app_ssh)

#### Test 2.1: SSH Key with Generated Keys

**Objective**: Verify SSH key configuration with TLS provider

**Steps**:
1. Create configuration:
```hcl
resource "tls_private_key" "test" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "bitrise_app_ssh" "test" {
  app_slug             = "your-app-slug"
  auth_ssh_private_key = tls_private_key.test.private_key_pem
  auth_ssh_public_key  = tls_private_key.test.public_key_openssh
}
```

2. Apply and verify SSH key is configured in Bitrise

**Expected Result**: SSH keys are successfully registered

#### Test 2.2: SSH Key with Provider Registration

**Objective**: Test automatic provider registration

**Steps**:
1. Add `is_register_key_into_provider_service = true`
2. Apply configuration
3. Verify public key is registered with git provider

**Expected Result**: Public key is registered with the git provider

#### Test 2.3: SSH Key from Files

**Objective**: Test using existing SSH keys

**Steps**:
1. Use `file()` function to load existing keys
2. Apply and verify

**Expected Result**: Keys are loaded and configured correctly

---

### Test Suite 3: Application Finalization (bitrise_app_finish)

#### Test 3.1: iOS App Configuration

**Objective**: Verify iOS app setup

**Steps**:
1. Create configuration:
```hcl
resource "bitrise_app_finish" "ios" {
  app_slug          = bitrise_app.test.app_slug
  project_type      = "ios"
  stack_id          = "osx-xcode-14.2.x"
  config            = file("bitrise-ios.yml")
  mode              = "manual"
  organization_slug = "your-org-slug"
}
```

2. Apply and verify in Bitrise

**Expected Result**: App is configured with iOS settings

#### Test 3.2: Android App Configuration

**Objective**: Verify Android app setup

**Steps**:
1. Use `project_type = "android"` and `stack_id = "linux-docker-android-22.04"`
2. Apply and verify

**Expected Result**: App is configured with Android settings

#### Test 3.3: Environment Variables

**Objective**: Test environment variable configuration

**Steps**:
1. Add `envs` map to configuration
2. Apply and verify environment variables in Bitrise

**Expected Result**: Environment variables are set correctly

---

### Test Suite 4: Secret Management (bitrise_app_secret)

### Test Suite 4: Secret Management (bitrise_app_secret)

#### Test 4.1: Basic Secret Creation

**Objective**: Verify basic secret creation works

**Steps**:
1. Create a Terraform configuration:
```hcl
resource "bitrise_app_secret" "test" {
  app_slug = "your-test-app-slug"
  name     = "TEST_SECRET"
  value    = "test-value-123"
}
```

2. Run `terraform apply`
3. Verify in Bitrise UI that the secret exists
4. Run `terraform plan` - should show no changes

**Expected Result**: Secret is created successfully, no drift detected

---

#### Test 4.2: Protected Secret

**Objective**: Verify protected secrets work correctly

**Steps**:
1. Create a protected secret:
```hcl
resource "bitrise_app_secret" "protected" {
  app_slug     = "your-test-app-slug"
  name         = "PROTECTED_SECRET"
  value        = "protected-value"
  is_protected = true
}
```

2. Run `terraform apply`
3. Try to read the secret via Bitrise API/UI (value should be hidden)
4. Modify the value in configuration
5. Run `terraform plan`

**Expected Result**: 
- Secret is created with protected flag
- Value is not visible in Bitrise
- Updates are detected properly

---

#### Test 4.3: Pull Request Exposure

**Objective**: Test PR exposure setting

**Steps**:
1. Create a secret with PR exposure enabled
2. Apply and verify the setting in Bitrise

**Expected Result**: Secret is marked as exposed for PRs

---

#### Test 4.4: Update Existing Secret

**Objective**: Verify updates work correctly

**Steps**:
1. Create a secret
2. Modify the value in Terraform
3. Run `terraform apply`
4. Verify the value changed in Bitrise

**Expected Result**: Secret value is updated successfully

---

#### Test 4.5: Import Secret

**Objective**: Verify import functionality

**Steps**:
1. Create a secret manually in Bitrise
2. Import it: `terraform import bitrise_app_secret.test app-slug/SECRET_NAME`
3. Run `terraform plan`

**Expected Result**: Secret is imported successfully

---

### Test Suite 5: Role Management (bitrise_app_roles)

#### Test 5.1: Basic Role Assignment

**Objective**: Verify role assignment works

**Steps**:
1. Create configuration:
```hcl
resource "bitrise_app_roles" "admins" {
  app_slug  = "your-app-slug"
  role_name = "admin"
  groups    = ["admin-team"]
}
```

2. Apply and verify in Bitrise Team settings

**Expected Result**: Groups are assigned to admin role

---

#### Test 5.2: Multiple Roles

**Objective**: Test managing multiple role types

**Steps**:
1. Create resources for admin, developer, and member roles
2. Apply all at once
3. Verify all roles are configured

**Expected Result**: All role assignments are created correctly

---

#### Test 5.3: Update Role Groups

**Objective**: Verify role updates work

**Steps**:
1. Create a role with specific groups
2. Modify the groups list
3. Apply and verify

**Expected Result**: Group assignments are updated

---

### Test Suite 6: Integration Tests

#### Test 6.1: Complete App Setup Flow

**Objective**: Verify all resources work together

**Steps**:
1. Create configuration with all resources:
   - bitrise_app
   - bitrise_app_ssh
   - bitrise_app_finish
   - bitrise_app_secret (multiple)
   - bitrise_app_roles

2. Apply in order and verify each step

**Expected Result**: Complete app setup works end-to-end

---

#### Test 6.2: Dependency Handling

**Objective**: Verify resource dependencies work

**Steps**:
1. Use `app_slug` from `bitrise_app` in other resources
2. Apply all at once
3. Verify Terraform handles dependencies correctly

**Expected Result**: Terraform creates resources in correct order

---

#### Test 6.3: Multi-Environment Setup

**Objective**: Test managing multiple environments

**Steps**:
1. Create configurations for dev, staging, prod
2. Apply and verify separation

**Expected Result**: Multiple environments are managed independently

---

## Automated Testing Checklist

For implementing automated acceptance tests:

### Application Resource Tests (`app_resource_test.go`)
- [ ] TestAccApp_basic - Basic app creation and read
- [ ] TestAccApp_publicRepo - Public repository handling
- [ ] TestAccApp_update - App updates
- [ ] TestAccApp_disappears - Resource disappears handling
- [ ] TestAccApp_invalidOrg - Error handling for invalid organization

### SSH Resource Tests (`app_ssh_resource_test.go`)
- [ ] TestAccAppSSH_basic - Basic SSH key configuration
- [ ] TestAccAppSSH_generated - Generated keys with TLS provider
- [ ] TestAccAppSSH_withProviderRegistration - Provider registration enabled
- [ ] TestAccAppSSH_update - SSH key updates
- [ ] TestAccAppSSH_invalidKeys - Error handling for invalid keys

### Finish Resource Tests (`app_finish_resource_test.go`)
- [ ] TestAccAppFinish_ios - iOS project configuration
- [ ] TestAccAppFinish_android - Android project configuration
- [ ] TestAccAppFinish_reactNative - React Native configuration
- [ ] TestAccAppFinish_withEnvs - Environment variables
- [ ] TestAccAppFinish_update - Configuration updates
- [ ] TestAccAppFinish_invalidStack - Error handling for invalid stack

### Secret Resource Tests (`app_secrets_resource_test.go`)
- [ ] TestAccAppSecret_basic - Basic creation and read
- [ ] TestAccAppSecret_update - Value updates
- [ ] TestAccAppSecret_protected - Protected secrets
- [ ] TestAccAppSecret_pullRequest - PR exposure settings
- [ ] TestAccAppSecret_expansion - Variable expansion settings
- [ ] TestAccAppSecret_import - Import functionality
- [ ] TestAccAppSecret_disappears - Resource disappears handling
- [ ] TestAccAppSecret_invalidAppSlug - Error handling

### Role Resource Tests (`app_roles_resource_test.go`)
- [ ] TestAccAppRoles_basic - Basic role assignment
- [ ] TestAccAppRoles_update - Group updates
- [ ] TestAccAppRoles_multipleRoles - Multiple role types
- [ ] TestAccAppRoles_import - Import functionality
- [ ] TestAccAppRoles_invalidRole - Error handling for invalid role names

### Integration Tests
- [ ] TestAccIntegration_completeSetup - Full app setup workflow
- [ ] TestAccIntegration_multipleApps - Multiple apps management
- [ ] TestAccIntegration_dependencies - Resource dependency handling

## Validation Points

After each test, verify:

1. **Terraform State**: 
   - Run `terraform show` to check state matches expected values
   - Verify sensitive values are marked as sensitive
   - Check computed values (app_slug, id) are populated

2. **Bitrise UI**:
   - **Apps**: Verify app appears in organization apps list
   - **SSH Keys**: Check SSH key is configured in app settings
   - **Configuration**: Verify project type and stack are correct
   - **Secrets**: Check secrets exist with correct flags (protected, PR exposure, expansion)
   - **Team**: Verify role assignments in Team tab

3. **API Response**:
   - Protected secrets don't return values
   - Non-protected secrets return correct values
   - App slug format is correct
   - All attributes are properly populated

4. **Drift Detection**:
   - Run `terraform plan` after applying - should show no changes
   - Manually change resource in Bitrise UI, run `terraform plan` - should detect drift
   - Protected secrets may always show as changed (expected behavior)

5. **Dependencies**:
   - Resources that depend on app_slug are created after the app
   - Deletion happens in reverse dependency order
   - No circular dependencies

## Common Issues and Solutions

### Application Registration Issues

**Issue**: App registration fails with authentication error
**Solution**: 
- Verify Bitrise token has admin access to the organization
- Check organization slug is correct
- Ensure git repository is accessible to your Bitrise account

**Issue**: App already exists
**Solution**: 
- Import existing app: `terraform import bitrise_app.existing app-slug`
- Or delete manually from Bitrise and rerun

### SSH Configuration Issues

**Issue**: SSH key registration fails
**Solution**:
- Verify private and public keys are valid and match
- Check key format (PEM for private, OpenSSH for public)
- Ensure app exists before trying to configure SSH

**Issue**: Provider registration fails
**Solution**:
- Verify Bitrise has OAuth access to your git provider
- Check that the integration is properly configured in Bitrise

### App Finish Issues

**Issue**: Invalid stack ID error
**Solution**:
- Check [Bitrise stacks documentation](https://devcenter.bitrise.io/en/infrastructure/build-stacks.html)
- Verify stack is compatible with project type
- Use correct stack ID format

**Issue**: Invalid bitrise.yml
**Solution**:
- Validate YAML syntax
- Ensure file exists at specified path
- Check file encoding (should be UTF-8)

### Secret Management Issues

**Issue**: Protected secret shows as changed every plan
**Solution**: This is expected. Protected secrets can't be read to detect drift.

**Issue**: Import fails for protected secret
**Solution**: You need to manually set the correct value in Terraform config before import.

**Issue**: Secret not updating
**Solution**: 
- Check if it's protected - may need to delete and recreate
- Verify app slug is correct
- Check for API rate limiting

### Role Management Issues

**Issue**: Invalid role name error
**Solution**: Use valid role names: admin, developer, member, platform_engineer

**Issue**: Group not found
**Solution**: 
- Verify group slug is correct
- Use data source `bitrise_org_groups` to list available groups
- Ensure groups exist in the organization

## Security Testing

Critical security checks:

- [ ] Verify sensitive values don't appear in `terraform plan` output
- [ ] Verify sensitive values don't appear in logs (check with `TF_LOG=DEBUG`)
- [ ] Verify protected secrets can't be read via API
- [ ] Verify PR-exposed secrets are truly exposed (test in actual PR build)
- [ ] Ensure SSH private keys are never logged
- [ ] Check that tokens are not exposed in error messages
- [ ] Verify state file encryption if using remote backend

## Performance Testing

For large-scale deployments:

- [ ] Test creating multiple apps simultaneously
- [ ] Test bulk secret creation (100+ secrets)
- [ ] Test import of existing resources
- [ ] Measure time for complete app setup workflow
- [ ] Check API rate limiting behavior

## End-to-End Workflow Test

Complete workflow validation:

```bash
# 1. Clean slate
terraform destroy -auto-approve

# 2. Create everything
terraform apply -auto-approve

# 3. Verify no drift
terraform plan | grep "No changes"

# 4. Manual change in Bitrise UI
# - Change a secret value
# - Modify a role assignment

# 5. Detect drift
terraform plan | grep "will be updated"

# 6. Reconcile
terraform apply -auto-approve

# 7. Clean up
terraform destroy -auto-approve
```

## Testing with Multiple Environments

Test configuration for dev, staging, and production:

```hcl
module "bitrise_app" {
  source = "./modules/bitrise-app"
  
  for_each = {
    dev     = { stack = "osx-xcode-14.2.x", protected_secrets = false }
    staging = { stack = "osx-xcode-14.2.x", protected_secrets = false }
    prod    = { stack = "osx-xcode-15.0.x", protected_secrets = true }
  }
  
  environment       = each.key
  stack_id          = each.value.stack
  protected_secrets = each.value.protected_secrets
  organization_slug = var.organization_slug
}
```

Verify:
- [ ] All environments are created successfully
- [ ] Environments are isolated
- [ ] Environment-specific configurations are applied
- [ ] Updates to one environment don't affect others
