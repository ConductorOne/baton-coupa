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
		`"travelUser","treasuryUser","invoicingUser",` +
		`"categoryPlannerUser","categoryStrategyUser"]`

	// setAccountGroupPath sets account groups for a user by id.
	setAccountGroupPath = `/api/users/%d?fields=["id",{"account_groups":["id","name"]}]`

	// setContentGroupPath sets content groups (business_groups) for a user by id.
	// Note: the Coupa API uses "business_groups" as the URL slug but "content_groups" in the fields projection.
	setContentGroupPath = `/api/users/%d?fields=["id",{"content_groups":["id","name"]}]`

	// updateUserPath set user id in the path.
	updateUserPath = `/api/users/%d`

	// createUserPath is the path for creating a new user.
	createUserPath = `/api/users`
)
