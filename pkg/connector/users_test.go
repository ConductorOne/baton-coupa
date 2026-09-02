package connector

import (
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
