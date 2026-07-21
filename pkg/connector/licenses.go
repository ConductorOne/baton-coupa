package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-coupa/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const licenseEntitlementName = "assigned"

type licenseBuilder struct {
	client *client.Client
}

func (o *licenseBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return licenseResourceType
}

func licenseResource(license *client.License, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	description := fmt.Sprintf("%s license in Coupa", license.Name)
	assignedEntitlementID := entitlement.NewEntitlementID(
		&v2.Resource{Id: &v2.ResourceId{ResourceType: licenseResourceType.Id, Resource: license.ID}},
		licenseEntitlementName,
	)

	return resourceSdk.NewResource(
		license.Name,
		licenseResourceType,
		license.ID,
		resourceSdk.WithParentResourceID(parentResourceID),
		resourceSdk.WithDescription(description),
		resourceSdk.WithLicenseProfileTrait(
			resourceSdk.WithLicenseName(license.Name),
			resourceSdk.WithLicenseEntitlementIDs(assignedEntitlementID),
		),
	)
}

func (o *licenseBuilder) List(
	_ context.Context,
	parentResourceID *v2.ResourceId,
	_ resourceSdk.SyncOpAttrs,
) (
	[]*v2.Resource,
	*resourceSdk.SyncOpResults,
	error,
) {
	coupaLicenses := []*client.License{
		{
			Name:        "AI classification",
			ID:          "aic-user",
			Description: "An AI Spend Classification license",
		},
		{
			Name:        "Analytics",
			ID:          "analytics-user",
			Description: "An Analytics license",
		},
		{
			Name:        "Category Planner",
			ID:          "category_planner_user",
			Description: "A Category Planner license",
		},
		{
			Name:        "Category Strategy",
			ID:          "category_strategy_user",
			Description: "A Category Strategy license",
		},
		{
			Name:        "CLM Advanced",
			ID:          "clm-advanced-user",
			Description: "A Contract Lifecycle Management Advanced license",
		},
		{
			Name:        "Contingent Workforce",
			ID:          "ccw-user",
			Description: "A Contingent Workforce license",
		},
		{
			// This does not revoke
			Name:        "Contracts",
			ID:          "contracts-user",
			Description: "A Contracts license",
		},
		{
			Name:        "Expense",
			ID:          "expense-user",
			Description: "An Expense license",
		},
		{
			Name:        "Intake",
			ID:          "intake-user",
			Description: "An Intake license",
		},
		{
			Name:        "Inventory",
			ID:          "inventory-user",
			Description: "An Inventory license",
		},
		{
			Name:        "Invoicing",
			ID:          "invoicing_user",
			Description: "An Invoicing license",
		},
		{
			Name:        "Navi AI Agent",
			ID:          "coupa-navi-ai-agent-user",
			Description: "A Navi AI Agent license",
		},
		{
			// This does not revoke
			Name:        "Purchasing",
			ID:          "purchasing-user",
			Description: "A Purchasing license",
		},
		{
			Name:        "Risk Assess",
			ID:          "risk-assess-user",
			Description: "A Risk Assess license",
		},
		{
			Name:        "Sourcing",
			ID:          "sourcing-user",
			Description: "A Sourcing license",
		},
		{
			Name:        "Spend Guard",
			ID:          "spend-guard-user",
			Description: "A Spend Guard license",
		},
		{
			Name:        "Supply Chain",
			ID:          "supply-chain-user",
			Description: "A Supply Chain license",
		},
		{
			Name:        "Travel",
			ID:          "travel-user",
			Description: "A Travel license",
		},
		{
			Name:        "Treasury",
			ID:          "treasury_user",
			Description: "A Treasury license",
		},
	}
	outputResources := make([]*v2.Resource, 0)
	for _, license := range coupaLicenses {
		resource, err := licenseResource(license, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		outputResources = append(outputResources, resource)
	}

	return outputResources, nil, nil
}

func (o *licenseBuilder) Entitlements(
	_ context.Context,
	resource *v2.Resource,
	_ resourceSdk.SyncOpAttrs,
) (
	[]*v2.Entitlement,
	*resourceSdk.SyncOpResults,
	error,
) {
	return []*v2.Entitlement{
		entitlement.NewAssignmentEntitlement(
			resource,
			licenseEntitlementName,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDisplayName(
				fmt.Sprintf("%s License", resource.DisplayName),
			),
			entitlement.WithDescription(
				fmt.Sprintf("%s license in Coupa", resource.DisplayName),
			),
		),
	}, nil, nil
}

func (o *licenseBuilder) Grants(
	ctx context.Context,
	resource *v2.Resource,
	opts resourceSdk.SyncOpAttrs,
) (
	[]*v2.Grant,
	*resourceSdk.SyncOpResults,
	error,
) {
	logger := ctxzap.Extract(ctx)

	licenseId := resource.Id.Resource

	logger.Debug(
		"Starting Licenses Grants",
		zap.String("license_id", licenseId),
		zap.String("token", opts.PageToken.Token),
	)

	outputGrants := make([]*v2.Grant, 0)
	var outputAnnotations annotations.Annotations

	var target client.LicenseGrantsQueryResponse
	response, ratelimitData, err := o.client.Query(
		ctx,
		client.LicenseGrantQuery(licenseId, opts.PageToken.Token),
		&target,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, &resourceSdk.SyncOpResults{Annotations: outputAnnotations}, err
	}
	defer response.Body.Close()

	lastId := ""
	for _, user := range target.Users {
		userId := strconv.Itoa(user.Id)
		outputGrants = append(
			outputGrants,
			grant.NewGrant(
				resource,
				licenseEntitlementName,
				&v2.ResourceId{
					ResourceType: userResourceType.Id,
					Resource:     userId,
				},
			),
		)
		lastId = userId
	}

	return outputGrants, &resourceSdk.SyncOpResults{NextPageToken: lastId, Annotations: outputAnnotations}, nil
}

func (o *licenseBuilder) Grant(ctx context.Context, resource *v2.Resource, entitlement *v2.Entitlement) ([]*v2.Grant, annotations.Annotations, error) {
	licenseIdToAdd := entitlement.Resource.Id.Resource

	userId, err := strconv.Atoi(resource.Id.Resource)
	if err != nil {
		return nil, nil, err
	}

	_, err = o.client.SetLicense(ctx, userId, licenseIdToAdd, true)
	if err != nil {
		return nil, nil, err
	}

	newGrant := grant.NewGrant(
		resource,
		licenseEntitlementName,
		&v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     resource.Id.Resource,
		},
	)

	return []*v2.Grant{newGrant}, nil, nil
}

func (o *licenseBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	if grant.Principal.Id.ResourceType != userResourceType.Id {
		return nil, fmt.Errorf("baton-coupa: principal resource type is not %s", userResourceType.Id)
	}

	licenseIdToRemove := grant.Entitlement.Resource.Id.Resource

	userId, err := strconv.Atoi(grant.Principal.Id.Resource)
	if err != nil {
		return nil, err
	}

	_, err = o.client.SetLicense(ctx, userId, licenseIdToRemove, false)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func newLicenseBuilder(ctx context.Context, client *client.Client) *licenseBuilder {
	return &licenseBuilder{
		client: client,
	}
}
