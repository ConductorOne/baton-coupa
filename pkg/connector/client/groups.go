package client

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// SetUserGroups sets the roles for a user.
// https://compass.coupa.com/en-us/products/product-documentation/integration-technical-documentation/the-coupa-core-api/resources/reference-data-resources/users-api-(users)
func (c *Client) SetUserGroups(
	ctx context.Context,
	userId int,
	groupIDs []int,
) (
	*UserGroupsApiResponse,
	*v2.RateLimitDescription,
	error,
) {
	err := c.Initialize(ctx)
	if err != nil {
		return nil, nil, err
	}

	request := struct {
		UserGroups []ResourceId `json:"user-groups"`
	}{}

	if len(groupIDs) == 0 {
		request.UserGroups = nil
	} else {
		for _, groupId := range groupIDs {
			request.UserGroups = append(request.UserGroups, ResourceId{Id: groupId})
		}
	}

	var userResponse UserGroupsApiResponse

	resonse, rateLimit, err := c.doRestRequest(
		ctx,
		http.MethodPut,
		c.baseUrl.JoinPath(fmt.Sprintf(setGroupPath, userId)),
		request,
		&userResponse,
	)

	if err != nil {
		return nil, rateLimit, err
	}
	defer resonse.Body.Close()

	return &userResponse, rateLimit, nil
}
