package client

import (
	"context"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	// ScopeAccountingRead is required for account group queries.
	// Only requested when sync-account-groups is enabled, since existing
	// OAuth clients may not have this scope configured.
	ScopeAccountingRead = "core.accounting.read"

	// ScopeCommonRead covers the business_groups (Content Groups) API endpoint.
	// This scope is already included in ScopesReadOnly, so no conditional scope
	// injection is required for content groups (unlike account groups which need
	// the dedicated core.accounting.read scope).
	ScopeCommonRead = "core.common.read"
)

var (
	ScopesReadOnly = []string{
		"core.business_entity.read",
		"core.common.read",
		"core.user_group.read",
		"core.user.read",
		"email login",
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
