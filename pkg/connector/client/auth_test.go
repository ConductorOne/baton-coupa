package client

import (
	"errors"
	"net/http"
	"testing"

	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWrapTokenError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name: "invalid_client maps to unauthenticated",
			err: &oauth2.RetrieveError{
				Response:  &http.Response{StatusCode: http.StatusBadRequest},
				ErrorCode: "invalid_client",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "invalid_grant maps to unauthenticated",
			err: &oauth2.RetrieveError{
				Response:  &http.Response{StatusCode: http.StatusBadRequest},
				ErrorCode: "invalid_grant",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "unauthorized_client maps to permission denied",
			err: &oauth2.RetrieveError{
				Response:  &http.Response{StatusCode: http.StatusForbidden},
				ErrorCode: "unauthorized_client",
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name: "access_denied maps to permission denied",
			err: &oauth2.RetrieveError{
				Response:  &http.Response{StatusCode: http.StatusForbidden},
				ErrorCode: "access_denied",
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name: "invalid_scope maps to invalid argument",
			err: &oauth2.RetrieveError{
				Response:  &http.Response{StatusCode: http.StatusBadRequest},
				ErrorCode: "invalid_scope",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid_request maps to invalid argument",
			err: &oauth2.RetrieveError{
				Response:  &http.Response{StatusCode: http.StatusBadRequest},
				ErrorCode: "invalid_request",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "unsupported_grant_type maps to invalid argument",
			err: &oauth2.RetrieveError{
				Response:  &http.Response{StatusCode: http.StatusBadRequest},
				ErrorCode: "unsupported_grant_type",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "unsupported_response_type maps to invalid argument",
			err: &oauth2.RetrieveError{
				Response:  &http.Response{StatusCode: http.StatusBadRequest},
				ErrorCode: "unsupported_response_type",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "unrecognized error code falls back to HTTP status",
			err: &oauth2.RetrieveError{
				Response:  &http.Response{StatusCode: http.StatusServiceUnavailable},
				ErrorCode: "server_error",
			},
			wantCode: codes.Unavailable,
		},
		{
			name: "recognized error code with nil response still classifies",
			err: &oauth2.RetrieveError{
				ErrorCode: "invalid_client",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "empty error code with response falls back to HTTP status",
			err: &oauth2.RetrieveError{
				Response: &http.Response{StatusCode: http.StatusInternalServerError},
			},
			// 5xx all map to Unavailable; see uhttp.GrpcCodeFromHTTPStatus.
			wantCode: codes.Unavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := wrapTokenError("failed to obtain read-only token", tt.err)

			st, ok := status.FromError(wrapped)
			if !ok {
				t.Fatalf("expected a gRPC status error, got %T: %v", wrapped, wrapped)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("got code %s, want %s (message: %s)", st.Code(), tt.wantCode, st.Message())
			}
			if !errors.Is(wrapped, tt.err) {
				t.Errorf("wrapped error does not preserve the original error via errors.Is")
			}
		})
	}
}

func TestWrapTokenError_Nil(t *testing.T) {
	if err := wrapTokenError("failed to obtain read-only token", nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestWrapTokenError_NonRetrieveError(t *testing.T) {
	original := errors.New("connection reset")

	wrapped := wrapTokenError("failed to obtain read-only token", original)

	if !errors.Is(wrapped, original) {
		t.Errorf("wrapped error does not preserve the original error via errors.Is")
	}
	if _, ok := status.FromError(wrapped); ok {
		t.Errorf("expected a plain wrapped error, not a gRPC status, for a non-RetrieveError")
	}
}

func TestOauthTokenErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  *oauth2.RetrieveError
		want string
	}{
		{
			name: "error code and description both present",
			err:  &oauth2.RetrieveError{ErrorCode: "invalid_client", ErrorDescription: "'client_secret' missing"},
			want: "invalid_client: 'client_secret' missing",
		},
		{
			name: "only error code present",
			err:  &oauth2.RetrieveError{ErrorCode: "invalid_client"},
			want: "invalid_client",
		},
		{
			name: "only description present, no error code (non-conformant server)",
			err:  &oauth2.RetrieveError{ErrorDescription: "client secret is missing"},
			want: "client secret is missing",
		},
		{
			name: "only response present",
			err:  &oauth2.RetrieveError{Response: &http.Response{Status: "400 Bad Request"}},
			want: "400 Bad Request",
		},
		{
			name: "nothing present",
			err:  &oauth2.RetrieveError{},
			want: "oauth2 token request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := oauthTokenErrorMessage(tt.err)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
