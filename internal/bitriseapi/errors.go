package bitriseapi

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrNotFound matches, via errors.Is, a response with status 404.
var ErrNotFound = errors.New("not found")

// StatusError reports a response whose status code the caller did not accept.
type StatusError struct {
	Op   string
	Code int
	Body string

	status string
}

// Error implements error.
func (e *StatusError) Error() string {
	return fmt.Sprintf("%s: unexpected status %s: %s", e.Op, e.Status(), e.Body)
}

// Is reports whether a 404 status error is compared against ErrNotFound.
func (e *StatusError) Is(target error) bool {
	return target == ErrNotFound && e.Code == http.StatusNotFound
}

// Status returns the HTTP status line, e.g. "404 Not Found".
func (e *StatusError) Status() string {
	if e.status != "" {
		return e.status
	}
	return fmt.Sprintf("%d %s", e.Code, http.StatusText(e.Code))
}

// Stage identifies the step of an API call that failed.
type Stage string

// Stages of an API call, in execution order.
const (
	StageEncode Stage = "encode request"
	StageBuild  Stage = "build request"
	StageSend   Stage = "send request"
	StageRead   Stage = "read response"
	StageDecode Stage = "decode response"
)

// CallError reports an API call that failed before a usable response was obtained.
type CallError struct {
	Op    string
	Stage Stage
	Err   error

	// status is set when the body of a rejected response could not be read.
	status *StatusError
}

// Error implements error.
func (e *CallError) Error() string {
	return fmt.Sprintf("%s: %s: %v", e.Op, e.Stage, e.Err)
}

// Unwrap returns the cause and, when present, the rejected status.
func (e *CallError) Unwrap() []error {
	if e.status == nil {
		return []error{e.Err}
	}
	return []error{e.Err, e.status}
}
