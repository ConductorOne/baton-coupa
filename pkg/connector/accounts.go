package connector

import (
	"context"
	"errors"
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

const accountAssignedEntitlementName = "assigned"

type accountBuilder struct {
	client *client.Client
}

func (o *accountBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return accountResourceType
}

func accountResource(account *client.Account, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	displayName := account.Name
	if account.Code != "" {
		displayName = fmt.Sprintf("%s (%s)", account.Name, account.Code)
	}

	description := fmt.Sprintf("%s billing account in Coupa", account.Name)

	return resourceSdk.NewResource(
		displayName,
		accountResourceType,
		account.ID,
		resourceSdk.WithParentResourceID(parentResourceID),
		resourceSdk.WithDescription(description),
	)
}

func (o *accountBuilder) List(
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
	logger.Debug("Starting Accounts List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	var target client.AccountsQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.AccountsQuery(pToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		// The Coupa instance may not expose accounts via GraphQL or the
		// OAuth client may lack core.accounting.read scope. Log the
		// error and return an empty list so the rest of the sync
		// continues.
		logger.Warn(
			"baton-coupa: failed to list accounts; billing account sync may require core.accounting.read scope",
			zap.Error(err),
		)
		return nil, "", outputAnnotations, nil
	}
	defer response.Body.Close()

	lastId := ""
	for _, account := range target.Accounts {
		lastId = strconv.Itoa(account.ID)

		if !account.Active {
			continue
		}

		resource, err := accountResource(account, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, resource)
	}

	return outputResources, lastId, outputAnnotations, nil
}

func (o *accountBuilder) Entitlements(
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
			accountAssignedEntitlementName,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDisplayName(
				fmt.Sprintf("%s Billing Account", resource.DisplayName),
			),
			entitlement.WithDescription(
				fmt.Sprintf("%s billing account in Coupa", resource.DisplayName),
			),
		),
	}, "", nil, nil
}

func (o *accountBuilder) Grants(
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

	accountId := resource.Id.Resource

	logger.Debug(
		"Starting Accounts Grants",
		zap.String("account_id", accountId),
		zap.String("token", pToken.Token),
	)

	outputGrants := make([]*v2.Grant, 0)
	var outputAnnotations annotations.Annotations

	var target client.AccountGrantsQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.AccountGrantQuery(accountId, pToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		logger.Warn(
			"baton-coupa: failed to query account grants",
			zap.String("account_id", accountId),
			zap.Error(err),
		)
		return nil, "", outputAnnotations, nil
	}
	defer response.Body.Close()

	lastId := ""
	for _, user := range target.Users {
		outputGrants = append(
			outputGrants,
			grant.NewGrant(
				resource,
				accountAssignedEntitlementName,
				&v2.ResourceId{
					ResourceType: userResourceType.Id,
					Resource:     strconv.Itoa(user.Id),
				},
			),
		)
		lastId = strconv.Itoa(user.Id)
	}

	return outputGrants, lastId, outputAnnotations, nil
}

func (o *accountBuilder) Grant(ctx context.Context, resource *v2.Resource, ent *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	accountIdToSet, err := strconv.Atoi(ent.Resource.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	userId, err := strconv.Atoi(resource.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	user, err := o.getUserAccounts(ctx, userId)
	if err != nil {
		return nil, nil, err
	}

	if user.DefaultAccount != nil && user.DefaultAccount.ID == accountIdToSet {
		return []*v2.Grant{}, annotations.New(&v2.GrantAlreadyExists{}), nil
	}

	userResponse, _, err := o.client.SetUserAccount(ctx, userId, &accountIdToSet)
	if err != nil {
		return nil, nil, err
	}

	if userResponse.DefaultAccount == nil || userResponse.DefaultAccount.ID != accountIdToSet {
		return nil, nil, errors.New("baton-coupa: account not set on user")
	}

	newGrant := grant.NewGrant(
		resource,
		accountAssignedEntitlementName,
		&v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     strconv.Itoa(user.Id),
		},
	)

	return []*v2.Grant{newGrant}, nil, nil
}

func (o *accountBuilder) Revoke(ctx context.Context, g *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if g.Principal.Id.ResourceType != userResourceType.Id {
		return nil, fmt.Errorf("baton-coupa: principal resource type is not %s", userResourceType.Id)
	}

	accountIdToRemove, err := strconv.Atoi(g.Entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, err
	}

	userId, err := strconv.Atoi(g.Principal.Id.Resource)
	if err != nil {
		return nil, err
	}

	user, err := o.getUserAccounts(ctx, userId)
	if err != nil {
		return nil, err
	}

	if user.DefaultAccount == nil || user.DefaultAccount.ID != accountIdToRemove {
		l.Info("baton-coupa: account not found on user")
		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	_, _, err = o.client.SetUserAccount(ctx, userId, nil)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (o *accountBuilder) getUserAccounts(ctx context.Context, userId int) (*client.UserAccounts, error) {
	var target client.UserAccountsResponse
	response, _, err := o.client.Query(
		ctx,
		client.GetUserAccounts(userId),
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

func newAccountBuilder(ctx context.Context, client *client.Client) *accountBuilder {
	return &accountBuilder{
		client: client,
	}
}
