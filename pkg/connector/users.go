package connector

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/conductorone/baton-coupa/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// Account creation schema field keys. These are the keys the connector reads
// out of AccountInfo.Profile, and must match the FieldMap advertised by
// Connector.Metadata.
const (
	accountFieldFirstname = "firstname"
	accountFieldLastname  = "lastname"
	accountFieldEmail     = "email"
	accountFieldLogin     = "login"
)

type userBuilder struct {
	client            *client.Client
	resourceType      *v2.ResourceType
	syncContentGroups bool
	syncAccountGroups bool
}

func (o *userBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for a Coupa user.
func userResource(user *client.User, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	status := v2.Status_RESOURCE_STATUS_DISABLED
	if user.Active {
		status = v2.Status_RESOURCE_STATUS_ENABLED
	}

	login := user.Email
	if user.Login != "" {
		login = user.Login
	}

	return resourceSdk.NewUserResource(
		user.Fullname,
		userResourceType,
		user.ID,
		[]resourceSdk.UserTraitOption{
			resourceSdk.WithEmail(user.Email, true),
			resourceSdk.WithUserLogin(login),
		},
		resourceSdk.WithResourceStatus(status, ""),
		resourceSdk.WithResourceProfile(map[string]any{
			"id":        user.ID,
			"login":     login,
			"email":     user.Email,
			"full_name": user.Fullname,
			"active":    user.Active,
		}),
		resourceSdk.WithParentResourceID(parentResourceID),
		resourceSdk.WithExternalID(&v2.ExternalId{
			Id: strconv.Itoa(user.ID),
		}),
	)
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (o *userBuilder) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	opts resourceSdk.SyncOpAttrs,
) (
	[]*v2.Resource,
	*resourceSdk.SyncOpResults,
	error,
) {
	logger := ctxzap.Extract(ctx)
	logger.Debug("Starting Users List", zap.String("token", opts.PageToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	var target client.UsersQueryResponse
	response, rateLimitData, err := o.client.Query(
		ctx,
		client.AllUsersQuery(opts.PageToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(rateLimitData)
	if err != nil {
		return nil, &resourceSdk.SyncOpResults{Annotations: outputAnnotations}, err
	}
	defer response.Body.Close()

	logger.Debug("Users List Response", zap.Any("response", target))

	lastId := ""
	for _, user := range target.Users {
		resource, err := userResource(user, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		outputResources = append(outputResources, resource)
		lastId = strconv.Itoa(user.ID)
	}

	return outputResources, &resourceSdk.SyncOpResults{NextPageToken: lastId, Annotations: outputAnnotations}, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(
	_ context.Context,
	_ *v2.Resource,
	_ resourceSdk.SyncOpAttrs,
) (
	[]*v2.Entitlement,
	*resourceSdk.SyncOpResults,
	error,
) {
	return nil, nil, nil
}

// Grants returns account group and/or content group membership grants for this user.
// These grants are generated from the user side to avoid the expensive pattern of
// fetching all users once per group.
func (o *userBuilder) Grants(
	ctx context.Context,
	resource *v2.Resource,
	_ resourceSdk.SyncOpAttrs,
) (
	[]*v2.Grant,
	*resourceSdk.SyncOpResults,
	error,
) {
	userId, err := strconv.Atoi(resource.Id.Resource)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-coupa: invalid user ID %q: %w", resource.Id.Resource, err)
	}

	outputGrants := make([]*v2.Grant, 0)
	var outputAnnotations annotations.Annotations

	var userGroups client.UserGroupsQueryResponse
	var query string
	switch {
	case o.syncAccountGroups && o.syncContentGroups:
		query = client.GetUserAccountAndContentGroupsByID(userId)
	case o.syncAccountGroups:
		query = client.GetUserAccountGroupsByID(userId)
	case o.syncContentGroups:
		query = client.GetUserContentGroupsByID(userId)
	default:
		return nil, &resourceSdk.SyncOpResults{Annotations: outputAnnotations}, nil
	}
	response, rateLimitData, err := o.client.Query(
		ctx,
		query,
		&userGroups,
	)
	outputAnnotations.WithRateLimiting(rateLimitData)
	if err != nil {
		return nil, &resourceSdk.SyncOpResults{Annotations: outputAnnotations}, err
	}
	defer response.Body.Close()

	// Emit account group grants.
	if o.syncAccountGroups {
		if len(userGroups.Users) > 0 {
			user := userGroups.Users[0]
			for _, accountGroup := range user.AccountGroups {
				outputGrants = append(outputGrants, grant.NewGrant(
					&v2.Resource{
						Id: &v2.ResourceId{
							ResourceType: accountGroupResourceType.Id,
							Resource:     strconv.Itoa(accountGroup.Id),
						},
					},
					accountGroupEntitlementName,
					resource.Id,
				))
			}
		}
	}

	// Emit content group grants when content group sync is enabled.
	if o.syncContentGroups {
		if len(userGroups.Users) > 0 {
			user := userGroups.Users[0]
			for _, contentGroup := range user.ContentGroups {
				outputGrants = append(outputGrants, grant.NewGrant(
					&v2.Resource{
						Id: &v2.ResourceId{
							ResourceType: contentGroupResourceType.Id,
							Resource:     strconv.Itoa(contentGroup.Id),
						},
					},
					contentGroupEntitlementName,
					resource.Id,
				))
			}
		}
	}

	return outputGrants, &resourceSdk.SyncOpResults{Annotations: outputAnnotations}, nil
}

// primaryEmail returns the account's primary email, falling back to the first
// email supplied when none is flagged primary.
func primaryEmail(accountInfo *v2.AccountInfo) string {
	emails := accountInfo.GetEmails()
	for _, e := range emails {
		if e.GetIsPrimary() {
			return e.GetAddress()
		}
	}
	if len(emails) > 0 {
		return emails[0].GetAddress()
	}
	return ""
}

// newCreateUserRequest maps an AccountInfo onto the Coupa user fields. Profile
// values come from the account creation schema advertised by
// Connector.Metadata; login and email fall back to the C1 user when the tenant
// has not mapped them.
func newCreateUserRequest(accountInfo *v2.AccountInfo) (*client.CreateUserRequest, error) {
	if accountInfo == nil {
		return nil, fmt.Errorf("baton-coupa: account info is required")
	}

	profileFields := accountInfo.GetProfile().GetFields()
	profileString := func(key string) string {
		return strings.TrimSpace(profileFields[key].GetStringValue())
	}

	login := profileString(accountFieldLogin)
	if login == "" {
		login = accountInfo.GetLogin()
	}
	if login == "" {
		return nil, fmt.Errorf("baton-coupa: login is required")
	}

	email := profileString(accountFieldEmail)
	if email == "" {
		email = primaryEmail(accountInfo)
	}
	if email == "" {
		return nil, fmt.Errorf("baton-coupa: email is required")
	}

	return &client.CreateUserRequest{
		Login:     login,
		Email:     email,
		Firstname: profileString(accountFieldFirstname),
		Lastname:  profileString(accountFieldLastname),
		Active:    true,
	}, nil
}

// CreateAccount creates a new user account in Coupa.
// Implements the AccountManagerLimited interface.
func (o *userBuilder) CreateAccount(
	ctx context.Context,
	accountInfo *v2.AccountInfo,
	credentialOptions *v2.LocalCredentialOptions,
) (
	connectorbuilder.CreateAccountResponse,
	[]*v2.PlaintextData,
	annotations.Annotations,
	error,
) {
	l := ctxzap.Extract(ctx)

	createReq, err := newCreateUserRequest(accountInfo)
	if err != nil {
		return nil, nil, nil, err
	}

	l.Debug("creating user account",
		zap.String("login", createReq.Login),
		zap.String("email", createReq.Email),
		zap.String("firstname", createReq.Firstname),
		zap.String("lastname", createReq.Lastname),
	)

	annos := annotations.New()
	userResponse, rateLimit, err := o.client.CreateUser(ctx, createReq)
	annos.WithRateLimiting(rateLimit)
	if err != nil {
		l.Error("failed to create user", zap.Error(err))
		return nil, nil, annos, fmt.Errorf("baton-coupa: failed to create user: %w", err)
	}

	l.Info("user created successfully",
		zap.Int("user_id", userResponse.ID),
		zap.String("login", userResponse.Login),
		zap.String("email", userResponse.Email),
	)

	// Create the resource for the newly created user
	newUser := &client.User{
		ID:       userResponse.ID,
		Login:    userResponse.Login,
		Email:    userResponse.Email,
		Fullname: userResponse.Fullname,
		Active:   userResponse.Active,
	}

	resource, err := userResource(newUser, nil)
	if err != nil {
		return nil, nil, annos, fmt.Errorf("baton-coupa: failed to create user resource: %w", err)
	}

	result := v2.CreateAccountResponse_SuccessResult_builder{
		Resource:              resource,
		IsCreateAccountResult: true,
	}.Build()

	return result, nil, annos, nil
}

// CreateAccountCapabilityDetails returns the capability details for account creation.
// Implements the AccountManagerLimited interface.
func (o *userBuilder) CreateAccountCapabilityDetails(
	ctx context.Context,
) (
	*v2.CredentialDetailsAccountProvisioning,
	annotations.Annotations,
	error,
) {
	// Coupa uses SSO/external authentication, so we indicate no password is needed
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
	}, nil, nil
}

func newUserBuilder(_ context.Context, client *client.Client, syncAccountGroups bool, syncContentGroups bool) *userBuilder {
	annos := annotations.Annotations(userResourceType.GetAnnotations())
	if syncAccountGroups || syncContentGroups {
		annos.Append(&v2.SkipEntitlements{})
	} else {
		annos.Append(&v2.SkipEntitlementsAndGrants{})
	}
	rt := &v2.ResourceType{
		Id:          userResourceType.Id,
		DisplayName: userResourceType.DisplayName,
		Traits:      userResourceType.Traits,
		Annotations: annos,
	}
	return &userBuilder{
		client:            client,
		resourceType:      rt,
		syncContentGroups: syncContentGroups,
		syncAccountGroups: syncAccountGroups,
	}
}
