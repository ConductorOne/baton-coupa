package connector

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/conductorone/baton-sdk/pkg/types/grant"

	"github.com/conductorone/baton-coupa/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const groupMemberEntitlementName = "member"

type groupBuilder struct {
	client *client.Client
}

func (o *groupBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return groupResourceType
}

func groupResource(group *client.Group, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	description := fmt.Sprintf("%s group in Coupa", group.Name)
	if group.Description != nil && *group.Description != "" {
		description = *group.Description
	}

	return resourceSdk.NewGroupResource(
		group.Name,
		groupResourceType,
		group.ID,
		[]resourceSdk.GroupTraitOption{},
		resourceSdk.WithParentResourceID(parentResourceID),
		resourceSdk.WithDescription(description),
	)
}

func (o *groupBuilder) List(
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
	logger.Debug("Starting Groups List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	var target client.GroupsQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.GroupsQuery(pToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	defer response.Body.Close()

	lastId := ""
	for _, group := range target.UserGroups {
		resource, err := groupResource(group, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, resource)
		lastId = strconv.Itoa(group.ID)
	}

	return outputResources, lastId, outputAnnotations, nil
}

func (o *groupBuilder) Entitlements(
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
			groupMemberEntitlementName,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDisplayName(
				fmt.Sprintf("%s Group", resource.DisplayName),
			),
			entitlement.WithDescription(
				fmt.Sprintf("%s group in Coupa", resource.DisplayName),
			),
		),
	}, "", nil, nil
}

func (o *groupBuilder) Grants(
	ctx context.Context,
	resource *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Grant,
	string,
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)

	groupId := resource.Id.Resource

	logger.Debug(
		"Starting Groups Grants",
		zap.String("group_id", groupId),
	)

	outputGrants := make([]*v2.Grant, 0)
	var outputAnnotations annotations.Annotations

	var target client.GroupMembersQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.GroupMembersQuery(groupId),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	defer response.Body.Close()

	// Every group must have _one_ user group.
	userGroups := target.UserGroups
	if len(userGroups) != 0 {
		for _, membership := range userGroups[0].Users {
			outputGrants = append(
				outputGrants,
				grant.NewGrant(
					resource,
					groupMemberEntitlementName,
					&v2.ResourceId{
						ResourceType: userResourceType.Id,
						Resource:     strconv.Itoa(membership.Id),
					},
				),
			)
		}
	}

	return outputGrants, "", outputAnnotations, nil
}

func (o *groupBuilder) Grant(ctx context.Context, resource *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	groupIdToAdd, err := strconv.Atoi(entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	userId, err := strconv.Atoi(resource.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	user, err := o.getUserGroupsResponse(ctx, userId)
	if err != nil {
		return nil, nil, err
	}

	for _, group := range user.Group {
		if group.ID == groupIdToAdd {
			return []*v2.Grant{}, annotations.New(&v2.GrantAlreadyExists{}), nil
		}
	}

	newGroupIDs := make([]int, 0)

	for _, group := range user.Group {
		newGroupIDs = append(newGroupIDs, group.ID)
	}

	newGroupIDs = append(newGroupIDs, groupIdToAdd)

	groups, _, err := o.client.SetUserGroups(ctx, userId, newGroupIDs)
	if err != nil {
		return nil, nil, err
	}

	if len(groups.Group) != len(newGroupIDs) {
		l.Debug(
			"baton-coupa: group not added to user",
			zap.Any("response", groups.Group),
		)
		return nil, nil, errors.New("baton-coupa: failed to add group to user")
	}

	newGrant := grant.NewGrant(
		resource,
		groupMemberEntitlementName,
		&v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     strconv.Itoa(userId),
		},
	)

	return []*v2.Grant{newGrant}, nil, nil
}

func (o *groupBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if grant.Principal.Id.ResourceType != userResourceType.Id {
		return nil, fmt.Errorf("baton-coupa: principal resource type is not %s", userResourceType.Id)
	}

	groupIdToRemove, err := strconv.Atoi(grant.Entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, err
	}

	userId, err := strconv.Atoi(grant.Principal.Id.Resource)
	if err != nil {
		return nil, err
	}

	user, err := o.getUserGroupsResponse(ctx, userId)
	if err != nil {
		return nil, err
	}

	index := slices.IndexFunc(user.Group, func(c client.Group) bool {
		return c.ID == groupIdToRemove
	})
	if index < 0 {
		l.Info(
			"baton-coupa: group not found in user",
		)

		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	if index == 0 {
		user.Group = user.Group[1:]
	} else {
		user.Group = append(user.Group[:index], user.Group[index+1:]...)
	}

	newGroups := make([]int, 0)
	for _, group := range user.Group {
		newGroups = append(newGroups, group.ID)
	}

	_, _, err = o.client.SetUserGroups(ctx, userId, make([]int, 0))
	if err != nil {
		return nil, err
	}

	userResponse, _, err := o.client.SetUserGroups(ctx, userId, newGroups)
	if err != nil {
		l.Error(
			"baton-coupa: error setting groups",
			zap.Error(err),
			zap.Ints("groups", newGroups),
		)
		return nil, err
	}

	if len(userResponse.Group) != len(newGroups) {
		return nil, errors.New("baton-coupa: group was not removed")
	}

	return nil, nil
}

func (o *groupBuilder) getUserGroupsResponse(ctx context.Context, userId int) (*client.UserGroups, error) {
	var target client.UserGroupsResponse
	response, _, err := o.client.Query(
		ctx,
		client.GetUserGroups(userId),
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

	return &target.Users[0], nil
}

func newGroupBuilder(ctx context.Context, client *client.Client) *groupBuilder {
	return &groupBuilder{
		client: client,
	}
}
