package bitriseapi

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusError_Is_matchesErrNotFoundOnlyFor404(t *testing.T) {
	cases := map[string]struct {
		code int
		want bool
	}{
		"404 not found":     {code: http.StatusNotFound, want: true},
		"410 gone":          {code: http.StatusGone, want: false},
		"500 server error":  {code: http.StatusInternalServerError, want: false},
		"401 unauthorized":  {code: http.StatusUnauthorized, want: false},
		"200 unexpected ok": {code: http.StatusOK, want: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := &StatusError{Op: "read app app-123", Code: tc.code}

			got := errors.Is(err, ErrNotFound)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStatusError_Status_withoutStatusLine_formatsCode(t *testing.T) {
	err := &StatusError{Op: "read app app-123", Code: http.StatusBadGateway}

	got := err.Status()

	assert.Equal(t, "502 Bad Gateway", got)
}

func TestStatusError_Error_includesOpStatusAndBody(t *testing.T) {
	err := &StatusError{Op: "read app app-123", Code: http.StatusForbidden, Body: `{"message":"forbidden"}`}

	got := err.Error()

	assert.Equal(t, `read app app-123: unexpected status 403 Forbidden: {"message":"forbidden"}`, got)
}

func TestCallError_Error_includesOpStageAndCause(t *testing.T) {
	err := &CallError{Op: "list available stacks", Stage: StageRead, Err: ErrBodyTooLarge}

	got := err.Error()

	assert.Equal(t, "list available stacks: read response: response body exceeds 1 MiB", got)
}
