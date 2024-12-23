package connector

import (
	"context"
	"fmt"
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

func newGroupBuilder(ctx context.Context, client *client.Client) *groupBuilder {
	return &groupBuilder{
		client: client,
	}
}
