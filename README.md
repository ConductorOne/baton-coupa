![Baton Logo](./docs/images/baton-logo.png)

#
`baton-coupa` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-coupa.svg)](https://pkg.go.dev/github.com/conductorone/baton-coupa) ![ci](https://github.com/conductorone/baton-coupa/actions/workflows/ci.yaml/badge.svg) ![verify](https://github.com/conductorone/baton-coupa/actions/workflows/verify.yaml/badge.svg)

`baton-coupa` is a connector for Coupa built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the Coupa API to sync data about users, user groups, roles, per-module licenses, account groups and content groups, and supports provisioning of group, role, license, account-group and content-group membership as well as Coupa user accounts.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

- [Coupa docs](https://compass.coupa.com/en-us/products/core-platform/integration-playbooks-and-resources/integration-knowledge-articles/postman-collection-for-coupa-apis)
- [Coupa GraphQL](https://compass.coupa.com/en-us/products/product-documentation/integration-technical-documentation/the-coupa-core-api/get-started-with-the-api/introducing-graphql)

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-coupa
baton-coupa
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_COUPA_DOMAIN=acme.coupacloud.com -e BATON_COUPA_CLIENT_ID=clientId -e BATON_COUPA_CLIENT_SECRET=clientSecret ghcr.io/conductorone/baton-coupa:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

The `baton` CLI cannot be installed with `go install` — its module carries `replace`
directives, which `go install` rejects. Build it from a checkout, or take a prebuilt binary
from the [baton-sdk releases](https://github.com/ConductorOne/baton-sdk/releases).

```
git clone --depth 1 https://github.com/ConductorOne/baton-sdk.git
cd baton-sdk && go build -o "$(go env GOPATH)/bin/baton" ./cmd/baton && cd -

go install github.com/conductorone/baton-coupa/cmd/baton-coupa@main

baton-coupa

baton resources
```

# Data Model

`baton-coupa` will pull down information about the following resources:

- Users
- Groups
- Roles
- Licenses
- Account Groups
- Content Groups

Account Groups sync is opt-in: enable the resource type when configuring the connector in C1,
and add the `core.accounting.read` scope to your Coupa OAuth client.

Grant, revoke and account creation require the `--provisioning` flag (`BATON_PROVISIONING`).

# Actions

`baton-coupa` supports the following actions on user accounts:

## Enable User
Enables a disabled user account in Coupa.

**Action Name:** `enable_user`

**Arguments:**
- `user_id` (required, string): The Coupa user's numeric ID — the `id` field on `/api/users`, passed as a string (for example `"42"`). A login or an email address is rejected.

**Returns:**
- `success` (boolean): `true` if the user was successfully enabled

**Example:**
```bash
baton-coupa --invoke-action enable_user --invoke-action-args='{"user_id":"42"}'
```

## Disable User
Disables an active user account in Coupa.

**Action Name:** `disable_user`

**Arguments:**
- `user_id` (required, string): The Coupa user's numeric ID — the `id` field on `/api/users`, passed as a string (for example `"42"`). A login or an email address is rejected.

**Returns:**
- `success` (boolean): `true` if the user was successfully disabled

**Example:**
```bash
baton-coupa --invoke-action disable_user --invoke-action-args='{"user_id":"42"}'
```

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually
building spreadsheets. We welcome contributions, and ideas, no matter how
small&mdash;our goal is to make identity and permissions sprawl less painful for
everyone. If you have questions, problems, or ideas: Please open a GitHub Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-coupa` Command Line Usage

```
baton-coupa

Usage:
  baton-coupa [flags]
  baton-coupa [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --client-id string             The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string         The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
      --coupa-client-id string       required: Your Coupa Client ID ($BATON_COUPA_CLIENT_ID)
      --coupa-client-secret string   required: Your Coupa Client Secret ($BATON_COUPA_CLIENT_SECRET)
      --coupa-domain string          required: Your Coupa Domain, ex: acme.coupacloud.com ($BATON_COUPA_DOMAIN)
  -f, --file string                  The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                         help for baton-coupa
      --log-format string            The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string             The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning                 This must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --skip-full-sync               This must be set to skip a full sync ($BATON_SKIP_FULL_SYNC)
      --ticketing                    This must be set to enable ticketing support ($BATON_TICKETING)
  -v, --version                      version for baton-coupa

Use "baton-coupa [command] --help" for more information about a command.
```
