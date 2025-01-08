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

type UserRoles struct {
	Id    int    `json:"id"`
	Roles []Role `json:"roles"`
}

type UserRolesResponse struct {
	Users []UserRoles `json:"users"`
}

type UserGroups struct {
	Id    int     `json:"id"`
	Group []Group `json:"userGroups"`
}

type UserGroupsResponse struct {
	Users []UserGroups `json:"users"`
}

type UserGroupsApiResponse struct {
	Id    int     `json:"id"`
	Group []Group `json:"user-groups"`
}

type UserRolesPutResponse struct {
	ResourceId
	Roles []Role `json:"roles"`
}

type UserLicenseResponse struct {
	Id              int  `json:"id"`
	RiskAssessUser  bool `json:"risk-assess-user"`
	AicUser         bool `json:"aic-user"`
	PurchasingUser  bool `json:"purchasing-user"`
	ExpenseUser     bool `json:"expense-user"`
	SourcingUser    bool `json:"sourcing-user"`
	InventoryUser   bool `json:"inventory-user"`
	ContractsUser   bool `json:"contracts-user"`
	AnalyticsUser   bool `json:"analytics-user"`
	SpendGuardUser  bool `json:"spend-guard-user"`
	CcwUser         bool `json:"ccw-user"`
	SupplyChainUser bool `json:"supply-chain-user"`
	TravelUser      bool `json:"travel-user"`
	TreasuryUser    bool `json:"treasury-user"`
}
