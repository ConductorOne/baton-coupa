package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

// AccountGroupResourceTypeID is the stable ID for the account_group resource type.
// Used by callers to check whether account groups are included in SyncResourceTypeIDs.
const AccountGroupResourceTypeID = "account_group"

func capabilityPermissions(perms ...string) *v2.CapabilityPermissions {
	cp := &v2.CapabilityPermissions{}
	for _, p := range perms {
		cp.Permissions = append(cp.Permissions, &v2.CapabilityPermission{Permission: p})
	}
	return cp
}

// The user resource type is for all user objects from the database.
var userResourceType = &v2.ResourceType{
	Id:          "user",
	DisplayName: "User",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
	Annotations: annotations.New(
		capabilityPermissions(
			"core.user.read",
			"core.user.write",
			"email login",
			"openid",
			"profile",
		),
	),
}

var groupResourceType = &v2.ResourceType{
	Id:          "group",
	DisplayName: "group",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	Annotations: annotations.New(
		capabilityPermissions(
			"core.user_group.read",
			"core.user_group.write",
			"core.user.read",
			"core.user.write",
		),
	),
}

var roleResourceType = &v2.ResourceType{
	Id:          "role",
	DisplayName: "role",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
	Annotations: annotations.New(
		capabilityPermissions(
			"core.common.read",
			"core.user.read",
			"core.user.write",
		),
	),
}

var licenseResourceType = &v2.ResourceType{
	Id:          "license",
	DisplayName: "license",
	Annotations: annotations.New(
		capabilityPermissions(
			"core.business_entity.read",
			"core.user.read",
			"core.user.write",
		),
	),
}

var accountGroupResourceType = &v2.ResourceType{
	Id:          AccountGroupResourceTypeID,
	DisplayName: "Account Group",
	Annotations: annotations.New(
		&v2.SkipEntitlementsAndGrants{},
		&v2.OptInRequired{},
		capabilityPermissions(
			"core.accounting.read",
			"core.user.read",
			"core.user.write",
		),
	),
}
