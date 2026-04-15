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

const accountGroupEntitlementName = "member"

type accountGroupBuilder struct {
	client *client.Client
}

func (o *accountGroupBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return accountGroupResourceType
}

func accountGroupResource(ag *client.AccountGroup, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	return resourceSdk.NewResource(
		ag.Name,
		accountGroupResourceType,
		strconv.Itoa(ag.ID),
		resourceSdk.WithParentResourceID(parentResourceID),
		resourceSdk.WithDescription(fmt.Sprintf("%s account group in Coupa", ag.Name)),
	)
}

func (o *accountGroupBuilder) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	opts resourceSdk.SyncOpAttrs,
) (
	[]*v2.Resource,
	*resourceSdk.SyncOpResults,
	error,
) {
	logger := ctxzap.Extract(ctx)
	logger.Debug("Starting Account Groups List", zap.String("token", opts.PageToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	var target client.AccountGroupsQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.AccountGroupsQuery(opts.PageToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resourceSdk.SyncOpResults{Annotations: outputAnnotations}, err
	}
	defer response.Body.Close()

	lastId := ""
	for _, ag := range target.AccountGroups {
		resource, err := accountGroupResource(ag, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		outputResources = append(outputResources, resource)
		lastId = strconv.Itoa(ag.ID)
	}

	return outputResources, &resourceSdk.SyncOpResults{NextPageToken: lastId, Annotations: outputAnnotations}, nil
}

// Entitlements returns nothing — entitlements are declared statically via StaticEntitlements.
func (o *accountGroupBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ resourceSdk.SyncOpAttrs) ([]*v2.Entitlement, *resourceSdk.SyncOpResults, error) {
	return nil, nil, nil
}

// StaticEntitlements declares the "member" entitlement once for the account_group resource type.
// The SDK stamps this template onto every account group resource during sync.
func (o *accountGroupBuilder) StaticEntitlements(_ context.Context, _ resourceSdk.SyncOpAttrs) ([]*v2.Entitlement, *resourceSdk.SyncOpResults, error) {
	return []*v2.Entitlement{
		v2.Entitlement_builder{
			Slug:        accountGroupEntitlementName,
			DisplayName: "Member",
			Description: "Member of account group in Coupa",
			Purpose:     v2.Entitlement_PURPOSE_VALUE_ASSIGNMENT,
			GrantableTo: []*v2.ResourceType{userResourceType},
		}.Build(),
	}, nil, nil
}

func (o *accountGroupBuilder) Grants(
	ctx context.Context,
	resource *v2.Resource,
	opts resourceSdk.SyncOpAttrs,
) (
	[]*v2.Grant,
	*resourceSdk.SyncOpResults,
	error,
) {
	logger := ctxzap.Extract(ctx)

	accountGroupId := resource.Id.Resource

	logger.Debug(
		"Starting Account Groups Grants",
		zap.String("account_group_id", accountGroupId),
		zap.String("token", opts.PageToken.Token),
	)

	outputGrants := make([]*v2.Grant, 0)
	var outputAnnotations annotations.Annotations

	// Coupa's GraphQL does not support filtering users by accountGroups[id].
	// We page through all users selecting their accountGroups and filter client-side.
	var target client.UserAccountGroupsQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.UserAccountGroupsQuery(opts.PageToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resourceSdk.SyncOpResults{Annotations: outputAnnotations}, err
	}
	defer response.Body.Close()

	lastId := ""
	for _, user := range target.Users {
		for _, ag := range user.AccountGroups {
			if strconv.Itoa(ag.Id) == accountGroupId {
				outputGrants = append(
					outputGrants,
					grant.NewGrant(
						resource,
						accountGroupEntitlementName,
						&v2.ResourceId{
							ResourceType: userResourceType.Id,
							Resource:     strconv.Itoa(user.Id),
						},
					),
				)
				break
			}
		}
		lastId = strconv.Itoa(user.Id)
	}

	return outputGrants, &resourceSdk.SyncOpResults{NextPageToken: lastId, Annotations: outputAnnotations}, nil
}

func (o *accountGroupBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	accountGroupIdToAdd, err := strconv.Atoi(entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	userId, err := strconv.Atoi(principal.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	user, err := o.getUserAccountGroups(ctx, userId)
	if err != nil {
		return nil, nil, err
	}

	for _, ag := range user.AccountGroups {
		if ag.Id == accountGroupIdToAdd {
			return []*v2.Grant{}, annotations.New(&v2.GrantAlreadyExists{}), nil
		}
	}

	newIDs := make([]int, 0, len(user.AccountGroups)+1)
	for _, ag := range user.AccountGroups {
		newIDs = append(newIDs, ag.Id)
	}
	newIDs = append(newIDs, accountGroupIdToAdd)

	response, _, err := o.client.SetAccountGroups(ctx, userId, newIDs)
	if err != nil {
		return nil, nil, err
	}

	if len(response.AccountGroups) != len(newIDs) {
		return nil, nil, errors.New("baton-coupa: account group was not added to user")
	}

	newGrant := grant.NewGrant(
		entitlement.Resource,
		accountGroupEntitlementName,
		&v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     strconv.Itoa(userId),
		},
	)

	return []*v2.Grant{newGrant}, nil, nil
}

func (o *accountGroupBuilder) Revoke(ctx context.Context, g *v2.Grant) (annotations.Annotations, error) {
	logger := ctxzap.Extract(ctx)
	if g.Principal.Id.ResourceType != userResourceType.Id {
		return nil, fmt.Errorf("baton-coupa: principal resource type is not %s", userResourceType.Id)
	}

	accountGroupIdToRemove, err := strconv.Atoi(g.Entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, err
	}

	userId, err := strconv.Atoi(g.Principal.Id.Resource)
	if err != nil {
		return nil, err
	}

	user, err := o.getUserAccountGroups(ctx, userId)
	if err != nil {
		return nil, err
	}

	newIDs := make([]int, 0, len(user.AccountGroups))
	found := false
	for _, ag := range user.AccountGroups {
		if ag.Id == accountGroupIdToRemove {
			found = true
			continue
		}
		newIDs = append(newIDs, ag.Id)
	}
	if !found {
		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	_, _, err = o.client.SetAccountGroups(ctx, userId, make([]int, 0))
	if err != nil {
		return nil, err
	}

	userResponse, _, err := o.client.SetAccountGroups(ctx, userId, newIDs)
	if err != nil {
		logger.Error("baton-coupa: error setting account groups", zap.Error(err), zap.Ints("account_groups", newIDs))
		return nil, err
	}

	if len(userResponse.AccountGroups) != len(newIDs) {
		return nil, errors.New("baton-coupa: account group was not removed from user")
	}

	return nil, nil
}

func (o *accountGroupBuilder) getUserAccountGroups(ctx context.Context, userId int) (*client.UserWithAccountGroups, error) {
	var target client.UserAccountGroupsQueryResponse
	response, _, err := o.client.Query(
		ctx,
		client.GetUserAccountGroupsByID(userId),
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

func newAccountGroupBuilder(_ context.Context, client *client.Client) *accountGroupBuilder {
	return &accountGroupBuilder{
		client: client,
	}
}
