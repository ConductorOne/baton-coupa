package main

import (
	"context"
	"slices"

	cfg "github.com/conductorone/baton-coupa/pkg/config"
	"github.com/conductorone/baton-coupa/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
)

var version = "dev"

func main() {
	ctx := context.Background()

	config.RunConnector(
		ctx,
		"baton-coupa",
		version,
		cfg.Config,
		getConnector,
		connectorrunner.WithDefaultCapabilitiesConnectorBuilderV2(&connector.Connector{SyncAccountGroups: true}),
	)
}

func getConnector(ctx context.Context, cc *cfg.Coupa, connectorOpts *cli.ConnectorOpts) (connectorbuilder.ConnectorBuilderV2, []connectorbuilder.Opt, error) {
	syncAccountGroups := slices.Contains(connectorOpts.SyncResourceTypeIDs, connector.AccountGroupResourceTypeID) || len(connectorOpts.SyncResourceTypeIDs) == 0
	cb, err := connector.New(
		ctx,
		cc.CoupaDomain,
		cc.CoupaClientId,
		cc.CoupaClientSecret,
		syncAccountGroups,
	)
	if err != nil {
		return nil, nil, err
	}
	return cb, nil, nil
}
