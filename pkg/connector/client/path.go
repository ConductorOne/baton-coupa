package client

const (
	apiPathAuth  = "/oauth2/token"
	apiPathQuery = "/api/graphql"

	// setRolesPath set user id in the path.
	setRolesPath = `/api/users/%d?fields=["id",{"roles":["id","description","name"]}]`
	// setGroupPath set user id in the path.
	setGroupPath = `/api/users/%d?fields=["id",{"user_groups":["id","name","description"]}]`

	// setLicensePath set user id in the path.
	setLicensePath = `/api/users/%d?fields=["id","analyticsUser","aicUser","ccwUser",` +
		`"contractsUser","expenseUser","inventoryUser","purchasingUser",` +
		`"riskAssessUser","sourcingUser","spendGuardUser","supplyChainUser",` +
		`"travelUser","treasuryUser"]`

	// setAccountPath set user id in the path.
	setAccountPath = `/api/users/%d?fields=["id",{"account":["id","name","code"]}]`

	// updateUserPath set user id in the path.
	updateUserPath = `/api/users/%d`

	// createUserPath is the path for creating a new user.
	createUserPath = `/api/users`
)
