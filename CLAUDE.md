# semctl — CLI for Semaphore UI

## Project Overview
Go CLI tool for managing Semaphore UI via its REST API. Built with Cobra + Viper.

## Key Patterns
- `pkg/semapi/` is a generated OpenAPI client — DO NOT HAND-EDIT
- User messages → stderr via `style.Success()`, `style.Error()`, etc.
- Data output → stdout via `output.Print()`, `output.PrintTable()`
- Config writing uses `yaml.v3` directly (not Viper) for partial file updates
- Auth precedence: `SEMCTL_API_TOKEN`/config api_token → cached token (`~/.cache/semctl/tokens/`) → username/password cookie login (creates + caches token). On 401 with creds available, `reauthTransport` re-logins once and retries
- `login` does NOT store the password unless `--save-password`; `logout` revokes the token server-side (`client.RevokeToken`)
- Env overrides (checked in config getters via `os.Getenv`, NOT Viper AutomaticEnv): `SEMCTL_SERVER`, `SEMCTL_SCHEME`, `SEMCTL_API_TOKEN`, `SEMCTL_AUTH_USERNAME`, `SEMCTL_AUTH_PASSWORD`
- Server resolution: `config.ResolveServer()` — `--server` flag > `SEMCTL_SERVER` > context config; parse host:port with `config.ParseHostPort` (net.SplitHostPort-based, errors on bad ports)
- HTTP: 30s default timeout (`client.SetTimeout` via `--timeout`), TLS options `--insecure`/`--ca-cert` + `server.insecure_skip_verify`/`server.ca_cert` config keys; `client.WarnIfPlaintext` on http to non-localhost
- Project-scoped commands: `getProjectID(cmd)` — `--project` flag (numeric ID or name, resolved case-insensitively) → `defaults.project_id` config → error
- Commands skipped in PersistentPreRunE: login, logout, completion, version, __complete (session flags like --timeout are processed before the skip)
- Interactive mode: `shouldAutoInteractive(cmd, inputsMissing)` pattern
- Update commands use `field=value` positional args pattern: fetch current resource, apply overrides, PUT back
- Confirmations use `confirmAction(cmd, prompt)` (cmd/confirm.go): `--yes` skips; non-TTY stdin without `--yes` errors; declining returns `errCancelled` → non-zero exit. Never inline `[y/N]` prompts
- `task run --wait/--follow` polls status until success/error/stopped; non-success = non-zero exit
- All API calls pass `nil` for authInfo, relying on `transport.DefaultAuthentication`

## API Client Gotchas
- `getProjectID(cmd)` returns `int32` but API params expect `int64` — always cast with `int64(pid)`
- Environment APIs use `apiClient.VariableGroup` (not a separate Environment client)
- Semaphore 2.18+ omits `secrets` from the environment LIST response — fetch by ID to get them
- Token APIs use `apiClient.Authentication` (GetUserTokens, PostUserTokens, DeleteUserTokensAPITokenID)
- Event APIs use `apiClient.Operations` (GetEvents, GetEventsLast)
- Key resource has no GET-by-ID endpoint — fetch from list and filter by ID
- APIToken.ID is a `string` (not int64 like other resource IDs)
- Schedule list endpoint (GET /project/{id}/schedules) is implemented by the server but missing from the official spec — scripts/patch-spec.py adds it (and Schedule.tpl_name) before client generation; the list returns ScheduleWithTpl objects

## Commit Style (Mandatory)
- Use [Conventional Commits](https://www.conventionalcommits.org): `type(scope): description`
- Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `build`
- Include key changes as bullet points in the commit body
- Example: `feat(cmd): add CRUD commands for all project resources`

## Declarative Apply System (`internal/apply/`)
- Config types in `types.go`, reconciliation in `reconcile.go`, execution in `executor.go`, export in `export.go`
- "Variable Groups" (UI/CLI) = "Environments" (API) — use `apiClient.VariableGroup` client, `models.Environment*` types
- EnvironmentRequest fields: `JSON` (extra vars), `Env` (env vars) — both expect JSON strings, NOT `KEY=VALUE`
- EnvironmentSecretRequest: `Type` is `"var"` (extra vars) or `"env"` (env vars); `Operation` is `"create"`, `"update"`, or `"delete"`
- On secret update: match existing secrets by name to get IDs, use `operation: "update"` for existing, `"create"` for new
- Processing order — create: project→keys→variable_groups→repos→inventories→templates→schedules; delete: reverse
- All resource name lookups are case-insensitive (`strings.EqualFold`); name→ID maps are keyed LOWERCASED — always `strings.ToLower` on write AND read
- Env expansion is strict `${VAR}`-only (`expandEnv`): unset var = error in apply, warning in validate (`ParseFileOffline`); `$${VAR}` escapes; bare `$WORD` untouched
- Updates MERGE over existing state (`mergeStr`/`mergeID`/`mergeBool` + `Reconciler.Existing*ByID` maps): empty config field = keep server value; template bools are `*bool` (nil = keep); SurveyVars/Vaults preserved from existing on update
- Validate rejects the literal `<set-me>` export placeholder (`ExportPlaceholder`)
- Schedules reconcile by name like other resources (duplicate names possible server-side: first match is managed, `state: absent` deletes ALL matches); `--skip-schedules` leaves them unmanaged
- Template updates must preserve fields apply doesn't manage: SurveyVars, Vaults, EnvironmentIds, TaskParams — copy from the existing template or the server wipes them
- Schema reference: `docs/apply-schema.md`

## Build
```bash
make                # Default: fmt + vet + build
make build          # Build binary
make test           # Run tests
make test-v         # Run tests with verbose output
make check          # Format + vet
make lint           # Run golangci-lint (requires separate install)
make generate       # Regenerate API client (requires Docker)
make install        # Install to $GOPATH/bin
```
