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
		[]string{accountFieldFirstname, accountFieldLastname, accountFieldEmail, accountFieldLogin},
		slices.Collect(maps.Keys(fieldMap)),
	)

	orders := make(map[int32]string, len(fieldMap))
	for key, field := range fieldMap {
		require.NotEmpty(t, field.GetDisplayName(), "field %s is missing a display name", key)
		require.NotEmpty(t, field.GetDescription(), "field %s is missing a description", key)
		require.NotNil(t, field.GetStringField(), "field %s is not a string field", key)

		require.NotContains(t, orders, field.GetOrder(), "field %s reuses an order", key)
		orders[field.GetOrder()] = key
	}

	require.True(t, fieldMap[accountFieldFirstname].GetRequired())
	require.True(t, fieldMap[accountFieldLastname].GetRequired())
}
