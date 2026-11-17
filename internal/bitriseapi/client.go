// Package bitriseapi is a minimal client for the Bitrise REST API.
package bitriseapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Client sends authenticated JSON requests to the Bitrise API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New returns a Client for the API rooted at baseURL that authenticates with token.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: &authTransport{token: token, base: http.DefaultTransport},
		},
	}
}

// Request describes a single API call.
type Request struct {
	// Op names the operation in errors, e.g. "create app secret MY_SECRET".
	Op     string
	Method string
	// Path is appended verbatim to the base URL.
	Path string
	// Body is sent JSON-encoded when non-nil.
	Body any
	// Accept lists the status codes treated as success; empty means 200 only.
	Accept []int
}

// RawResponse receives an accepted response body without decoding it.
type RawResponse struct {
	ContentType string
	Body        []byte
}

// Do sends req and decodes an accepted response body into out.
//
// A nil out skips reading the body; a *RawResponse receives it undecoded.
// A rejected status yields a *StatusError, any other failure a *CallError.
func (c *Client) Do(ctx context.Context, req Request, out any) error {
	httpReq, err := c.newHTTPRequest(ctx, req)
	if err != nil {
		return err
	}

	tflog.Debug(ctx, "Sending Bitrise API request", map[string]any{
		"method": req.Method,
		"url":    httpReq.URL.String(),
	})
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return &CallError{Op: req.Op, Stage: StageSend, Err: err}
	}
	defer drainAndClose(resp.Body)
	tflog.Debug(ctx, "Received Bitrise API response", map[string]any{"status": resp.StatusCode})

	if !accepted(req.Accept, resp.StatusCode) {
		return rejected(req.Op, resp)
	}
	if out == nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &CallError{Op: req.Op, Stage: StageRead, Err: err}
	}
	if raw, ok := out.(*RawResponse); ok {
		raw.ContentType = resp.Header.Get("Content-Type")
		raw.Body = body
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &CallError{Op: req.Op, Stage: StageDecode, Err: err}
	}
	return nil
}

func (c *Client) newHTTPRequest(ctx context.Context, req Request) (*http.Request, error) {
	var body io.Reader
	if req.Body != nil {
		payload, err := json.Marshal(req.Body)
		if err != nil {
			return nil, &CallError{Op: req.Op, Stage: StageEncode, Err: err}
		}
		body = bytes.NewReader(payload)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, c.baseURL+req.Path, body)
	if err != nil {
		return nil, &CallError{Op: req.Op, Stage: StageBuild, Err: err}
	}
	return httpReq, nil
}

func accepted(codes []int, code int) bool {
	if len(codes) == 0 {
		return code == http.StatusOK
	}
	return slices.Contains(codes, code)
}

// rejected turns a response with an unexpected status into a StatusError.
func rejected(op string, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	statusErr := &StatusError{Op: op, Code: resp.StatusCode, Body: string(body), status: resp.Status}
	if err != nil {
		return &CallError{Op: op, Stage: StageRead, Err: err, status: statusErr}
	}
	return statusErr
}

// drainAndClose lets the transport reuse the connection without reading unbounded data.
func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 64<<10))
	_ = body.Close()
}

// authTransport adds the API token and JSON content type to every request.
type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", t.token)
	req.Header.Set("Content-Type", "application/json")
	return t.base.RoundTrip(req)
}
