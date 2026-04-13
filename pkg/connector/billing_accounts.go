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

const billingAccountMemberEntitlementName = "member"

type billingAccountBuilder struct {
	client *client.Client
}

func (o *billingAccountBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return billingAccountResourceType
}

func billingAccountResource(account *client.Account, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	displayName := account.Name
	if account.Code != "" {
		displayName = fmt.Sprintf("%s (%s)", account.Name, account.Code)
	}

	return resourceSdk.NewResource(
		displayName,
		billingAccountResourceType,
		account.ID,
		resourceSdk.WithParentResourceID(parentResourceID),
	)
}

func (o *billingAccountBuilder) List(
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
	logger.Debug("Starting Billing Accounts List", zap.String("token", pToken.Token))

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
		return nil, "", outputAnnotations, err
	}
	defer response.Body.Close()

	lastId := ""
	for _, account := range target.Accounts {
		resource, err := billingAccountResource(account, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, resource)
		lastId = strconv.Itoa(account.ID)
	}

	return outputResources, lastId, outputAnnotations, nil
}

func (o *billingAccountBuilder) Entitlements(
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
			billingAccountMemberEntitlementName,
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

func (o *billingAccountBuilder) Grants(
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
		"Starting Billing Account Grants",
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
		return nil, "", outputAnnotations, err
	}
	defer response.Body.Close()

	lastId := ""
	for _, user := range target.Users {
		outputGrants = append(
			outputGrants,
			grant.NewGrant(
				resource,
				billingAccountMemberEntitlementName,
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

func (o *billingAccountBuilder) Grant(ctx context.Context, resource *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	accountIdToAdd, err := strconv.Atoi(entitlement.Resource.Id.Resource)
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

	for _, account := range user.Accounts {
		if account.ID == accountIdToAdd {
			return []*v2.Grant{}, annotations.New(&v2.GrantAlreadyExists{}), nil
		}
	}

	accountIDs := make([]int, 0)
	for _, account := range user.Accounts {
		accountIDs = append(accountIDs, account.ID)
	}
	accountIDs = append(accountIDs, accountIdToAdd)

	userResponse, _, err := o.client.SetUserAccounts(ctx, userId, accountIDs)
	if err != nil {
		return nil, nil, err
	}

	if len(userResponse.Accounts) != len(accountIDs) {
		return nil, nil, errors.New("baton-coupa: billing accounts not set")
	}

	newGrant := grant.NewGrant(
		resource,
		billingAccountMemberEntitlementName,
		&v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     strconv.Itoa(user.Id),
		},
	)

	return []*v2.Grant{newGrant}, nil, nil
}

func (o *billingAccountBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if grant.Principal.Id.ResourceType != userResourceType.Id {
		return nil, fmt.Errorf("baton-coupa: principal resource type is not %s", userResourceType.Id)
	}

	accountIdToRemove, err := strconv.Atoi(grant.Entitlement.Resource.Id.Resource)
	if err != nil {
		return nil, err
	}

	userId, err := strconv.Atoi(grant.Principal.Id.Resource)
	if err != nil {
		return nil, err
	}

	user, err := o.getUserAccounts(ctx, userId)
	if err != nil {
		return nil, err
	}

	index := slices.IndexFunc(user.Accounts, func(c client.Account) bool {
		return c.ID == accountIdToRemove
	})
	if index < 0 {
		l.Info("baton-coupa: billing account not found in user")
		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	if index == 0 {
		user.Accounts = user.Accounts[1:]
	} else {
		user.Accounts = append(user.Accounts[:index], user.Accounts[index+1:]...)
	}

	newAccountIDs := make([]int, 0)
	for _, account := range user.Accounts {
		newAccountIDs = append(newAccountIDs, account.ID)
	}

	// Clear all accounts first, then re-set desired accounts.
	// This follows the same two-step pattern used for roles and groups,
	// as Coupa may require clearing before re-assignment.
	_, _, err = o.client.SetUserAccounts(ctx, userId, make([]int, 0))
	if err != nil {
		return nil, err
	}

	userResponse, _, err := o.client.SetUserAccounts(ctx, userId, newAccountIDs)
	if err != nil {
		l.Error(
			"baton-coupa: error setting billing accounts",
			zap.Error(err),
			zap.Ints("accounts", newAccountIDs),
		)
		return nil, err
	}

	if len(userResponse.Accounts) != len(newAccountIDs) {
		return nil, errors.New("baton-coupa: billing account was not removed")
	}

	return nil, nil
}

func (o *billingAccountBuilder) getUserAccounts(ctx context.Context, userId int) (*client.UserAccounts, error) {
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

func newBillingAccountBuilder(ctx context.Context, client *client.Client) *billingAccountBuilder {
	return &billingAccountBuilder{
		client: client,
	}
}
