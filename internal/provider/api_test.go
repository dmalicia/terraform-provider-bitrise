package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"terraform-provider-bitrise/internal/bitriseapi"
)

func TestConfiguredClient_withoutProviderData_returnsNil(t *testing.T) {
	var diags diag.Diagnostics

	client := configuredClient(nil, "Unexpected Type", &diags)

	assert.Nil(t, client)
	assert.False(t, diags.HasError())
}

func TestConfiguredClient_withClient_returnsIt(t *testing.T) {
	var diags diag.Diagnostics
	want := bitriseapi.New("https://api.example.com", "token")

	client := configuredClient(want, "Unexpected Type", &diags)

	assert.Same(t, want, client)
	assert.False(t, diags.HasError())
}

func TestConfiguredClient_withUnexpectedType_addsError(t *testing.T) {
	var diags diag.Diagnostics

	client := configuredClient("not a client", "Unexpected Type", &diags)

	assert.Nil(t, client)
	require.Len(t, diags, 1)
	assert.Equal(t, "Unexpected Type", diags[0].Summary())
	assert.Contains(t, diags[0].Detail(), "got: string")
}

func TestResourceWithClient_Configure_withUnexpectedType_addsError(t *testing.T) {
	r := &resourceWithClient{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: 42}, resp)

	assert.Nil(t, r.client)
	require.Len(t, resp.Diagnostics, 1)
	assert.Equal(t, "Unexpected Resource Configure Type", resp.Diagnostics[0].Summary())
}

func TestDataSourceWithClient_Configure_withUnexpectedType_addsError(t *testing.T) {
	d := &dataSourceWithClient{}
	resp := &datasource.ConfigureResponse{}

	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: 42}, resp)

	assert.Nil(t, d.client)
	require.Len(t, resp.Diagnostics, 1)
	assert.Equal(t, "Unexpected Data Source Configure Type", resp.Diagnostics[0].Summary())
}

func TestCallDiags_Add_withCallError_reportsTextOfItsStage(t *testing.T) {
	cases := map[string]struct {
		stage   bitriseapi.Stage
		summary string
	}{
		"encode": {stage: bitriseapi.StageEncode, summary: "Error marshaling request"},
		"build":  {stage: bitriseapi.StageBuild, summary: "Error creating HTTP request"},
		"send":   {stage: bitriseapi.StageSend, summary: "Error sending HTTP request"},
		"read":   {stage: bitriseapi.StageRead, summary: "Error reading response body"},
		"decode": {stage: bitriseapi.StageDecode, summary: "Error parsing response"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var diags diag.Diagnostics
			err := &bitriseapi.CallError{Op: "read app app-123", Stage: tc.stage, Err: errors.New("boom")}

			standardCallDiags("%s - %s").add(&diags, err)

			require.Len(t, diags, 1)
			assert.Equal(t, tc.summary, diags[0].Summary())
			assert.Equal(t, "boom", diags[0].Detail())
		})
	}
}

func TestCallDiags_Add_withStatusError_reportsStatusAndBody(t *testing.T) {
	var diags diag.Diagnostics
	err := &bitriseapi.StatusError{Op: "read app app-123", Code: http.StatusForbidden, Body: `{"message":"forbidden"}`}

	standardCallDiags("Failed to read app: %s - %s").add(&diags, err)

	require.Len(t, diags, 1)
	assert.Equal(t, "API Error", diags[0].Summary())
	assert.Equal(t, `Failed to read app: 403 Forbidden - {"message":"forbidden"}`, diags[0].Detail())
}

func TestCallDiags_Add_withOtherError_reportsGenericError(t *testing.T) {
	var diags diag.Diagnostics

	standardCallDiags("%s - %s").add(&diags, errors.New("unexpected"))

	require.Len(t, diags, 1)
	assert.Equal(t, "Bitrise API Error", diags[0].Summary())
	assert.Equal(t, "unexpected", diags[0].Detail())
}
