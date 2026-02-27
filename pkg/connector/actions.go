package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"

	config "github.com/conductorone/baton-sdk/pb/c1/config/v1"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/actions"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	ActionEnableUser  = "enable_user"
	ActionDisableUser = "disable_user"
)

var enableUserAction = &v2.BatonActionSchema{
	Name:        ActionEnableUser,
	DisplayName: "Enable User",
	Description: "Enables a disabled user account in Coupa, allowing them to access the system",
	Arguments: []*config.Field{
		{
			Name:        "user_id",
			DisplayName: "User ID",
			Description: "The numeric ID of the user to enable",
			Field:       &config.Field_StringField{},
			IsRequired:  true,
		},
	},
	ReturnTypes: []*config.Field{
		{
			Name:        "success",
			DisplayName: "Success",
			Description: "Indicates whether the user was successfully enabled",
			Field:       &config.Field_BoolField{},
		},
	},
	ActionType: []v2.ActionType{
		v2.ActionType_ACTION_TYPE_ACCOUNT_ENABLE,
	},
}

var disableUserAction = &v2.BatonActionSchema{
	Name:        ActionDisableUser,
	DisplayName: "Disable User",
	Description: "Disables an active user account in Coupa, preventing them from accessing the system",
	Arguments: []*config.Field{
		{
			Name:        "user_id",
			DisplayName: "User ID",
			Description: "The numeric ID of the user to disable",
			Field:       &config.Field_StringField{},
			IsRequired:  true,
		},
	},
	ReturnTypes: []*config.Field{
		{
			Name:        "success",
			DisplayName: "Success",
			Description: "Indicates whether the user was successfully disabled",
			Field:       &config.Field_BoolField{},
		},
	},
	ActionType: []v2.ActionType{
		v2.ActionType_ACTION_TYPE_ACCOUNT_DISABLE,
	},
}

func (c *Connector) GlobalActions(ctx context.Context, registry actions.ActionRegistry) error {
	err := registry.Register(ctx, enableUserAction, c.enableUser)
	if err != nil {
		return err
	}

	err = registry.Register(ctx, disableUserAction, c.disableUser)
	if err != nil {
		return err
	}

	return nil
}

// extractAndValidateUserId extracts and validates the user_id argument from action arguments.
// Returns the user_id as both string and int, or an error if validation fails.
func extractAndValidateUserId(args *structpb.Struct) (string, int, error) {
	if args == nil {
		return "", 0, fmt.Errorf("arguments cannot be nil")
	}

	if args.Fields == nil {
		return "", 0, fmt.Errorf("arguments fields cannot be nil")
	}

	userId, ok := args.Fields["user_id"]
	if !ok {
		return "", 0, fmt.Errorf("missing required argument: user_id")
	}

	if userId == nil {
		return "", 0, fmt.Errorf("user_id value cannot be nil")
	}

	userIdStr := userId.GetStringValue()
	if userIdStr == "" {
		return "", 0, fmt.Errorf("user_id cannot be empty")
	}

	userIdInt, err := strconv.Atoi(userIdStr)
	if err != nil {
		return "", 0, fmt.Errorf("user_id must be a valid integer: %w", err)
	}

	if userIdInt <= 0 {
		return "", 0, fmt.Errorf("user_id must be a positive integer, got: %d", userIdInt)
	}

	return userIdStr, userIdInt, nil
}

// createActionResponse creates a standardized action response with a success field.
func createActionResponse(success bool) *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"success": structpb.NewBoolValue(success),
		},
	}
}

func (c *Connector) enableUser(ctx context.Context, args *structpb.Struct) (*structpb.Struct, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	userIdStr, userIdInt, err := extractAndValidateUserId(args)
	if err != nil {
		return nil, nil, err
	}

	l.Debug("enabling user", zap.String("userId", userIdStr), zap.Int("userIdInt", userIdInt))

	annos := annotations.New()
	updatedUser, rateLimit, err := c.client.UpdateUser(ctx, userIdInt, true)
	annos.WithRateLimiting(rateLimit)
	if err != nil {
		l.Error("failed to enable user", zap.String("userId", userIdStr), zap.Error(err))
		return nil, annos, fmt.Errorf("failed to enable user %s: %w", userIdStr, err)
	}

	success := updatedUser.Active
	if !success {
		l.Warn("user enable operation completed but user is still inactive",
			zap.String("userId", userIdStr),
			zap.Bool("active", updatedUser.Active))
	}

	return createActionResponse(success), annos, nil
}

func (c *Connector) disableUser(ctx context.Context, args *structpb.Struct) (*structpb.Struct, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	userIdStr, userIdInt, err := extractAndValidateUserId(args)
	if err != nil {
		return nil, nil, err
	}

	l.Debug("disabling user", zap.String("userId", userIdStr), zap.Int("userIdInt", userIdInt))

	annos := annotations.New()
	updatedUser, rateLimit, err := c.client.UpdateUser(ctx, userIdInt, false)
	annos.WithRateLimiting(rateLimit)
	if err != nil {
		l.Error("failed to disable user", zap.String("userId", userIdStr), zap.Error(err))
		return nil, annos, fmt.Errorf("failed to disable user %s: %w", userIdStr, err)
	}

	success := !updatedUser.Active
	if !success {
		l.Warn("user disable operation completed but user is still active",
			zap.String("userId", userIdStr),
			zap.Bool("active", updatedUser.Active))
	}

	return createActionResponse(success), annos, nil
}
