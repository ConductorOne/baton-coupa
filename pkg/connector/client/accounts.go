package client

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// SetUserAccount sets the default account for a user.
// https://compass.coupa.com/en-us/products/product-documentation/integration-technical-documentation/the-coupa-core-api/resources/reference-data-resources/users-api-(users)
func (c *Client) SetUserAccount(
	ctx context.Context,
	userId int,
	accountID *int,
) (
	*UserAccountPutResponse,
	*v2.RateLimitDescription,
	error,
) {
	err := c.Initialize(ctx)
	if err != nil {
		return nil, nil, err
	}

	request := struct {
		DefaultAccount *ResourceId `json:"default-account"`
	}{}

	if accountID != nil {
		request.DefaultAccount = &ResourceId{Id: *accountID}
	}

	var userResponse UserAccountPutResponse

	response, rateLimit, err := c.doRestRequest(
		ctx,
		http.MethodPut,
		c.baseUrl.JoinPath(fmt.Sprintf(setAccountPath, userId)),
		request,
		&userResponse,
	)
	if err != nil {
		return nil, rateLimit, err
	}
	defer response.Body.Close()

	return &userResponse, rateLimit, nil
}
