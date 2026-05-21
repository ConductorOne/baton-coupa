package client

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// SetContentGroups sets the content groups (business_groups) for a user.
// Content groups in Coupa are also known as business groups in the API.
// https://compass.coupa.com/en-us/products/product-documentation/integration-technical-documentation/the-coupa-core-api/resources/reference-data-resources/content-groups-api-(business_groups)
func (c *Client) SetContentGroups(
	ctx context.Context,
	userId int,
	contentGroupIDs []int,
) (
	*UserContentGroupsApiResponse,
	*v2.RateLimitDescription,
	error,
) {
	err := c.Initialize(ctx)
	if err != nil {
		return nil, nil, err
	}

	request := struct {
		ContentGroups []ResourceId `json:"content-groups"`
	}{}

	if len(contentGroupIDs) == 0 {
		request.ContentGroups = nil
	} else {
		for _, id := range contentGroupIDs {
			request.ContentGroups = append(request.ContentGroups, ResourceId{Id: id})
		}
	}

	var userResponse UserContentGroupsApiResponse

	_, rateLimit, err := c.doRestRequest(
		ctx,
		http.MethodPut,
		c.baseUrl.JoinPath(fmt.Sprintf(setContentGroupPath, userId)),
		request,
		&userResponse,
	)
	if err != nil {
		return nil, rateLimit, err
	}

	return &userResponse, rateLimit, nil
}
