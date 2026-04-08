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
//nolint:revive // long URL
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

	request := UpdateUserRequest{
		Active: active,
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

// CreateUser creates a new user in Coupa.
// https://compass.coupa.com/en-us/products/product-documentation/integration-technical-documentation/the-coupa-core-api/resources/reference-data-resources/users-api-(users)
func (c *Client) CreateUser(
	ctx context.Context,
	login string,
	email string,
	firstname string,
	lastname string,
) (
	*CreateUserResponse,
	*v2.RateLimitDescription,
	error,
) {
	l := ctxzap.Extract(ctx)

	err := c.Initialize(ctx)
	if err != nil {
		l.Error("Failed to initialize client", zap.Error(err))
		return nil, nil, err
	}

	request := CreateUserRequest{
		Login:     login,
		Email:     email,
		Firstname: firstname,
		Lastname:  lastname,
		Active:    true,
	}

	var userResponse CreateUserResponse

	url := c.baseUrl.JoinPath(createUserPath)

	_, rateLimit, err := c.doRestRequest(
		ctx,
		http.MethodPost,
		url,
		request,
		&userResponse,
	)

	if err != nil {
		l.Error("CreateUser request failed", zap.Error(err))
		return nil, rateLimit, err
	}

	return &userResponse, rateLimit, nil
}
