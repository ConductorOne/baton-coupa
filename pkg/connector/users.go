package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-coupa/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
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
	response, rateLimitData, err := o.client.Query(
		ctx,
		client.AllUsersQuery(pToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(rateLimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	defer response.Body.Close()

	logger.Debug("Users List Response", zap.Any("response", target))

	lastId := ""
	for _, user := range target.Users {
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

	if accountInfo == nil {
		return nil, nil, nil, fmt.Errorf("baton-coupa: account info is required")
	}

	login := accountInfo.GetLogin()
	if login == "" {
		return nil, nil, nil, fmt.Errorf("baton-coupa: login is required")
	}

	// Get the primary email from the account info
	var email string
	for _, e := range accountInfo.GetEmails() {
		if e.GetIsPrimary() {
			email = e.GetAddress()
			break
		}
	}
	if email == "" && len(accountInfo.GetEmails()) > 0 {
		email = accountInfo.GetEmails()[0].GetAddress()
	}
	if email == "" {
		return nil, nil, nil, fmt.Errorf("baton-coupa: email is required")
	}

	// Extract optional first/last name from the profile if available
	var firstname, lastname string
	if profile := accountInfo.GetProfile(); profile != nil {
		if fn := profile.GetFields()["firstname"]; fn != nil {
			firstname = fn.GetStringValue()
		}
		if ln := profile.GetFields()["lastname"]; ln != nil {
			lastname = ln.GetStringValue()
		}
	}

	l.Debug("creating user account",
		zap.String("login", login),
		zap.String("email", email),
		zap.String("firstname", firstname),
		zap.String("lastname", lastname),
	)

	annos := annotations.New()
	userResponse, rateLimit, err := o.client.CreateUser(ctx, login, email, firstname, lastname)
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

func newUserBuilder(ctx context.Context, client *client.Client) *userBuilder {
	return &userBuilder{
		client: client,
	}
}
