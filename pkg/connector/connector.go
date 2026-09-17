package connector

import (
	"context"
	"io"

	"github.com/conductorone/baton-coupa/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"golang.org/x/oauth2"
)

type Connector struct {
	client            *client.Client
	ctx               context.Context
	SyncAccountGroups bool
	SyncContentGroups bool
}

// ResourceSyncers returns a ResourceSyncerV2 for each resource type that should be synced from the upstream service.
func (d *Connector) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncerV2 {
	syncers := []connectorbuilder.ResourceSyncerV2{
		newUserBuilder(ctx, d.client, d.SyncAccountGroups, d.SyncContentGroups),
		newGroupBuilder(ctx, d.client),
		newRoleBuilder(ctx, d.client),
		newLicenseBuilder(ctx, d.client),
		newAccountGroupBuilder(ctx, d.client),
		newContentGroupBuilder(ctx, d.client),
	}
	return syncers
}

// Asset takes an input AssetRef and attempts to fetch it using the connector's authenticated http client
// It streams a response, always starting with a metadata object, following by chunked payloads for the asset.
func (d *Connector) Asset(ctx context.Context, asset *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

// Metadata returns metadata about the connector.
func (d *Connector) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Coupa Connector",
		Description: "Connector syncing Coupa users, groups, roles, and licenses",
		AccountCreationSchema: &v2.ConnectorAccountCreationSchema{
			FieldMap: accountCreationFields(),
		},
	}, nil
}

func accountCreationFields() map[string]*v2.ConnectorAccountCreationSchema_Field {
	stringField := func(name, description, placeholder string, order int32) *v2.ConnectorAccountCreationSchema_Field {
		return &v2.ConnectorAccountCreationSchema_Field{DisplayName: name, Description: description, Placeholder: placeholder, Order: order, Field: &v2.ConnectorAccountCreationSchema_Field_StringField{StringField: &v2.ConnectorAccountCreationSchema_StringField{}}}
	}
	boolField := func(name, description string, order int32) *v2.ConnectorAccountCreationSchema_Field {
		return &v2.ConnectorAccountCreationSchema_Field{DisplayName: name, Description: description, Order: order, Field: &v2.ConnectorAccountCreationSchema_Field_BoolField{BoolField: &v2.ConnectorAccountCreationSchema_BoolField{}}}
	}
	fields := map[string]*v2.ConnectorAccountCreationSchema_Field{
		accountFieldFirstname: {
			DisplayName: "First name",
			Required:    false,
			Description: "First name of the person who will own the Coupa user.",
			Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
				StringField: &v2.ConnectorAccountCreationSchema_StringField{},
			},
			Placeholder: "John",
			Order:       1,
		},
		accountFieldLastname: {
			DisplayName: "Last name",
			Required:    false,
			Description: "Last name of the person who will own the Coupa user.",
			Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
				StringField: &v2.ConnectorAccountCreationSchema_StringField{},
			},
			Placeholder: "Doe",
			Order:       2,
		},
		accountFieldEmail: {
			DisplayName: "Email",
			Required:    false,
			Description: "Email address of the Coupa user. Defaults to the C1 user's primary email.",
			Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
				StringField: &v2.ConnectorAccountCreationSchema_StringField{},
			},
			Placeholder: "john.doe@example.com",
			Order:       3,
		},
		accountFieldLogin: {
			DisplayName: "Login",
			Required:    false,
			Description: "Login for the Coupa user. Defaults to the C1 user's username.",
			Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
				StringField: &v2.ConnectorAccountCreationSchema_StringField{},
			},
			Placeholder: "john.doe",
			Order:       4,
		},
		accountFieldSSOIdentifier:        stringField("SSO identifier", "Single sign-on identifier for the Coupa user.", "john.doe", 5),
		accountFieldEmployeeNumber:       stringField("Employee number", "Employee number for the Coupa user.", "12345", 6),
		accountFieldManagerLogin:         stringField("Manager login", "Login of the user's manager in Coupa.", "manager.login", 7),
		accountFieldPurchasingUser:       boolField("Purchasing user", "Assign a Coupa Purchasing license during account creation.", 8),
		accountFieldInvoicingUser:        boolField("Invoicing user", "Assign a Coupa Invoicing license during account creation.", 9),
		accountFieldSourcingUser:         boolField("Sourcing user", "Assign a Coupa Sourcing license during account creation.", 10),
		accountFieldAccountSecurityType:  {DisplayName: "Account security type", Description: "Coupa account security type identifier.", Order: 11, Field: &v2.ConnectorAccountCreationSchema_Field_IntField{IntField: &v2.ConnectorAccountCreationSchema_IntField{}}},
		accountFieldAuthenticationMethod: stringField("Authentication method", "Coupa authentication method: coupa_credentials, ldap, or saml (case-sensitive).", "saml", 12),
		accountFieldDefaultLocale:        stringField("Default locale", "Default locale for the Coupa user.", "en", 13),
		accountFieldDefaultAccountType:   stringField("Default account type", "Name of the user's default Coupa account type.", "Chart of Accounts", 14),
		accountFieldDefaultCurrency:      stringField("Default currency", "ISO code of the user's default currency.", "USD", 15),
		accountFieldCustomFields:         {DisplayName: "Custom fields", Description: "Coupa instance-specific user fields sent in the custom-fields namespace.", Order: 16, Field: &v2.ConnectorAccountCreationSchema_Field_MapField{MapField: &v2.ConnectorAccountCreationSchema_MapField{}}},
	}
	return fields
}

// Validate is called to ensure that the connector is properly configured. It should exercise any API credentials
// to be sure that they are valid.
func (d *Connector) Validate(ctx context.Context) (annotations.Annotations, error) {
	err := d.client.Initialize(ctx)
	return nil, err
}

// SetTokenSource this method makes Coupa implement the OAuth2Connector
// interface. When an OAuth2Connector is created, this method gets called.
func (d *Connector) SetTokenSource(tokenSource oauth2.TokenSource) {
	logger := ctxzap.Extract(d.ctx)
	logger.Debug("baton-coupa: SetTokenSource start")
	d.client.ReadOnlyTokenSource = tokenSource
}

// New returns a new instance of the connector.
func New(
	ctx context.Context,
	instanceUrl string,
	clientId string,
	clientSecret string,
	syncAccountGroups bool,
	syncContentGroups bool,
	baseURL string,
) (*Connector, error) {
	coupaClient, err := client.New(
		ctx,
		instanceUrl,
		clientId,
		clientSecret,
		syncAccountGroups,
		baseURL,
	)
	if err != nil {
		return nil, err
	}

	return &Connector{client: coupaClient, ctx: ctx, SyncAccountGroups: syncAccountGroups, SyncContentGroups: syncContentGroups}, nil
}
