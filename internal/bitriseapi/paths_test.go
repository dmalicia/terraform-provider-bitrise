package bitriseapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaths_buildEndpointPaths(t *testing.T) {
	cases := map[string]struct {
		got  string
		want string
	}{
		"app":                {got: AppPath("app-123"), want: "/v0.1/apps/app-123"},
		"app finish":         {got: AppFinishPath("app-123"), want: "/v0.1/apps/app-123/finish"},
		"app ssh key":        {got: AppSSHKeyPath("app-123"), want: "/v0.1/apps/app-123/register-ssh-key"},
		"app bitrise.yml":    {got: AppBitriseYMLPath("app-123"), want: "/v0.1/apps/app-123/bitrise.yml"},
		"app role":           {got: AppRolePath("app-123", "admin"), want: "/v0.1/apps/app-123/roles/admin"},
		"app secrets":        {got: AppSecretsPath("app-123"), want: "/v0.1/apps/app-123/secrets"},
		"app secret":         {got: AppSecretPath("app-123", "MY_SECRET"), want: "/v0.1/apps/app-123/secrets/MY_SECRET"},
		"org groups":         {got: OrgGroupsPath("org-abc"), want: "/v0.1/organizations/org-abc/groups"},
		"org secrets":        {got: OrgSecretsPath("org-abc"), want: "/v0.1/organizations/org-abc/secrets"},
		"org secret":         {got: OrgSecretPath("org-abc", "MY_SECRET"), want: "/v0.1/organizations/org-abc/secrets/MY_SECRET"},
		"org secret escaped": {got: OrgSecretPath("org abc", "A/B"), want: "/v0.1/organizations/org%20abc/secrets/A%2FB"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}
