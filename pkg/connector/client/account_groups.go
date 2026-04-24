package client

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// SetAccountGroups sets the account groups for a user.
// https://docs.coupa.com/en/developer-documentation/the-coupa-core-api/resources/reference-data-resources/users-api-users
func (c *Client) SetAccountGroups(
	ctx context.Context,
	userId int,
	accountGroupIDs []int,
) (
	*UserAccountGroupsApiResponse,
	*v2.RateLimitDescription,
	error,
) {
	err := c.Initialize(ctx)
	if err != nil {
		return nil, nil, err
	}

	request := struct {
		AccountGroups []ResourceId `json:"account-groups"`
	}{}

	if len(accountGroupIDs) == 0 {
		request.AccountGroups = nil
	} else {
		for _, id := range accountGroupIDs {
			request.AccountGroups = append(request.AccountGroups, ResourceId{Id: id})
		}
	}

	var userResponse UserAccountGroupsApiResponse

	response, rateLimit, err := c.doRestRequest(
		ctx,
		http.MethodPut,
		c.baseUrl.JoinPath(fmt.Sprintf(setAccountGroupPath, userId)),
		request,
		&userResponse,
	)
	if err != nil {
		return nil, rateLimit, err
	}
	defer response.Body.Close()

	return &userResponse, rateLimit, nil
}
