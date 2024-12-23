package connector

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
	for _, role := range target.Roles {
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

	for _, user := range target.Users {
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

func (o *roleBuilder) Grant(ctx context.Context, resource *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	roleIdToAdd, err := strconv.Atoi(entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	userId, err := strconv.Atoi(resource.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	var target client.UserRolesResponse
	response, _, err := o.client.Query(
		ctx,
		client.GetUserRoles(userId),
		&target,
	)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	if len(target.Users) == 0 {
		return nil, nil, errors.New("baton-coupa: user not found")
	}

	if len(target.Users) > 1 {
		return nil, nil, fmt.Errorf("baton-coupa: multiple users found for id %d", userId)
	}

	user := target.Users[0]

	for _, role := range user.Roles {
		if role.ID == roleIdToAdd {
			return []*v2.Grant{}, annotations.New(&v2.GrantAlreadyExists{}), nil
		}
	}

	rolesToAdd := make([]int, 0)

	for _, role := range user.Roles {
		rolesToAdd = append(rolesToAdd, role.ID)
	}

	rolesToAdd = append(rolesToAdd, roleIdToAdd)

	userResponse, _, err := o.client.SetRoles(ctx, userId, rolesToAdd)
	if err != nil {
		return nil, nil, err
	}

	if len(userResponse.Roles) != len(rolesToAdd) {
		return nil, nil, errors.New("baton-coupa: roles not set")
	}

	newGrant := grant.NewGrant(
		resource,
		roleMemberEntitlementName,
		&v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     strconv.Itoa(user.Id),
		},
	)

	return []*v2.Grant{newGrant}, nil, nil
}

func (o *roleBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if grant.Principal.Id.ResourceType != userResourceType.Id {
		return nil, fmt.Errorf("baton-coupa: principal resource type is not %s", userResourceType.Id)
	}

	roleIdToRemove, err := strconv.Atoi(grant.Entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, err
	}

	userId, err := strconv.Atoi(grant.Principal.Id.Resource)
	if err != nil {
		return nil, err
	}

	var target client.UserRolesResponse
	response, _, err := o.client.Query(
		ctx,
		client.GetUserRoles(userId),
		&target,
	)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if len(target.Users) == 0 {
		return nil, errors.New("baton-coupa: user not found")
	}

	if len(target.Users) > 1 {
		return nil, fmt.Errorf("baton-coupa: multiple users found for id %d", userId)
	}

	user := target.Users[0]

	index := slices.IndexFunc(user.Roles, func(c client.Role) bool {
		return c.ID == roleIdToRemove
	})
	if index < 0 {
		l.Info(
			"baton-coupa: scope not found in user",
		)

		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	if index == 0 {
		user.Roles = user.Roles[1:]
	} else {
		user.Roles = append(user.Roles[:index], user.Roles[index+1:]...)
	}

	newRoles := make([]int, 0)
	for _, role := range user.Roles {
		newRoles = append(newRoles, role.ID)
	}

	userResponse, _, err := o.client.SetRoles(ctx, userId, newRoles)
	if err != nil {
		return nil, err
	}

	if len(userResponse.Roles) != len(newRoles) {
		return nil, errors.New("baton-coupa: roles was not set")
	}

	return nil, nil
}

func newRoleBuilder(ctx context.Context, client *client.Client) *roleBuilder {
	return &roleBuilder{
		client: client,
	}
}
