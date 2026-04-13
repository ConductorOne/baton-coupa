package client

import "fmt"

const (
	getAllUsersQuery = `query getUsers{
	users(query: "%s&type[blank]=true") {
		id
		email
		fullname
		active
	}
}`

	getGroupsQuery = `query getGroups {
	userGroups(query: "%s") {
		id
		name
		description
	}
}`

	getGroupMemberListQuery = `query getGroupMembers {
	userGroups(query: "id=%s") {
		id
		users {
			id
		}
	}
}`
	getRoleQuery = `query getRoles {
	roles(query: "%s") {
		id
		name
		description
	}
}`

	getRoleGrantListQuery = `query getRoleGrants {
	users(query: "roles[id]=%s%s") {
		id
	}
}`

	getLicenseGrantListQuery = `query getRoleGrants {
	users(query: "%s=true%s") {
		id
	}
}`

	getUserRoles = `query getUsers {
	users(query: "id=%d") { 
		id roles { id name description }
	} 
}
`

	getUserGroups = `query getUsers {
	users(query: "id=%d") {
		id userGroups { id name description }
	}
}
`

	getAccountsQuery = `query getAccounts {
	accounts(query: "%s") {
		id
		name
		code
		active
	}
}`

	getAccountGrantsQuery = `query getAccountGrants {
	users(query: "default_account[id]=%s%s") {
		id
	}
}`

	getUserAccounts = `query getUsers {
	users(query: "id=%d") {
		id defaultAccount { id name code }
	}
}
`
)

func pagination(pg string) string {
	if pg == "" {
		return ""
	}
	return fmt.Sprintf("id[gt]=%s", pg)
}

func appendedPagination(pg string) string {
	apg := pagination(pg)
	if apg == "" {
		return ""
	}
	return fmt.Sprintf("&%s", apg)
}

func AllUsersQuery(pg string) string {
	return fmt.Sprintf(getAllUsersQuery, pagination(pg))
}

func GroupsQuery(pg string) string {
	return fmt.Sprintf(getGroupsQuery, pagination(pg))
}

func GroupMembersQuery(groupID string) string {
	return fmt.Sprintf(getGroupMemberListQuery, groupID)
}

func RolesQuery(pg string) string {
	return fmt.Sprintf(getRoleQuery, pagination(pg))
}

func RoleGrantQuery(roleID string, pg string) string {
	return fmt.Sprintf(getRoleGrantListQuery, roleID, appendedPagination(pg))
}

func LicenseGrantQuery(licenseName string, pg string) string {
	return fmt.Sprintf(getLicenseGrantListQuery, licenseName, appendedPagination(pg))
}

func GetUserRoles(userId int) string {
	return fmt.Sprintf(getUserRoles, userId)
}

func GetUserGroups(userId int) string {
	return fmt.Sprintf(getUserGroups, userId)
}

func AccountsQuery(pg string) string {
	return fmt.Sprintf(getAccountsQuery, pagination(pg))
}

func AccountGrantQuery(accountID string, pg string) string {
	return fmt.Sprintf(getAccountGrantsQuery, accountID, appendedPagination(pg))
}

func GetUserAccounts(userId int) string {
	return fmt.Sprintf(getUserAccounts, userId)
}
