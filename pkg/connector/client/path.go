package client

const (
	apiPathAuth  = "/oauth2/token"
	apiPathQuery = "/api/graphql"

	// setRolesPath set user id in the path.
	setRolesPath = `/users/%d?fields=["id",{"roles":["id","description","name"]}]`
)
