package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

// WithBearerToken - TODO(marcos): move this function to `baton-sdk`.
func WithBearerToken(token string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token))
}

func (c *Client) doRequest(
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
	options := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
		WithBearerToken(c.bearerToken),
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

	if err := json.Unmarshal(bodyBytes, &target); err != nil {
		return nil, nil, err
	}

	return response, &ratelimitData, nil
}