package client

type ResourceId struct {
	Id int `json:"id"`
}

type Query struct {
	Query     string            `json:"query"`
	Variables map[string]string `json:"variables,omitempty"`
}

type UsersQueryResponse struct {
	Users []*User `json:"users"`
}

type GroupsQueryResponse struct {
	UserGroups []*Group `json:"userGroups"`
}

type RolesQueryResponse struct {
	Roles []*Role `json:"roles"`
}

type GroupMembersQueryResponse struct {
	UserGroups []struct {
		Id    int    `json:"id"`
		Name  string `json:"name"`
		Users []struct {
			Id int `json:"id"`
		} `json:"users"`
	} `json:"userGroups"`
}

type RoleGrantsQueryResponse struct {
	Users []struct {
		Id int `json:"id"`
	} `json:"users"`
}

type LicenseGrantsQueryResponse struct {
	Users []struct {
		Id int `json:"id"`
	} `json:"users"`
}

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Fullname string `json:"fullname"`
	Active   bool   `json:"active"`
}

type Group struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type Role struct {
	Name        string  `json:"name"`
	ID          int     `json:"id"`
	Description *string `json:"description,omitempty"`
}

type License struct {
	Name        string
	ID          string
	Description string
}

type UserRolesResponse struct {
	Users []struct {
		Id    int    `json:"id"`
		Roles []Role `json:"roles"`
	} `json:"users"`
}

type UserRolesPutResponse struct {
	ResourceId
	Roles []Role `json:"roles"`
}
