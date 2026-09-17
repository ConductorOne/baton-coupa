package connector

import (
	"encoding/json"
	"testing"

	"github.com/conductorone/baton-coupa/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func profile(t *testing.T, fields map[string]any) *structpb.Struct {
	t.Helper()

	s, err := structpb.NewStruct(fields)
	require.NoError(t, err)

	return s
}

func TestNewCreateUserRequest(t *testing.T) {
	tests := []struct {
		name        string
		accountInfo *v2.AccountInfo
		expected    *client.CreateUserRequest
		expectedErr string
	}{
		{
			name: "extended and custom fields",
			accountInfo: &v2.AccountInfo{
				Login:  "john.doe",
				Emails: []*v2.AccountInfo_Email{{Address: "john@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{
					"sso-identifier": "john-sso", "employee-number": "E123", "manager.login": "jane.manager",
					"purchasing-user": true, "invoicing-user": false, "sourcing-user": true,
					"account-security-type": 2, "authentication-method": "saml", "default-locale": "en-GB",
					"default-account-type": "Corporate", "default-currency": "GBP",
					"custom-fields": map[string]any{"custom-department-code": "ENG", "custom-enabled": true},
				}),
			},
			expected: &client.CreateUserRequest{
				Login: "john.doe", Email: "john@example.com", Active: true,
				SSOIdentifier: "john-sso", EmployeeNumber: "E123", Manager: &client.UserReference{Login: "jane.manager"},
				PurchasingUser: boolPointer(true), InvoicingUser: boolPointer(false), SourcingUser: boolPointer(true),
				AccountSecurityType: intPointer(2), AuthenticationMethod: "saml", DefaultLocale: "en-GB",
				DefaultAccountType: &client.NamedReference{Name: "Corporate"}, DefaultCurrency: &client.CurrencyReference{Code: "GBP"},
				CustomFields: map[string]any{"custom-department-code": "ENG", "custom-enabled": true},
			},
		},
		{
			name: "null and wrong-kind optional fields are omitted",
			accountInfo: &v2.AccountInfo{
				Login:  "john.doe",
				Emails: []*v2.AccountInfo_Email{{Address: "john@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{
					"purchasing-user":       nil,
					"invoicing-user":        "false",
					"sourcing-user":         1,
					"account-security-type": "2",
					"manager.login":         "   ",
					"default-account-type":  "\t",
					"default-currency":      "  ",
				}),
			},
			expected: &client.CreateUserRequest{
				Login: "john.doe", Email: "john@example.com", Active: true,
			},
		},
		{
			name:        "nil account info",
			accountInfo: nil,
			expectedErr: "account info is required",
		},
		{
			name: "profile fields drive every value",
			accountInfo: &v2.AccountInfo{
				Login:  "fallback-login",
				Emails: []*v2.AccountInfo_Email{{Address: "fallback@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{
					"firstname": "John",
					"lastname":  "Doe",
					"email":     "john.doe@example.com",
					"login":     "john.doe",
				}),
			},
			expected: &client.CreateUserRequest{
				Login:     "john.doe",
				Email:     "john.doe@example.com",
				Firstname: "John",
				Lastname:  "Doe",
				Active:    true,
			},
		},
		{
			name: "unmapped login and email fall back to the account",
			accountInfo: &v2.AccountInfo{
				Login: "fallback-login",
				Emails: []*v2.AccountInfo_Email{
					{Address: "secondary@example.com"},
					{Address: "primary@example.com", IsPrimary: true},
				},
				Profile: profile(t, map[string]any{
					"firstname": "John",
					"lastname":  "Doe",
				}),
			},
			expected: &client.CreateUserRequest{
				Login:     "fallback-login",
				Email:     "primary@example.com",
				Firstname: "John",
				Lastname:  "Doe",
				Active:    true,
			},
		},
		{
			name: "first email is used when none is primary",
			accountInfo: &v2.AccountInfo{
				Login:  "fallback-login",
				Emails: []*v2.AccountInfo_Email{{Address: "first@example.com"}},
			},
			expected: &client.CreateUserRequest{
				Login:  "fallback-login",
				Email:  "first@example.com",
				Active: true,
			},
		},
		{
			// Unmapped names are passed through empty, as before the schema
			// existed. CreateUserRequest omits them and Coupa decides.
			name: "unmapped names are not rejected",
			accountInfo: &v2.AccountInfo{
				Login:   "john.doe",
				Emails:  []*v2.AccountInfo_Email{{Address: "john.doe@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{"lastname": "Doe"}),
			},
			expected: &client.CreateUserRequest{
				Login:    "john.doe",
				Email:    "john.doe@example.com",
				Lastname: "Doe",
				Active:   true,
			},
		},
		{
			// Only the identifier lookups trim, so the fallback still engages.
			name: "whitespace-only login and email fall back to the account",
			accountInfo: &v2.AccountInfo{
				Login:  "fallback-login",
				Emails: []*v2.AccountInfo_Email{{Address: "primary@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{
					"login": "   ",
					"email": "\t",
				}),
			},
			expected: &client.CreateUserRequest{
				Login:  "fallback-login",
				Email:  "primary@example.com",
				Active: true,
			},
		},
		{
			name: "padded login and email are trimmed",
			accountInfo: &v2.AccountInfo{
				Login:  "fallback-login",
				Emails: []*v2.AccountInfo_Email{{Address: "fallback@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{
					"login": "  john.doe  ",
					"email": " john.doe@example.com ",
				}),
			},
			expected: &client.CreateUserRequest{
				Login:  "john.doe",
				Email:  "john.doe@example.com",
				Active: true,
			},
		},
		{
			// Names keep the pre-schema pass-through, untrimmed.
			name: "names are passed through untrimmed",
			accountInfo: &v2.AccountInfo{
				Login:   "john.doe",
				Emails:  []*v2.AccountInfo_Email{{Address: "john.doe@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{"firstname": " John ", "lastname": "Doe"}),
			},
			expected: &client.CreateUserRequest{
				Login:     "john.doe",
				Email:     "john.doe@example.com",
				Firstname: " John ",
				Lastname:  "Doe",
				Active:    true,
			},
		},
		{
			name: "missing login",
			accountInfo: &v2.AccountInfo{
				Emails: []*v2.AccountInfo_Email{{Address: "john.doe@example.com", IsPrimary: true}},
			},
			expectedErr: "login is required",
		},
		{
			name:        "missing email",
			accountInfo: &v2.AccountInfo{Login: "john.doe"},
			expectedErr: "email is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := newCreateUserRequest(test.accountInfo)
			if test.expectedErr != "" {
				require.ErrorContains(t, err, test.expectedErr)
				require.Nil(t, got)
				// Mapping problems are the tenant's configuration, not a connector
				// bug, so they must not surface as a retryable Internal error.
				require.Equal(t, codes.InvalidArgument, status.Code(err))
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected, got)
		})
	}
}

func boolPointer(value bool) *bool { return &value }
func intPointer(value int) *int    { return &value }

func TestCreateUserRequestMarshalJSON(t *testing.T) {
	req := client.CreateUserRequest{
		Login: "john", Email: "john@example.com", Active: true,
		Manager: &client.UserReference{Login: "manager"}, InvoicingUser: boolPointer(false),
		CustomFields: map[string]any{"custom-one": "value"},
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(data, &payload))
	require.Equal(t, "john", payload["login"])
	require.Equal(t, true, payload["active"])
	require.Equal(t, false, payload["invoicing-user"])
	require.Equal(t, map[string]any{"custom-one": "value"}, payload["custom-fields"])
	require.Equal(t, map[string]any{"login": "manager"}, payload["manager"])
	require.NotContains(t, payload, "default-currency")
}
