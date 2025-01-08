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
	client *client.Client
	ctx    context.Context
}

// ResourceSyncers returns a ResourceSyncer for each resource type that should be synced from the upstream service.
func (d *Connector) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		newUserBuilder(ctx, d.client),
		newGroupBuilder(ctx, d.client),
		newRoleBuilder(ctx, d.client),
		newLicenseBuilder(ctx, d.client),
	}
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
) (*Connector, error) {
	coupaClient, err := client.New(
		ctx,
		instanceUrl,
		clientId,
		clientSecret,
	)
	if err != nil {
		return nil, err
	}
	return &Connector{client: coupaClient, ctx: ctx}, nil
}
