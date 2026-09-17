package connector

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// The UI renders profile field mappings off this schema, so an unadvertised
// field is one CreateAccount can never receive.
func TestMetadataAccountCreationSchema(t *testing.T) {
	ctx := context.Background()

	metadata, err := (&Connector{}).Metadata(ctx)
	require.NoError(t, err)

	fieldMap := metadata.GetAccountCreationSchema().GetFieldMap()
	require.ElementsMatch(t,
		[]string{
			accountFieldFirstname, accountFieldLastname, accountFieldEmail, accountFieldLogin,
			accountFieldSSOIdentifier, accountFieldEmployeeNumber, accountFieldManagerLogin,
			accountFieldPurchasingUser, accountFieldInvoicingUser, accountFieldSourcingUser,
			accountFieldAccountSecurityType, accountFieldAuthenticationMethod, accountFieldDefaultLocale,
			accountFieldDefaultAccountType, accountFieldDefaultCurrency, accountFieldCustomFields,
		},
		slices.Collect(maps.Keys(fieldMap)),
	)

	orders := make(map[int32]string, len(fieldMap))
	for key, field := range fieldMap {
		require.NotEmpty(t, field.GetDisplayName(), "field %s is missing a display name", key)
		require.NotEmpty(t, field.GetDescription(), "field %s is missing a description", key)
		require.True(t, field.GetStringField() != nil || field.GetBoolField() != nil || field.GetIntField() != nil || field.GetMapField() != nil, "field %s has no supported type", key)

		require.NotContains(t, orders, field.GetOrder(), "field %s reuses an order", key)
		orders[field.GetOrder()] = key
	}

	// No field may be Required. C1 hard-fails provisioning when a required
	// schema field has no expression in the tenant's stored account-provision
	// config, and no existing config can name these keys — the connector never
	// advertised a schema for the UI to offer.
	for key, field := range fieldMap {
		require.False(t, field.GetRequired(), "field %s is marked required", key)
	}
}
