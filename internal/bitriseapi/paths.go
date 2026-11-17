package bitriseapi

import "net/url"

// Paths of the API endpoints used by the provider, relative to the base URL.
const (
	RegisterAppPath     = "/v0.1/apps/register"
	AvailableStacksPath = "/v0.1/available-stacks"
)

// AppPath returns the path of an app.
func AppPath(appSlug string) string {
	return "/v0.1/apps/" + appSlug
}

// AppFinishPath returns the path that finishes the setup of an app.
func AppFinishPath(appSlug string) string {
	return AppPath(appSlug) + "/finish"
}

// AppSSHKeyPath returns the path that registers the SSH key of an app.
func AppSSHKeyPath(appSlug string) string {
	return AppPath(appSlug) + "/register-ssh-key"
}

// AppBitriseYMLPath returns the path of the bitrise.yml of an app.
func AppBitriseYMLPath(appSlug string) string {
	return AppPath(appSlug) + "/bitrise.yml"
}

// AppRolePath returns the path of the groups assigned to a role of an app.
func AppRolePath(appSlug, roleName string) string {
	return AppPath(appSlug) + "/roles/" + roleName
}

// AppSecretsPath returns the path of the secrets collection of an app.
func AppSecretsPath(appSlug string) string {
	return AppPath(appSlug) + "/secrets"
}

// AppSecretPath returns the path of a single app secret.
func AppSecretPath(appSlug, name string) string {
	return AppSecretsPath(appSlug) + "/" + name
}

// OrgGroupsPath returns the path of the groups of an organization.
func OrgGroupsPath(orgSlug string) string {
	return "/v0.1/organizations/" + orgSlug + "/groups"
}

// OrgSecretsPath returns the path of the secrets collection of an organization.
func OrgSecretsPath(orgSlug string) string {
	return "/v0.1/organizations/" + url.PathEscape(orgSlug) + "/secrets"
}

// OrgSecretPath returns the path of a single organization secret.
func OrgSecretPath(orgSlug, name string) string {
	return OrgSecretsPath(orgSlug) + "/" + url.PathEscape(name)
}
