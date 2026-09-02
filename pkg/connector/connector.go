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
			FieldMap: map[string]*v2.ConnectorAccountCreationSchema_Field{
				accountFieldFirstname: {
					DisplayName: "First name",
					Required:    true,
					Description: "First name of the person who will own the Coupa user.",
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Placeholder: "John",
					Order:       1,
				},
				accountFieldLastname: {
					DisplayName: "Last name",
					Required:    true,
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
			},
		},
	}, nil
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
