package client

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// SetRoles sets the roles for a user.
// https://compass.coupa.com/en-us/products/product-documentation/integration-technical-documentation/the-coupa-core-api/resources/reference-data-resources/users-api-(users)
func (c *Client) SetRoles(
	ctx context.Context,
	userId int,
	roleNames []int,
) (
	*UserRolesPutResponse,
	*v2.RateLimitDescription,
	error,
) {
	err := c.Initialize(ctx)
	if err != nil {
		return nil, nil, err
	}

	request := struct {
		Roles []ResourceId `json:"roles"`
	}{}

	if len(roleNames) == 0 {
		request.Roles = nil
	} else {
		for _, role := range roleNames {
			request.Roles = append(request.Roles, struct {
				Id int `json:"id"`
			}{Id: role})
		}
	}

	var userResponse UserRolesPutResponse

	resonse, rateLimit, err := c.doRestRequest(
		ctx,
		http.MethodPut,
		c.baseUrl.JoinPath(fmt.Sprintf(setRolesPath, userId)),
		request,
		&userResponse,
	)

	if err != nil {
		return nil, rateLimit, err
	}

	defer resonse.Body.Close()

	return &userResponse, rateLimit, nil
}
