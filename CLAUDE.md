# semctl — CLI for Semaphore UI

## Project Overview
Go CLI tool for managing Semaphore UI via its REST API. Built with Cobra; config is plain yaml.v3.

## Key Patterns
- `pkg/semapi/` is a generated OpenAPI client — DO NOT HAND-EDIT
- User messages → stderr via `style.Success()`, `style.Error()`, etc.; `--quiet` (`style.SetQuiet`) suppresses Success/Info but never Warning/Error
- Data output → stdout via `output.Print()`, `output.PrintTable()`; `Print`/`PrintJSON`/`PrintYAML` return errors (never os.Exit) — always return or check them
- Config is owned end-to-end by `yaml.v3` (Viper dropped): reads go through the typed `fileConfig` struct, writes through a generic-map round-trip (`updateConfigFile`) that preserves unknown keys, holds an advisory flock + in-process mutex, writes atomically (temp+rename, 0600), and refuses to overwrite an unparseable file. Context names are CASE-INSENSITIVE — `config.NormalizeContextName` (lowercase) is applied at every boundary including `client.TokenCachePathForContext`; a config defining both `Prod` and `prod` fails Load
- Auth precedence: `SEMCTL_API_TOKEN`/config api_token → cached token (`~/.cache/semctl/tokens/`) → username/password cookie login (creates + caches token). On 401 with creds available, `reauthTransport` re-logins once and retries
- Token cache is SERVER-BOUND: the cache file stores `{token, server}` (server = `scheme://host:port`); `loadCachedToken` refuses a token whose server ≠ the currently resolved server (legacy caches without the field are invalid → one forced re-login). A `--server`/`SEMCTL_SERVER` override that differs from the context's configured server (`config.ServerRedirected()`) disables both the cached token AND the re-login/password fallback, so a redefined-context or CWD-config attack can't exfiltrate credentials. `client.ServerID(scheme,host,port)` builds the binding string
- Context names are validated (`config.ValidateContextName`, `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`) at every entry point (login/logout/context-delete/rename/ApplyContext/SaveContext, and `current_context` on Load) — they land in token-cache file paths and config map keys, so no `/`, `..`, or `.`
- `login` does NOT store the password unless `--save-password`; `logout` revokes the token server-side (`client.RevokeToken`)
- Env overrides (checked in config getters via `os.Getenv`, NOT Viper AutomaticEnv): `SEMCTL_SERVER`, `SEMCTL_SCHEME`, `SEMCTL_API_TOKEN`, `SEMCTL_AUTH_USERNAME`, `SEMCTL_AUTH_PASSWORD`
- Server resolution: `config.ResolveServer()` — `--server` flag > `SEMCTL_SERVER` > context config; parse host:port with `config.ParseHostPort` (net.SplitHostPort-based, errors on bad ports)
- HTTP: 30s default timeout (`client.SetTimeout` via `--timeout`), TLS options `--insecure`/`--ca-cert` + `server.insecure_skip_verify`/`server.ca_cert` config keys; `client.WarnIfPlaintext` on http to non-localhost
- Project-scoped commands: `getProjectID(cmd)` — `--project` flag (numeric ID or name, resolved case-insensitively) → `defaults.project_id` config → error
- Commands skipped in PersistentPreRunE: login, logout, completion, version, __complete (session flags like --timeout are processed before the skip)
- Interactive mode: `shouldAutoInteractive(cmd, inputsMissing)` pattern — suppressed whenever the output format ≠ table (any of --json/--yaml/--output/config default). Update commands with no `field=value` args launch a pre-filled form (`xUpdateForm` next to each command); form selects populate from the API via `nameIDOptions(cmd, xNameIDs, includeNone)` (form.go)
- Runner commands are GLOBAL by default; only an explicit `--project` flag selects project scope (`runnerScope(cmd)` — config defaults.project_id deliberately ignored)
- Update commands use `field=value` positional args pattern: fetch current resource, apply overrides, PUT back
- List/show/delete commands MUST go through the generics in cmd/resource_helpers.go — `runList(what, headers, fetch, row)`, `runShow(what, fetch, fields)`, `runDelete(cmd, what, id, del)`, `parseIDArg`. The fetch/del closure owns the typed go-swagger params; auth (`client.NewAuthenticatedClient`) happens BEFORE the helper so auth errors stay unwrapped. `what` is plural for lists, singular for show/delete
- Positional resource args accept ID or name: `resolveIDOrName(cmd, args[0], what, xNameIDs)` with per-resource fetchers in cmd/resolve.go (users match by USERNAME; runners follow `runnerScope`). The same fetchers back tab-completion (`completeResourceNames`). Unknown name → exit 4; ambiguous name (dup names, case-insensitive) → error listing IDs. Tasks (no names) and tokens (string IDs) keep `parseIDArg`/raw args
- Output format: global `--output table|json|yaml` with `--json`/`--yaml` as shorthands; conflicting formats error (`resolveOutputFormat` in root.go — it pointer-compares the flag against root's, because export shadows `--output` with its deprecated file-path flag; `-f/--file` is export's canonical file flag)
- Confirmations use `confirmAction(cmd, prompt)` (cmd/confirm.go): `--yes` skips; non-TTY stdin without `--yes` errors; declining returns `errCancelled` → non-zero exit. Never inline `[y/N]` prompts
- `task run --wait/--follow` polls status until success/error/stopped; non-success = non-zero exit
- Ctrl-C/SIGTERM cancel the root context (`signal.NotifyContext` in `Execute()`; a second Ctrl-C force-kills). Long-running work must watch `cmd.Context()`: the `task run --wait` poll loop, `confirmAction`, and the apply executor (`Execute(ctx, plan)`) all do — an interrupt stops before the next action, is NOT counted as an apply error, and warns that re-running apply resumes. EVERY API call is interrupt-cancellable even without an explicit ctx: `client.SetRootContext(cmd.Context())` (PersistentPreRunE) + `translatingTransport.SubmitContext` graft the session ctx onto calls whose own ctx can't be cancelled (the generated client passes `context.Background()`, never the transport default — don't rely on `Runtime.Context`); raw login/revoke HTTP uses `sessionCtx()`. Explicit per-call ctx via the generated `<Op>Context(ctx, params, authInfo)` call variants — `params.SetContext` is deprecated and trips lint (SA1019). `TranslateAPIError` renders `context.Canceled` as "interrupted"; it maps to exit 5
- Exit codes (cmd/exitcodes.go, mapped in `Execute()` via `exitCodeFor`): 1 generic, 2 apply drift (`--detailed-exitcode`), 3 auth (client.ErrNoCredentials/ErrAuthFailed sentinels + 401/403), 4 not-found (404), 5 cancelled (errCancelled), 6 task failed, 7 wait timeout. Attach specific codes with `withExitCode(err, code)`
- Under `--json`/`--yaml`, `task run` and all create commands print the created resource to stdout (env create clears `Password` first); `apply` prints `[]apply.PlanJSON` plan docs to stdout
- All API calls pass `nil` for authInfo, relying on `transport.DefaultAuthentication`
- API errors are translated at the TRANSPORT layer (`client.translatingTransport` wrapping every call from `NewAuthenticatedClient`): raw go-swagger dumps become human messages with status + server identity; `TranslateAPIError` keeps the original error in the chain and exposes `Code()`, so exit-code mapping and `IsNotFound` still work. SEMCTL_DEBUG appends the raw dump
- Version awareness: `client.TargetSemaphoreVersion` (major.minor, BUMP when regenerating the client), `client.WarnIfVersionMismatch(api)` (once per process, called by apply), `client.HTTPStatus(err)`/`client.IsNotFound(err)` for go-swagger status extraction. List-endpoint 404s get a version hint in `runList`; apply's schedule reconciliation tolerates a missing schedules API (pre-2.18) by leaving schedules unmanaged with a warning

