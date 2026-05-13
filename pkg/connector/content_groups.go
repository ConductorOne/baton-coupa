package connector

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-coupa/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const contentGroupEntitlementName = "member"

type contentGroupBuilder struct {
	client *client.Client
}

func (o *contentGroupBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return contentGroupResourceType
}

func contentGroupResource(cg *client.ContentGroup, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	return resourceSdk.NewResource(
		cg.Name,
		contentGroupResourceType,
		strconv.Itoa(cg.ID),
		resourceSdk.WithParentResourceID(parentResourceID),
		resourceSdk.WithDescription(fmt.Sprintf("%s content group in Coupa", cg.Name)),
	)
}

func (o *contentGroupBuilder) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	opts resourceSdk.SyncOpAttrs,
) (
	[]*v2.Resource,
	*resourceSdk.SyncOpResults,
	error,
) {
	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	var target client.ContentGroupsQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.ContentGroupsQuery(opts.PageToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resourceSdk.SyncOpResults{Annotations: outputAnnotations}, err
	}
	defer response.Body.Close()

	lastId := ""
	for _, cg := range target.ContentGroups {
		resource, err := contentGroupResource(cg, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		outputResources = append(outputResources, resource)
		lastId = strconv.Itoa(cg.ID)
	}

	return outputResources, &resourceSdk.SyncOpResults{NextPageToken: lastId, Annotations: outputAnnotations}, nil
}

// Entitlements returns nothing — entitlements are declared statically via StaticEntitlements.
func (o *contentGroupBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ resourceSdk.SyncOpAttrs) ([]*v2.Entitlement, *resourceSdk.SyncOpResults, error) {
	return nil, nil, nil
}

// StaticEntitlements declares the "member" entitlement once for the content_group resource type.
// The SDK stamps this template onto every content group resource during sync.
func (o *contentGroupBuilder) StaticEntitlements(_ context.Context, _ resourceSdk.SyncOpAttrs) ([]*v2.Entitlement, *resourceSdk.SyncOpResults, error) {
	return []*v2.Entitlement{
		v2.Entitlement_builder{
			Slug:        contentGroupEntitlementName,
			DisplayName: "Member",
			Description: "Member of content group in Coupa",
			Purpose:     v2.Entitlement_PURPOSE_VALUE_ASSIGNMENT,
			GrantableTo: []*v2.ResourceType{userResourceType},
		}.Build(),
	}, nil, nil
}

// Grants is a no-op — content group membership grants are generated from the user
// builder's Grants() method (one call per user) to avoid the O(content_groups × user_pages)
// cost of scanning all users for each group. This method is also skipped by the
// SkipEntitlementsAndGrants annotation on contentGroupResourceType.
func (o *contentGroupBuilder) Grants(_ context.Context, _ *v2.Resource, _ resourceSdk.SyncOpAttrs) ([]*v2.Grant, *resourceSdk.SyncOpResults, error) {
	return nil, nil, nil
}

func (o *contentGroupBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	contentGroupIdToAdd, err := strconv.Atoi(entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	userId, err := strconv.Atoi(principal.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	user, err := o.getUserContentGroups(ctx, userId)
	if err != nil {
		return nil, nil, err
	}

	for _, cg := range user.ContentGroups {
		if cg.Id == contentGroupIdToAdd {
			return nil, annotations.New(&v2.GrantAlreadyExists{}), nil
		}
	}

	newIDs := make([]int, 0, len(user.ContentGroups)+1)
	for _, cg := range user.ContentGroups {
		newIDs = append(newIDs, cg.Id)
	}
	newIDs = append(newIDs, contentGroupIdToAdd)

	response, _, err := o.client.SetContentGroups(ctx, userId, newIDs)
	if err != nil {
		return nil, nil, err
	}

	if len(response.ContentGroups) != len(newIDs) {
		return nil, nil, errors.New("baton-coupa: content group was not added to user")
	}

	newGrant := grant.NewGrant(
		entitlement.Resource,
		contentGroupEntitlementName,
		&v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     strconv.Itoa(userId),
		},
	)

	return []*v2.Grant{newGrant}, nil, nil
}

// Revoke removes a content group from a user.
// Revoking, as with account groups, is a two-step process where you first clear all content
// groups from the user and then put back the desired ones. This is required because the
// Coupa PUT /api/users/:id endpoint replaces the entire content-groups array.
func (o *contentGroupBuilder) Revoke(ctx context.Context, g *v2.Grant) (annotations.Annotations, error) {
	logger := ctxzap.Extract(ctx)
	if g.Principal.Id.ResourceType != userResourceType.Id {
		return nil, fmt.Errorf("baton-coupa: principal resource type is not %s", userResourceType.Id)
	}

	contentGroupIdToRemove, err := strconv.Atoi(g.Entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, err
	}

	userId, err := strconv.Atoi(g.Principal.Id.Resource)
	if err != nil {
		return nil, err
	}

	user, err := o.getUserContentGroups(ctx, userId)
	if err != nil {
		return nil, err
	}

	newIDs := make([]int, 0, len(user.ContentGroups))
	found := false
	for _, cg := range user.ContentGroups {
		if cg.Id == contentGroupIdToRemove {
			found = true
			continue
		}
		newIDs = append(newIDs, cg.Id)
	}
	if !found {
		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	_, _, err = o.client.SetContentGroups(ctx, userId, make([]int, 0))
	if err != nil {
		return nil, err
	}

	userResponse, _, err := o.client.SetContentGroups(ctx, userId, newIDs)
	if err != nil {
		logger.Error("baton-coupa: error setting content groups", zap.Error(err), zap.Ints("content_groups", newIDs))
		return nil, err
	}

	if len(userResponse.ContentGroups) != len(newIDs) {
		return nil, errors.New("baton-coupa: content group was not removed from user")
	}

	return nil, nil
}

func (o *contentGroupBuilder) getUserContentGroups(ctx context.Context, userId int) (*client.UserWithContentGroups, error) {
	var target client.UserContentGroupsQueryResponse
	response, _, err := o.client.Query(
		ctx,
		client.GetUserContentGroupsByID(userId),
		&target,
	)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if len(target.Users) == 0 {
		return nil, fmt.Errorf("baton-coupa: user %d not found", userId)
	}
	if len(target.Users) > 1 {
		return nil, fmt.Errorf("baton-coupa: multiple users found for id %d", userId)
	}

	return &target.Users[0], nil
}

func newContentGroupBuilder(_ context.Context, client *client.Client) *contentGroupBuilder {
	return &contentGroupBuilder{
		client: client,
	}
}
