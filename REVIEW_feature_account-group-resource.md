# Connector Code Review: baton-coupa

**Branch:** `feature/account-group-resource` | **Base:** `main` | **Date:** 2026-04-15
**PR:** No PR found (uncommitted changes)
**Review type:** Full | **Build:** PASS | **Tests:** PASS (no test files)

## Summary

| Severity | Count |
|----------|-------|
| Critical | 1 |
| Warning  | 12 |
| Suggestion | 3 |

## Breaking Changes

- **B7 — New required OAuth scope** (`pkg/connector/client/auth.go`): Adding `core.accounting.read` to `ScopesReadOnly` means existing Coupa OAuth clients that don't have this scope will receive 403 errors on account group queries. Existing deployments must update their OAuth client configuration in Coupa to include this scope.

## Critical Issues

### `pkg/connector/account_groups.go` — Two-step clear+reset data loss in Revoke()

**ID:** REVOKE-001 | **Lines:** 243–258

Revoke() makes two sequential `SetAccountGroups` calls: first with an empty slice (clears ALL account groups), then with the desired `newIDs`. If the second call fails (transient network error, API timeout, etc.), the user is left with **zero account groups** — all memberships are permanently lost.

This is a destructive, non-atomic operation. The error on line 249–251 is logged and returned, but no compensation or restore attempt is made.

**Recommendation:** Either:
1. Validate whether a single `SetAccountGroups(ctx, userId, newIDs)` call is sufficient (skipping the clearing step), or  
2. On second-call failure, attempt a best-effort restore of the original `user.AccountGroups`, or  
3. At minimum, add a `WARN`-level log before the first call explaining that account groups will be temporarily cleared and that a failure mid-operation leaves the user with no account groups.

## Warnings

### `pkg/connector/account_groups.go` — Pagination termination: extra round-trip on final page

**ID:** R8-001, R8-002 | **Lines:** 67–77 (List), 136–155 (Grants)

