package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

// AccountGroupResourceTypeID is the stable ID for the account_group resource type.
// Used by callers to check whether account groups are included in SyncResourceTypeIDs.
const AccountGroupResourceTypeID = "account_group"

// The user resource type is for all user objects from the database.
var userResourceType = &v2.ResourceType{
	Id:          "user",
	DisplayName: "User",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
}

var groupResourceType = &v2.ResourceType{
	Id:          "group",
	DisplayName: "group",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
}

var roleResourceType = &v2.ResourceType{
	Id:          "role",
	DisplayName: "role",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
}

var licenseResourceType = &v2.ResourceType{
	Id:          "license",
	DisplayName: "license",
}

var accountGroupResourceType = &v2.ResourceType{
	Id:          AccountGroupResourceTypeID,
	DisplayName: "Account Group",
	Annotations: annotations.New(
		&v2.SkipEntitlementsAndGrants{},
		&v2.OptInRequired{},
		&v2.CapabilityPermissions{
			Permissions: []*v2.CapabilityPermission{
				{
					Permission: "core.accounting.read",
				},
			},
		},
	),
}
