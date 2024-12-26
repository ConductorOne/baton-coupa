package client

import (
	"context"
	"fmt"
	"net/http"
)

// SetLicense sets the roles for a user.
// https://compass.coupa.com/en-us/products/product-documentation/integration-technical-documentation/the-coupa-core-api/resources/reference-data-resources/users-api-(users)
func (c *Client) SetLicense(
	ctx context.Context,
	userId int,
	licenseId string,
	active bool,
) (*UserLicenseResponse, error) {
	err := c.Initialize(ctx)
	if err != nil {
		return nil, err
	}

	request := map[string]bool{
		licenseId: active,
	}

	var userResponse UserLicenseResponse

	resonse, _, err := c.doRestRequest(
		ctx,
		http.MethodPut,
		c.baseUrl.JoinPath(fmt.Sprintf(setLicensePath, userId)),
		request,
		&userResponse,
	)

	if err != nil {
		return nil, err
	}

	defer resonse.Body.Close()

	return &userResponse, nil
}
