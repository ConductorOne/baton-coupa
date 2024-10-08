package connector

import (
	"context"
	"strconv"

	"github.com/conductorone/baton-coupa/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type userBuilder struct {
	client *client.Client
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

// Create a new connector resource for a Coupa user.
func userResource(user *client.User, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	status := v2.UserTrait_Status_STATUS_DISABLED
	if user.Active {
		status = v2.UserTrait_Status_STATUS_ENABLED
	}

	return resourceSdk.NewUserResource(
		user.Fullname,
		userResourceType,
		user.ID,
		[]resourceSdk.UserTraitOption{
			resourceSdk.WithEmail(user.Email, true),
			resourceSdk.WithStatus(status),
			resourceSdk.WithUserProfile(
				map[string]interface{}{
					"id":        user.ID,
					"email":     user.Email,
					"full_name": user.Fullname,
					"active":    user.Active,
				}),
			resourceSdk.WithUserLogin(user.Email),
		},
		resourceSdk.WithParentResourceID(parentResourceID),
	)
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	pToken *pagination.Token,
) (
	[]*v2.Resource,
	string,
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)
	logger.Debug("Starting Users List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	var target client.UsersQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.AllUsersQuery(pToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	defer response.Body.Close()

	lastId := ""
	for _, user := range target.Data.Users {
		resource, err := userResource(user, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, resource)
		lastId = strconv.Itoa(user.ID)
	}

	return outputResources, lastId, outputAnnotations, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(
	_ context.Context,
	_ *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Entitlement,
	string,
	annotations.Annotations,
	error,
) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(
	_ context.Context,
	_ *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Grant,
	string,
	annotations.Annotations,
	error,
) {
	return nil, "", nil, nil
}

func newUserBuilder(ctx context.Context, client *client.Client) *userBuilder {
	return &userBuilder{
		client: client,
	}
}
