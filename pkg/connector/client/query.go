package client

import "fmt"

const (
	getAllUsersQuery = `query getUsers{
	users(Query: "%s&type[blank]=true") {
		id
		email
		fullname
		active
	}
}`

	getGroupsQuery = `query getGroups {
	userGroups(Query: "%s") {
		id
		name
		description
	}
}`

	getGroupMemberListQuery = `query getGroupMembers {
	userGroups(Query: "id=%s") {
		id
		users {
			id
		}
	}
}`
	getRoleQuery = `query getRoles {
	roles(Query: "%s") {
		id
		name
		description
	}
}`

	getRoleGrantListQuery = `query getRoleGrants {
	users(Query: "roles[id]=%s%s") {
		id
	}
}`

	getLicenseGrantListQuery = `query getRoleGrants {
	users(Query: "%s=true%s") {
		id
	}
}`
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
