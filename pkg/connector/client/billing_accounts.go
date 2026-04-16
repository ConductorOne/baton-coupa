package client

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// SetUserAccounts sets the billing account assignments for a user.
// This follows the same pattern as SetRoles and SetUserGroups.
func (c *Client) SetUserAccounts(
	ctx context.Context,
	userId int,
	accountIDs []int,
) (
	*UserAccountsPutResponse,
	*v2.RateLimitDescription,
	error,
) {
	err := c.Initialize(ctx)
	if err != nil {
		return nil, nil, err
	}

	request := struct {
		Accounts []ResourceId `json:"account"`
	}{}

	if len(accountIDs) == 0 {
		request.Accounts = nil
	} else {
		for _, accountID := range accountIDs {
			request.Accounts = append(request.Accounts, ResourceId{Id: accountID})
		}
	}

	var userResponse UserAccountsPutResponse

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
