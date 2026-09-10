package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc/codes"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

const (
	// ScopeAccountingRead is required for account group queries.
	// Only requested when sync-account-groups is enabled, since existing
	// OAuth clients may not have this scope configured.
	ScopeAccountingRead = "core.accounting.read"
)

var (
	ScopesReadOnly = []string{
		"core.business_entity.read",
		"core.common.read",
		"core.user_group.read",
		"core.user.read",
		"login",
		"openid",
		"profile",
	}
	ScopesReadWrite = append(
		ScopesReadOnly,
		"core.user_group.write",
		"core.user.write",
	)
)

func getTokenSource(
	ctx context.Context,
	baseUrl *url.URL,
	clientId string,
	clientSecret string,
	scopes ...string,
) oauth2.TokenSource {
	cfg := clientcredentials.Config{
		AuthStyle:    oauth2.AuthStyleInHeader,
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes:       scopes,
		TokenURL:     baseUrl.JoinPath(apiPathAuth).String(),
	}
	return cfg.TokenSource(ctx)
}

// wrapTokenError maps an oauth2 token-exchange failure to a gRPC status by
// its RFC 6749 §5.2 "error" parameter, since golang.org/x/oauth2 bypasses
// uhttp.BaseHttpClient and would otherwise surface as an unclassified
// codes.Unknown.
func wrapTokenError(action string, err error) error {
	if err == nil {
		return nil
	}

	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) {
		code := codes.Unknown
		switch retrieveErr.ErrorCode {
		case "invalid_client", "invalid_grant":
			code = codes.Unauthenticated
		case "unauthorized_client", "access_denied":
			code = codes.PermissionDenied
		case "invalid_scope", "invalid_request", "unsupported_grant_type", "unsupported_response_type":
			code = codes.InvalidArgument
		default:
			if retrieveErr.Response != nil {
				code = uhttp.GrpcCodeFromHTTPStatus(retrieveErr.Response.StatusCode)
			}
		}
		return uhttp.WrapErrors(code, fmt.Sprintf("baton-coupa: %s: %s", action, oauthTokenErrorMessage(retrieveErr)), err)
	}

	return fmt.Errorf("baton-coupa: %s: %w", action, err)
}

// oauthTokenErrorMessage prefers the RFC 6749 error/error_description pair
// the token endpoint sent, since the HTTP status alone can be misleading
// (e.g. Coupa returns invalid_client on a 400 regardless of cause).
func oauthTokenErrorMessage(retrieveErr *oauth2.RetrieveError) string {
	switch {
	case retrieveErr.ErrorCode != "" && retrieveErr.ErrorDescription != "":
		return fmt.Sprintf("%s: %s", retrieveErr.ErrorCode, retrieveErr.ErrorDescription)
	case retrieveErr.ErrorCode != "":
		return retrieveErr.ErrorCode
	case retrieveErr.ErrorDescription != "":
		return retrieveErr.ErrorDescription
	case retrieveErr.Response != nil:
		return retrieveErr.Response.Status
	default:
		return "oauth2 token request failed"
	}
}