Both `List()` and `Grants()` always return `lastId` (the last item's ID) as the next page token, even when the page is the final page. Termination only occurs when the *next* call returns zero items. This causes one unnecessary extra API call per sync per resource type.

While not an infinite loop, it does mean each sync for each account group's grants makes an extra round-trip per account group. At scale (many account groups × users), this is amplified.

**Recommendation:** Add an early-termination check using the page size:
```go
// In List(), after the loop:
if len(target.AccountGroups) == 0 {
    return outputResources, "", outputAnnotations, nil
}
```
Check against the API page size constant to detect partial final pages and return `""` immediately.

---

### `pkg/connector/client/auth.go` — New required scope is breaking for existing installs

**ID:** B7-001 | **Lines:** 13

Adding `core.accounting.read` to `ScopesReadOnly` requires existing Coupa OAuth client configurations to be updated. Installations that don't add this scope will fail account group queries. Consider graceful degradation: if account group queries return 403, log a warning and skip account group sync rather than failing the entire sync.

---

### `pkg/connector/client/path.go` — Field name mismatch: underscore vs hyphen

**ID:** C7-001 | **Lines:** 19

`setAccountGroupPath` requests fields as `{"account_groups":["id","name"]}` (underscore), but the REST response model `UserAccountGroupsApiResponse` uses JSON tag `account-groups` (hyphen). If Coupa's API uses the hyphen form for both request and response, the `fields` query parameter may silently return no account group data.

**Recommendation:** Verify against Coupa API documentation whether the `fields` query parameter uses `account_groups` (underscore) or `account-groups` (hyphen), and ensure the path constant matches.

---

### `pkg/connector/account_groups.go` — Bare errors without `baton-coupa:` prefix in Grant/Revoke

**ID:** R4-001, R4-002 | **Lines:** 161–168 (Grant), 215–221 (Revoke)

`strconv.Atoi` errors are returned without wrapping, prefix, or field context, making them hard to diagnose in logs.

**Recommendation:**
```go
// Grant():
accountGroupIdToAdd, err := strconv.Atoi(entitlement.Resource.Id.Resource)
if err != nil {
    return nil, nil, fmt.Errorf("baton-coupa: invalid account group ID %q: %w", entitlement.Resource.Id.Resource, err)
}
```

---

### `pkg/connector/account_groups.go` — `l` variable name should be `logger`

**ID:** R3-001 | **Lines:** 210

In `Revoke()`, the logger is named `l`. The rest of the file and codebase uses `logger` consistently.

**Recommendation:** Rename `l := ctxzap.Extract(ctx)` to `logger := ctxzap.Extract(ctx)`.

---

### `pkg/connector/account_groups.go` — `GrantAlreadyExists` returns empty slice instead of nil

**ID:** GRANT-001 | **Lines:** 176–179

Grant() returns `[]*v2.Grant{}` (empty non-nil slice) with the `GrantAlreadyExists` annotation. The SDK convention for idempotent already-exists responses is to return `nil` as the grants slice.

**Recommendation:** Change to:
```go
return nil, annotations.New(&v2.GrantAlreadyExists{}), nil
```

---

### `pkg/connector/client/account_groups.go` — Possible double body close in SetAccountGroups

**ID:** H1-SETACCOUNTGROUPS | **Lines:** 51

`doRestRequest` may already read and close `response.Body` internally before returning. The caller's `defer response.Body.Close()` would then be a double-close on an already-consumed body. While `http.Response.Body` handles double-close gracefully, it is a code smell. Verify whether `doRestRequest` closes the body internally; if so, remove the redundant defer.

---

### `pkg/connector/account_groups.go` — Missing nil guards on protobuf nested fields

**ID:** P6-NIL | **Lines:** 161, 166, 211, 215–217

`Grant()` accesses `entitlement.Resource.Id.Resource` and `principal.Id.Resource` without nil-checking intermediate fields. `Revoke()` accesses `g.Principal.Id.ResourceType`, `g.Entitlement.Resource.Id.Resource` without nil guards. A malformed grant proto causes a nil pointer panic.

**Recommendation:** Add nil guards before dereferencing:
```go
if g.Principal == nil || g.Principal.Id == nil {
    return nil, fmt.Errorf("baton-coupa: grant has nil principal")
}
if g.Entitlement == nil || g.Entitlement.Resource == nil || g.Entitlement.Resource.Id == nil {
    return nil, fmt.Errorf("baton-coupa: grant has nil entitlement resource")
}
```

---

### `pkg/connector/account_groups.go` — Rate limit data discarded in Grant/Revoke

**ID:** H4-RATELIMIT | **Lines:** 188, 243, 248

All `SetAccountGroups()` calls discard the returned `*v2.RateLimitDescription` with `_`. The SDK cannot back-pressure provisioning operations if rate limit data is not propagated.

**Recommendation:** Capture and return rate limit annotations from SetAccountGroups in Grant() and Revoke(), consistent with how List() handles rate limiting.

---

### `pkg/connector/client/auth.go` — ScopesReadWrite missing scopes documented in connector.mdx (pre-existing)

**ID:** SCOPE-MISMATCH | **Lines:** 22–26

The docs (`docs/connector.mdx`) list `core.business_entity.write` and `core.common.write` as required for READ/WRITE access, but `ScopesReadWrite` in code only appends `core.user_group.write` and `core.user.write`. This pre-existing mismatch should be resolved: either add the missing scopes to the code or verify they are not needed and remove them from the docs.

## Suggestions

### `pkg/connector/resource_types.go` — Verify SkipEntitlements doesn't suppress StaticEntitlements

**ID:** R6-001 | **Lines:** 35–41

`accountGroupResourceType` has `SkipEntitlements` annotation, which suppresses per-resource `Entitlements()` calls. Confirm the SDK does NOT suppress `StaticEntitlements()` when this annotation is set. The `Entitlements()` method already returns nil, so the annotation may be redundant.

---

### `pkg/connector/account_groups.go` — Unused `ctx` in newAccountGroupBuilder

**ID:** R2-001 | **Lines:** 283

`newAccountGroupBuilder` accepts `ctx context.Context` but doesn't use it. Consistent with other builders for signature uniformity, but the parameter is unused.

---

### `pkg/connector/account_groups.go` — WithRateLimiting called before error check

**ID:** H1-RATELIMIT-ORDER | **Lines:** 61, 130

`outputAnnotations.WithRateLimiting(ratelimitData)` is called before checking `if err != nil`. This is consistent with the pattern in other builders (List, Grants in groups.go, roles.go) and is intentional — it captures rate limit data even on error paths. No action needed, but note the pattern for future reviewers.

## Files Reviewed

| File | Category | Findings | Status |
|------|----------|----------|--------|
| `pkg/connector/client/auth.go` | Client | B7-001, SCOPE-MISMATCH | Reviewed |
| `pkg/connector/client/models.go` | Client | None | Reviewed |
| `pkg/connector/client/query.go` | Client | None | Reviewed |
| `pkg/connector/client/path.go` | Client | C7-001 | Reviewed |
| `pkg/connector/client/account_groups.go` | Client | H1-SETACCOUNTGROUPS | Reviewed |
| `pkg/connector/connector.go` | Connector Core | None | Reviewed |
| `pkg/connector/resource_types.go` | Resource Types | R6-001 | Reviewed |
| `pkg/connector/account_groups.go` | Resource Builder + Provisioning | REVOKE-001, R8-001, R8-002, R4-001, R4-002, R3-001, GRANT-001, P6-NIL, H4-RATELIMIT, R2-001 | Reviewed |
| `baton_capabilities.json` | Capabilities | None | Reviewed |
| `docs/connector.mdx` | Documentation | None (SCOPE-MISMATCH pre-existing) | Reviewed |
