package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-coupa/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const roleMemberEntitlementName = "member"

type roleBuilder struct {
	client *client.Client
}

func (o *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return roleResourceType
}

func roleResource(role *client.Role, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	description := fmt.Sprintf("%s role in Coupa", role.Name)
	if role.Description != nil && *role.Description != "" {
		description = *role.Description
	}

	return resourceSdk.NewRoleResource(
		role.Name,
		roleResourceType,
		role.ID,
		[]resourceSdk.RoleTraitOption{},
		resourceSdk.WithParentResourceID(parentResourceID),
		resourceSdk.WithDescription(description),
	)
}

func (o *roleBuilder) List(
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
	logger.Debug("Starting Roles List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	var target client.RolesQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.RolesQuery(pToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	defer response.Body.Close()

	lastId := ""
	for _, role := range target.Data.Roles {
		resource, err := roleResource(role, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, resource)
		lastId = strconv.Itoa(role.ID)
	}

	return outputResources, lastId, outputAnnotations, nil
}

func (o *roleBuilder) Entitlements(
	_ context.Context,
	resource *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Entitlement,
	string,
	annotations.Annotations,
	error,
) {
	return []*v2.Entitlement{
		entitlement.NewAssignmentEntitlement(
			resource,
			roleMemberEntitlementName,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDisplayName(
				fmt.Sprintf("%s Role", resource.DisplayName),
			),
			entitlement.WithDescription(
				fmt.Sprintf("%s role in Coupa", resource.DisplayName),
			),
		),
	}, "", nil, nil
}

func (o *roleBuilder) Grants(
	ctx context.Context,
	resource *v2.Resource,
	pToken *pagination.Token,
) (
	[]*v2.Grant,
	string,
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)

	roleId := resource.Id.Resource

	logger.Debug(
		"Starting Roles Grants",
		zap.String("role_id", roleId),
		zap.String("token", pToken.Token),
	)

	outputGrants := make([]*v2.Grant, 0)
	var outputAnnotations annotations.Annotations

	var target client.RoleGrantsQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.RoleGrantQuery(roleId, pToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	defer response.Body.Close()

	for _, user := range target.Data.Users {
		outputGrants = append(
			outputGrants,
			grant.NewGrant(
				resource,
				roleMemberEntitlementName,
				&v2.ResourceId{
					ResourceType: userResourceType.Id,
					Resource:     strconv.Itoa(user.Id),
				},
			),
		)
	}

	return outputGrants, "", outputAnnotations, nil
}

func newRoleBuilder(ctx context.Context, client *client.Client) *roleBuilder {
	return &roleBuilder{
		client: client,
	}
}
