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
	Login    string `json:"login"`
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

type Account struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Active      bool    `json:"active"`
	AccountType *string `json:"accountType,omitempty"`
}

type AccountsQueryResponse struct {
	Accounts []*Account `json:"accounts"`
}

type AccountGrantsQueryResponse struct {
	Users []struct {
		Id int `json:"id"`
	} `json:"users"`
}

type UserAccounts struct {
	Id       int       `json:"id"`
	Accounts []Account `json:"account"`
}

type UserAccountsResponse struct {
	Users []UserAccounts `json:"users"`
}

type UserAccountsPutResponse struct {
	ResourceId
	Accounts []Account `json:"account"`
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

type UpdateUserRequest struct {
	Active bool `json:"active"`
}

type UpdateUserResponse struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Fullname string `json:"fullname"`
	Active   bool   `json:"active"`
}

// CreateUserRequest represents the request body for creating a user in Coupa.
// Reference: https://compass.coupa.com/en-us/products/product-documentation/integration-technical-documentation/the-coupa-core-api/resources/reference-data-resources/users-api-(users)
type CreateUserRequest struct {
	Login     string `json:"login"`
	Email     string `json:"email"`
	Firstname string `json:"firstname,omitempty"`
	Lastname  string `json:"lastname,omitempty"`
	Active    bool   `json:"active"`
}

// CreateUserResponse represents the response from creating a user in Coupa.
type CreateUserResponse struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Fullname  string `json:"fullname"`
	Active    bool   `json:"active"`
}
