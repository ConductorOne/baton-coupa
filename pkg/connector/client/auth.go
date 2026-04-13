package client

import (
	"context"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
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

	// ScopesAccounting are the additional OAuth scopes required for
	// billing account sync. Customers must grant these scopes on their
	// Coupa OAuth client to use billing account features.
	// If the OAuth client does not have these scopes, billing account
	// sync will not work, but the rest of the connector will function
	// normally.
	ScopesAccounting = []string{
		"core.accounting.read",
		"core.accounting.write",
	}
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
