package config

import (
	"testing"

	"github.com/conductorone/baton-sdk/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestConfigs(t *testing.T) {
	testCases := []test.TestCase{
		{
			Message: "empty",
			IsValid: false,
			Configs: map[string]string{},
		},
		{
			Message: "bad url",
			IsValid: false,
			Configs: map[string]string{
				"coupa-client-id":     "1",
				"coupa-client-secret": "1",
				"coupa-domain":        "1",
			},
		},

		{
			Message: "all",
			IsValid: true,
			Configs: map[string]string{
				"coupa-client-id":     "1",
				"coupa-client-secret": "1",
				"coupa-domain":        "https://example.coupacloud.com",
			},
		},
	}

	test.ExerciseTestCases(t, ConfigurationSchema, ValidateConfig, testCases)
}

func TestNormalizeCoupaURL(t *testing.T) {
	testCases := []struct {
		message  string
		url      string
		expected string
	}{
		{
			message:  "with scheme",
			url:      "https://www.coupacloud.com",
			expected: "https://www.coupacloud.com",
		}, {
			message:  "without scheme",
			url:      "www.coupacloud.com",
			expected: "https://www.coupacloud.com",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.message, func(t *testing.T) {
			actual, err := NormalizeCoupaURL(testCase.url)
			if err != nil {
				t.Fatal(err)
			}
			require.Equal(t, testCase.expected, actual)
		})
	}
}
