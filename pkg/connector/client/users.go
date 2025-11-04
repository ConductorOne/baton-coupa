package client

import (
	"context"
	"fmt"
	"net/http"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// UpdateUser updates the user's active status.
// https://compass.coupa.com/en-us/products/total-spend-management-platform/integration-playbooks-and-resources/other-integration-playbooks/erp-integration-adapters/integration-scenarios/1.-user-integration-scenarios-(optional)/1.4-deactivate-a-user
func (c *Client) UpdateUser(
	ctx context.Context,
	userId int,
	active bool,
) (
	*UpdateUserResponse,
	*v2.RateLimitDescription,
	error,
) {
	l := ctxzap.Extract(ctx)

	err := c.Initialize(ctx)
	if err != nil {
		l.Error("Failed to initialize client", zap.Error(err))
		return nil, nil, err
	}

	request := map[string]interface{}{
		"active": active,
	}

	var userResponse UpdateUserResponse

	url := c.baseUrl.JoinPath(fmt.Sprintf(updateUserPath, userId))

	response, rateLimit, err := c.doRestRequest(
		ctx,
		http.MethodPut,
		url,
		request,
		&userResponse,
	)

	if err != nil {
		l.Error("UpdateUser request failed", zap.Error(err))
		return nil, rateLimit, err
	}

	defer response.Body.Close()

	return &userResponse, rateLimit, nil
}
