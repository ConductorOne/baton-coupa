package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"go.uber.org/zap"

	"github.com/conductorone/baton-coupa/pkg/config"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"golang.org/x/oauth2"
)

type innerGraphqlResponse struct {
	Data   *json.RawMessage `json:"data,omitempty"`
	Errors []struct {
		Message string `json:"message,omitempty"`
	} `json:"errors"`
}

type Client struct {
	baseUrl              *url.URL
	readOnlyToken        string
	readWriteToken       string
	initialized          bool
	ReadOnlyTokenSource  oauth2.TokenSource
	readWriteTokenSource oauth2.TokenSource
	wrapper              *uhttp.BaseHttpClient
}

func New(
	ctx context.Context,
	instanceUrl string,
	clientId string,
	clientSecret string,
) (*Client, error) {
	httpClient, err := uhttp.NewClient(
		ctx,
		uhttp.WithLogger(
			true,
			ctxzap.Extract(ctx),
		),
	)
	if err != nil {
		return nil, err
	}

	normalizedUrl, err := config.NormalizeCoupaURL(instanceUrl)
	if err != nil {
		return nil, err
	}

	baseUrl, err := url.Parse(normalizedUrl)
	if err != nil {
		return nil, err
	}

	coupaClient := &Client{
		baseUrl: baseUrl,
		wrapper: uhttp.NewBaseHttpClient(httpClient),
	}

	if clientId != "" && clientSecret != "" {
		coupaClient.ReadOnlyTokenSource = getTokenSource(
			ctx,
			baseUrl,
			clientId,
			clientSecret,
			ScopesReadOnly...,
		)
		coupaClient.readWriteTokenSource = getTokenSource(
			ctx,
			baseUrl,
			clientId,
			clientSecret,
			ScopesReadWrite...,
		)
	}

	return coupaClient, nil
}

func (c *Client) Query(
	ctx context.Context,
	rawQuery string,
	target interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	err := c.Initialize(ctx)
	if err != nil {
		return nil, nil, err
	}

	l := ctxzap.Extract(ctx)

	l.Debug("Querying Coupa", zap.String("query", rawQuery))

	return c.doGraphQLRequest(
		ctx,
		http.MethodPost,
		c.baseUrl.JoinPath(apiPathQuery),
		Query{Query: rawQuery},
		&target,
	)
}

func (c *Client) Initialize(ctx context.Context) error {
	logger := ctxzap.Extract(ctx)
	if c.initialized {
		logger.Debug("Coupa client already initialized")
		return nil
	}
	logger.Debug("Initializing Coupa client")

	rtoken, err := c.ReadOnlyTokenSource.Token()
	if err != nil {
		return err
	}

	rwtoken, err := c.readWriteTokenSource.Token()
	if err != nil {
		return err
	}

	c.readOnlyToken = rtoken.AccessToken
	c.readWriteToken = rwtoken.AccessToken
	c.initialized = true
	return nil
}
