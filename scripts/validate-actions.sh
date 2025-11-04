#!/usr/bin/env bash
set -exo pipefail

BATON_USER_ID=$1

if [ -z "$BATON_CONNECTOR" ]; then
  echo "BATON_CONNECTOR not set."
  exit 1
fi
if [ -z "$BATON" ]; then
  echo "BATON not set. using baton"
  BATON=baton
fi
if [ -z "$BATON_USER_ID" ]; then
  echo "BATON_USER_ID not set."
  exit 1
fi

# Error on unbound variables now that we've set BATON & BATON_CONNECTOR
set -u

echo "========================================="
echo "Testing User Enable/Disable Actions"
echo "User ID: $BATON_USER_ID"
echo "========================================="

# Initial SYNC
echo "Initial sync..."
$BATON_CONNECTOR

# Get initial user status
echo "Getting initial user status..."
INITIAL_STATUS=$($BATON resources --resource-type user --output-format=json | jq -r ".resources[] | select(.resource.id.resource == \"$BATON_USER_ID\") | .resource.annotations[] | select(.\"@type\" == \"type.googleapis.com/c1.connector.v2.UserTrait\") | .status.status")
echo "Initial status: $INITIAL_STATUS"

# Test 1: Disable the user
echo ""
echo "Test 1: Disabling user..."
$BATON_CONNECTOR --invoke-action disable_user --invoke-action-args="{\"userId\":\"$BATON_USER_ID\"}"

# Sync and verify user is disabled
echo "Syncing after disable..."
$BATON_CONNECTOR
DISABLED_STATUS=$($BATON resources --resource-type user --output-format=json | jq -r ".resources[] | select(.resource.id.resource == \"$BATON_USER_ID\") | .resource.annotations[] | select(.\"@type\" == \"type.googleapis.com/c1.connector.v2.UserTrait\") | .status.status")
echo "Status after disable: $DISABLED_STATUS"

if [ "$DISABLED_STATUS" != "STATUS_DISABLED" ]; then
  echo "ERROR: Expected user to be disabled, but status is: $DISABLED_STATUS"
  exit 1
fi
echo "✓ User successfully disabled"

# Test 2: Enable the user
echo ""
echo "Test 2: Enabling user..."
$BATON_CONNECTOR --invoke-action enable_user --invoke-action-args="{\"userId\":\"$BATON_USER_ID\"}"

# Sync and verify user is enabled
echo "Syncing after enable..."
$BATON_CONNECTOR
ENABLED_STATUS=$($BATON resources --resource-type user --output-format=json | jq -r ".resources[] | select(.resource.id.resource == \"$BATON_USER_ID\") | .resource.annotations[] | select(.\"@type\" == \"type.googleapis.com/c1.connector.v2.UserTrait\") | .status.status")
echo "Status after enable: $ENABLED_STATUS"

if [ "$ENABLED_STATUS" != "STATUS_ENABLED" ]; then
  echo "ERROR: Expected user to be enabled, but status is: $ENABLED_STATUS"
  exit 1
fi
echo "✓ User successfully enabled"

# Test 3: Disable again to test idempotency
echo ""
echo "Test 3: Disabling user again (idempotency test)..."
$BATON_CONNECTOR --invoke-action disable_user --invoke-action-args="{\"userId\":\"$BATON_USER_ID\"}"

# Sync and verify
echo "Syncing after second disable..."
$BATON_CONNECTOR
DISABLED_AGAIN_STATUS=$($BATON resources --resource-type user --output-format=json | jq -r ".resources[] | select(.resource.id.resource == \"$BATON_USER_ID\") | .resource.annotations[] | select(.\"@type\" == \"type.googleapis.com/c1.connector.v2.UserTrait\") | .status.status")
echo "Status after second disable: $DISABLED_AGAIN_STATUS"

if [ "$DISABLED_AGAIN_STATUS" != "STATUS_DISABLED" ]; then
  echo "ERROR: Expected user to be disabled, but status is: $DISABLED_AGAIN_STATUS"
  exit 1
fi
echo "✓ User successfully disabled (idempotent)"

# Test 4: Enable again to test idempotency
echo ""
echo "Test 4: Enabling user again (idempotency test)..."
$BATON_CONNECTOR --invoke-action enable_user --invoke-action-args="{\"userId\":\"$BATON_USER_ID\"}"

# Sync and verify
echo "Syncing after second enable..."
$BATON_CONNECTOR
ENABLED_AGAIN_STATUS=$($BATON resources --resource-type user --output-format=json | jq -r ".resources[] | select(.resource.id.resource == \"$BATON_USER_ID\") | .resource.annotations[] | select(.\"@type\" == \"type.googleapis.com/c1.connector.v2.UserTrait\") | .status.status")
echo "Status after second enable: $ENABLED_AGAIN_STATUS"

if [ "$ENABLED_AGAIN_STATUS" != "STATUS_ENABLED" ]; then
  echo "ERROR: Expected user to be enabled, but status is: $ENABLED_AGAIN_STATUS"
  exit 1
fi
echo "✓ User successfully enabled (idempotent)"

# Restore to initial state
echo ""
echo "Restoring user to initial state..."
if [ "$INITIAL_STATUS" = "STATUS_ENABLED" ]; then
  $BATON_CONNECTOR --invoke-action enable_user --invoke-action-args="{\"userId\":\"$BATON_USER_ID\"}"
  echo "✓ User restored to enabled state"
elif [ "$INITIAL_STATUS" = "STATUS_DISABLED" ]; then
  $BATON_CONNECTOR --invoke-action disable_user --invoke-action-args="{\"userId\":\"$BATON_USER_ID\"}"
  echo "✓ User restored to disabled state"
fi

echo ""
echo "========================================="
echo "✓ All action tests passed successfully!"
echo "========================================="

