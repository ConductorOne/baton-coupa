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
				Profile: profile(t, map[string]any{
					"firstname": "John",
					"lastname":  "Doe",
				}),
			},
			expected: &client.CreateUserRequest{
				Login:     "fallback-login",
				Email:     "first@example.com",
				Firstname: "John",
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
		{
			name: "unmapped firstname",
			accountInfo: &v2.AccountInfo{
				Login:   "john.doe",
				Emails:  []*v2.AccountInfo_Email{{Address: "john.doe@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{"lastname": "Doe"}),
			},
			expectedErr: "firstname is required",
		},
		{
			name: "whitespace-only firstname",
			accountInfo: &v2.AccountInfo{
				Login:   "john.doe",
				Emails:  []*v2.AccountInfo_Email{{Address: "john.doe@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{"firstname": "   ", "lastname": "Doe"}),
			},
			expectedErr: "firstname is required",
		},
		{
			name: "unmapped lastname",
			accountInfo: &v2.AccountInfo{
				Login:   "john.doe",
				Emails:  []*v2.AccountInfo_Email{{Address: "john.doe@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{"firstname": "John"}),
			},
			expectedErr: "lastname is required",
		},
		{
			name: "whitespace-only lastname",
			accountInfo: &v2.AccountInfo{
				Login:   "john.doe",
				Emails:  []*v2.AccountInfo_Email{{Address: "john.doe@example.com", IsPrimary: true}},
				Profile: profile(t, map[string]any{"firstname": "John", "lastname": "\t"}),
			},
			expectedErr: "lastname is required",
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
