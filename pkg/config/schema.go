package config

import (
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/spf13/viper"
)

var (
	errCoupaDomainNotMatched = errors.New("coupa domain did not match the filters. must be of the form of https://[your-company].coupacloud.com or https://[your-company].coupahost.com")
	coupaDomainOptions       = []string{
		"coupacloud.com",
		"coupahost.com",
	}
	ClientIdField = field.StringField(
		"coupa-client-id",
		field.WithRequired(true),
		field.WithDescription("Your Coupa Client ID"),
	)
	ClientSecretField = field.StringField(
		"coupa-client-secret",
		field.WithRequired(true),
		field.WithDescription("Your Coupa Client Secret"),
	)
	CoupaDomain = field.StringField(
		"coupa-domain",
		field.WithRequired(true),
		field.WithDescription("Your Coupa Domain, ex: acme.coupacloud.com"),
	)
	// ConfigurationFields defines the external configuration required for the
	// connector to run. Note: these fields can be marked as optional or
	// required.
	ConfigurationFields = []field.SchemaField{
		ClientIdField,
		ClientSecretField,
		CoupaDomain,
	}

	ConfigurationSchema = field.Configuration{
		Fields: ConfigurationFields,
	}
)

// TODO(marcos) move this code to baton-sdk.
const (
	dns1123LabelFmt       string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123LabelInvalid   string = "value must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character"
	dns1123LabelTooLong   string = "value must be less than 64 characters"
	DNS1123LabelMaxLength int    = 63
)

var dns1123LabelRegexp = regexp.MustCompile("^" + dns1123LabelFmt + "$")

// IsDNS1123Label tests for a string that conforms to the definition of a label in DNS (RFC 1123).
func IsDNS1123Label(value string) error {
	if len(value) > DNS1123LabelMaxLength {
		return errors.New(dns1123LabelTooLong)
	}
	if !dns1123LabelRegexp.MatchString(value) {
		return errors.New(dns1123LabelInvalid)
	}
	return nil
}

func NormalizeCoupaURL(domain string) (string, error) {
	if strings.Contains(domain, "//") {
		u, err := url.Parse(domain)
		if err != nil {
			return "", err
		}
		domain = u.Host
	}

	domain = strings.ToLower(domain)

	parts := strings.Split(domain, ".")

	if len(parts) != 3 {
		return "", errCoupaDomainNotMatched
	}

	tenantDomain := parts[0]
	err := IsDNS1123Label(tenantDomain)
	if err != nil {
		return "", errCoupaDomainNotMatched
	}

	coupaDomain := strings.Join(parts[1:], ".")
	if !slices.Contains(coupaDomainOptions, coupaDomain) {
		return "", errCoupaDomainNotMatched
	}

	rv := "https://" + strings.Join([]string{tenantDomain, coupaDomain}, ".")
	return rv, nil
}

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(v *viper.Viper) error {
	_, err := NormalizeCoupaURL(v.GetString(CoupaDomain.FieldName))
	return err
}
