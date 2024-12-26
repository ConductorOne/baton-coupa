package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

// WithBearerToken - TODO(marcos): move this function to `baton-sdk`.
func WithBearerToken(token string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token))
}

func (c *Client) doGraphQLRequest(
	ctx context.Context,
	method string,
	url *url.URL,
	payload interface{},
	target interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	l := ctxzap.Extract(ctx)

	options := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
		WithBearerToken(c.readOnlyToken),
	}
	if payload != nil {
		options = append(options, uhttp.WithJSONBody(payload))
	}

	request, err := c.wrapper.NewRequest(ctx, method, url, options...)
	if err != nil {
		return nil, nil, err
	}
	var ratelimitData v2.RateLimitDescription
	response, err := c.wrapper.Do(
		request,
		uhttp.WithRatelimitData(&ratelimitData),
	)
	if err != nil {
		return nil, &ratelimitData, err
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, &ratelimitData, err
	}

	var innerResponse innerGraphqlResponse

	if err := json.Unmarshal(bodyBytes, &innerResponse); err != nil {
		l.Error("Failed to unmarshal response body", zap.Error(err))
		return nil, nil, err
	}

	if len(innerResponse.Errors) > 0 {
		l.Error("Received errors from the server", zap.Any("errors", innerResponse.Errors))
		return nil, nil, errors.New(innerResponse.Errors[0].Message)
	}

	if innerResponse.Data != nil {
		if err := json.Unmarshal(*innerResponse.Data, target); err != nil {
			l.Error("Failed to unmarshal response data", zap.Error(err))
			return nil, nil, err
		}
	}

	return response, &ratelimitData, nil
}

// COUPA does not suport mutations for graphQL.

func (c *Client) doRestRequest(
	ctx context.Context,
	method string,
	url *url.URL,
	payload interface{},
	target interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	l := ctxzap.Extract(ctx)

	options := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
		WithBearerToken(c.readWriteToken),
	}
	if payload != nil {
		options = append(options, uhttp.WithJSONBody(payload))
	}

	request, err := c.wrapper.NewRequest(ctx, method, url, options...)
	if err != nil {
		return nil, nil, err
	}
	var ratelimitData v2.RateLimitDescription
	response, err := c.wrapper.Do(
		request,
		uhttp.WithRatelimitData(&ratelimitData),
	)
	if err != nil {
		return nil, &ratelimitData, err
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, &ratelimitData, err
	}

	if response.StatusCode == http.StatusInternalServerError {
		return response, &ratelimitData, fmt.Errorf("baton-coupa: internal server error %s", string(bodyBytes))
	}

	if response.StatusCode == http.StatusBadRequest {
		return response, &ratelimitData, fmt.Errorf("baton-coupa: bad request %s", string(bodyBytes))
	}

	if err := json.Unmarshal(bodyBytes, &target); err != nil {
		l.Error("Failed to unmarshal response body", zap.Error(err))
		return nil, nil, err
	}

	return response, &ratelimitData, nil
}