## API Client Gotchas
- `getProjectID(cmd)` returns `int32` but API params expect `int64` — always cast with `int64(pid)`
- Environment APIs use `apiClient.VariableGroup` (not a separate Environment client)
- Semaphore 2.18+ omits `secrets` from the environment LIST response — fetch by ID to get them
- Token APIs use `apiClient.Authentication` (GetUserTokens, PostUserTokens, DeleteUserTokensAPITokenID)
- Event APIs use `apiClient.Operations` (GetEvents, GetEventsLast)
- Key resource has no GET-by-ID endpoint — fetch from list and filter by ID
- APIToken.ID is a `string` (not int64 like other resource IDs); 2.18 MASKS token IDs in the list response (8-char prefix) — DELETE /user/tokens/{id} accepts either the prefix or the full token (live-verified)
- Schedule list endpoint (GET /project/{id}/schedules) is implemented by the server but missing from the official spec — scripts/patch-spec.py adds it (and Schedule.tpl_name) before client generation; the list returns ScheduleWithTpl objects

## Releases
- Release flow: docs PR adding a row to COMPATIBILITY.md (NOT the README — the README Compatibility section is just a pointer) → merge → tag vX.Y.Z on that merge commit via gh API refs (triggers release.yml/goreleaser)

## Commit Style (Mandatory)
- Use [Conventional Commits](https://www.conventionalcommits.org): `type(scope): description`
- Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `build`
- Include key changes as bullet points in the commit body
- Example: `feat(cmd): add CRUD commands for all project resources`
- Every PR contains exactly ONE commit. If the branch has accumulated multiple
  commits, squash them before opening the PR (`git reset --soft main` and
  recommit with a message covering everything); if a PR is already open,
  squash and force-push with `--force-with-lease`

## Declarative Apply System (`internal/apply/`)
- Config types in `types.go`, reconciliation in `reconcile.go`, execution in `executor.go`, export in `export.go`
- "Variable Groups" (UI/CLI) = "Environments" (API) — use `apiClient.VariableGroup` client, `models.Environment*` types
- EnvironmentRequest fields: `JSON` (extra vars), `Env` (env vars) — both expect JSON strings, NOT `KEY=VALUE`
- EnvironmentSecretRequest: `Type` is `"var"` (extra vars) or `"env"` (env vars); `Operation` is `"create"`, `"update"`, or `"delete"`
- On secret update: match existing secrets by name to get IDs, use `operation: "update"` for existing, `"create"` for new
- Processing order — create: project→keys→variable_groups→repos→inventories→templates→schedules; delete: reverse
- All resource name lookups are case-insensitive (`strings.EqualFold`); name→ID maps are keyed LOWERCASED — always `strings.ToLower` on write AND read
- Env expansion is strict `${VAR}`-only (`expandEnv`): unset var = error in apply, warning in validate (`ParseFileOffline`); `$${VAR}` escapes; bare `$WORD` untouched. Runs AFTER parse (`expandConfigEnv` walks parsed string fields/slices/string-maps via reflection) so an env value can't inject YAML structure — consequently `${VAR}` only works in string fields, NOT numeric ones like `ssh_key_id`
- Updates MERGE over existing state (`mergeStr`/`mergeID`/`mergeBool` + `Reconciler.Existing*ByID` maps): empty config field = keep server value; template bools are `*bool` (nil = keep); SurveyVars/Vaults preserved from existing on update
- Validate rejects the literal `<set-me>` export placeholder (`ExportPlaceholder`)
- Schedules reconcile by name like other resources (duplicate names possible server-side: first match is managed, `state: absent` deletes ALL matches); `--skip-schedules` leaves them unmanaged
- Template updates must preserve fields apply doesn't manage (SurveyVars, Vaults, EnvironmentIds, TaskParams) — every TemplateRequest builder MUST call `apply.PreserveUnmanagedTemplateFields(req, existing)` (nil existing = fresh create); hand-rolling the copy is how the wipe bug happened twice
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
