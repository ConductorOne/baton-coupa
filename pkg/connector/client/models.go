package client

type Query struct {
	Query     string            `json:"Query"`
	Variables map[string]string `json:"variables,omitempty"`
}

type UsersQueryResponse struct {
	Data struct {
		Users []*User `json:"users"`
	} `json:"data"`
}

type GroupsQueryResponse struct {
	Data struct {
		UserGroups []*Group `json:"userGroups"`
	} `json:"data"`
}

type RolesQueryResponse struct {
	Data struct {
		Roles []*Role `json:"roles"`
	} `json:"data"`
}

type GroupMembersQueryResponse struct {
	Data struct {
		UserGroups []struct {
			Id    int    `json:"id"`
			Name  string `json:"name"`
			Users []struct {
				Id int `json:"id"`
			} `json:"users"`
		} `json:"userGroups"`
	} `json:"data"`
}

type RoleGrantsQueryResponse struct {
	Data struct {
		Users []struct {
			Id int `json:"id"`
		} `json:"users"`
	} `json:"data"`
}

type LicenseGrantsQueryResponse struct {
	Data struct {
		Users []struct {
			Id int `json:"id"`
		} `json:"users"`
	} `json:"data"`
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
