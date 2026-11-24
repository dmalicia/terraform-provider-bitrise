package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"terraform-provider-bitrise/internal/bitriseapi"
)

// apiCaller is the part of the Bitrise API client used by resources and data sources.
type apiCaller interface {
	Do(ctx context.Context, req bitriseapi.Request, out any) error
}

// resourceWithClient implements Configure for resources that call the Bitrise API.
type resourceWithClient struct {
	client apiCaller
}

// Configure stores the client created by BitriseProvider.Configure.
func (r *resourceWithClient) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configuredClient(req.ProviderData, "Unexpected Resource Configure Type", &resp.Diagnostics); client != nil {
		r.client = client
	}
}

// dataSourceWithClient implements Configure for data sources that call the Bitrise API.
type dataSourceWithClient struct {
	client apiCaller
}

// Configure stores the client created by BitriseProvider.Configure.
func (d *dataSourceWithClient) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configuredClient(req.ProviderData, "Unexpected Data Source Configure Type", &resp.Diagnostics); client != nil {
		d.client = client
	}
}

// configuredClient returns the client set by BitriseProvider.Configure, or nil
// when the provider has not been configured yet.
func configuredClient(providerData any, summary string, diags *diag.Diagnostics) apiCaller {
	if providerData == nil {
		return nil
	}
	client, ok := providerData.(apiCaller)
	if !ok {
		diags.AddError(summary, fmt.Sprintf("Expected a Bitrise API client, got: %T. Please report this issue to the provider developers.", providerData))
		return nil
	}
	return client
}

// diagText is a diagnostic summary plus a detail format whose single verb receives the cause.
type diagText struct {
	summary string
	detail  string
}

// callDiags holds the diagnostics a call site reports for each way an API call can fail.
type callDiags struct {
	encode, build, send, read, decode diagText
	status                            func(*bitriseapi.StatusError) (summary, detail string)
}

// add reports err, as returned by apiCaller.Do, as an error diagnostic.
func (c callDiags) add(diags *diag.Diagnostics, err error) {
	var callErr *bitriseapi.CallError
	if errors.As(err, &callErr) {
		text := c.forStage(callErr.Stage)
		diags.AddError(text.summary, fmt.Sprintf(text.detail, callErr.Err))
		return
	}
	var statusErr *bitriseapi.StatusError
	if errors.As(err, &statusErr) {
		diags.AddError(c.status(statusErr))
		return
	}
	diags.AddError("Bitrise API Error", err.Error())
}

func (c callDiags) forStage(stage bitriseapi.Stage) diagText {
	switch stage {
	case bitriseapi.StageEncode:
		return c.encode
	case bitriseapi.StageBuild:
		return c.build
	case bitriseapi.StageSend:
		return c.send
	case bitriseapi.StageRead:
		return c.read
	default:
		return c.decode
	}
}

// standardCallDiags returns the texts shared by most resources and data sources.
func standardCallDiags(statusFormat string) callDiags {
	return callDiags{
		encode: diagText{"Error marshaling request", "%s"},
		build:  diagText{"Error creating HTTP request", "%s"},
		send:   diagText{"Error sending HTTP request", "%s"},
		read:   diagText{"Error reading response body", "%s"},
		decode: diagText{"Error parsing response", "%s"},
		status: func(e *bitriseapi.StatusError) (string, string) {
			return "API Error", fmt.Sprintf(statusFormat, e.Status(), e.Body)
		},
	}
}
