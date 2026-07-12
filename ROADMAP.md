# semctl Roadmap

Pending features, fixes, and hardening work. Updated 2026-07-12 after a full code review
(security / UX / CI-CD / architecture). Completed items (interactive create forms, user
commands, apply, export) have been removed.

---

## 1. Confirmed Bugs (fix first — all reproduced)

| # | Bug | Where | Notes |
|---|-----|-------|-------|
| 1.1 | `semctl login -s <host>` fails with `unknown shorthand flag: 's'` | `cmd/login.go:190` | Local `--server` flag shadows the root persistent `-s, --server` and pflag drops the shorthand. Same shadowing (divergent help text) for `--context` on login/logout. Remove the local flags and read the persistent ones. |
| 1.2 | `--no-color` is a no-op | `cmd/root.go:27` → `internal/output/output.go:41-43` | `output.DisableColor()` has an empty body; the real `style.DisableColor()` (`style.go:63`) is never called. Also wire emoji off (`SetEmojiEnabled` is dead code) — CI logs currently always get ✅/❌. |
| 1.3 | Subcommand typos exit 0 | all parent commands (`cmd/project.go`, `task.go`, …) | `semctl project lst` prints help and exits 0 — silent success for scripts. Add unknown-subcommand errors on parent commands. |
| 1.4 | Empty list = exit 1 in table mode, exit 0 + `null` in `--json` | every `*_list.go` (e.g. `cmd/repo_list.go:56-58`), `internal/output/json.go` | Empty is a normal state: print info to stderr, exit 0, and emit `[]` (not `null`) for nil slices in JSON/YAML. |
| 1.5 | `key update` silently wipes stored secrets | `cmd/key_update.go:101-115` | Sub-structs rebuilt only from passed fields with `OverrideSecret: true` always set — `key update 2 login=x` erases the private key. Refuse partial secret updates or require explicit confirmation. |
| 1.6 | Auto-interactive checks stdout TTY only | `cmd/form.go:78`, `internal/style/style.go:68` | `semctl project create < /dev/null` on a terminal launches a form that dies on EOF with a raw bubbletea error. Gate on `IsTTY() && IsStdinTTY()`. |
| 1.7 | Form abort surfaces huh's raw `user aborted` | form call sites | Map `huh.ErrUserAborted` → `errCancelled` so form abort and confirm decline speak the same language. |

---

## 2. Security Hardening

Ordered by severity; items 2.1–2.3 are design-level.

- **2.1 Bind cached tokens to server identity (HIGH).** Token cache
  (`~/.cache/semctl/tokens/<context>.json`, `internal/client/client.go:340`) stores only the
  token, keyed by context name. A CWD `semctl.yaml`/`.semctl.yaml` (auto-loaded,
  `internal/config/config.go:74-81`) can redefine a context to point at an attacker's host —
  the cached bearer token is sent there, and with `SEMCTL_AUTH_USERNAME/PASSWORD` set,
  `reauthTransport` sends the password after the 401. Fix: store the server host in the cache
  file and refuse mismatches; consider opt-in trust for CWD configs (direnv-style). Also
  covers the `--server`-override variant (skip cached-token/password fallback when the
  override differs from the context's server).
- **2.2 Expand `${VAR}` after parsing, not before (MEDIUM).** `internal/apply/types.go:196`
  expands over raw file text, so an env value containing newlines injects YAML structure
  (e.g. a branch name containing `"\nproject_state: absent` + `apply --yes` = project
  deletion). Walk parsed string scalars instead, or YAML-escape substituted values.
- **2.3 Disable go-openapi's `DEBUG` wire dumping (MEDIUM).** `go-openapi/runtime` enables
  full request/response dumps — including `Authorization` headers and secret bodies — when
  the generic `DEBUG`/`SWAGGER_DEBUG` env var is non-empty (verified in v0.32.4
  `logger/logger.go:13-22`). Set `transport.Debug = false` in `newClientWithToken`, or gate
  behind `SEMCTL_DEBUG` with header redaction.
- **2.4 Validate context names (MEDIUM).** Names flow unsanitized into cache paths
  (`client.go:341`) — `../../foo` escapes the tokens dir (arbitrary 0600 file write on save,
  arbitrary `.json` delete on logout/context delete). Enforce `^[A-Za-z0-9._-]+$` at all
  entry points (`--context`, config `current_context`, `login --context`). Also closes Viper
  key injection via dots in names (`config.go:149,270,282,294`).
- **2.5 `export -o` writes 0644** (`cmd/export.go:85`) — exports contain plaintext variables
  even with secrets masked. Write 0600 like config/tokens.
- **2.6 `env show --json` prints the password the table masks** (`cmd/env_show.go:46-54`).
  Pick one policy; redact in all formats.
- **2.7 Non-argv secret input for key/env commands.** `key create --private-key/--passphrase`
  and `env create --password` only accept secrets via argv (ps/shell-history leak); the
  key create Example even teaches `--private-key "$(cat ~/.ssh/id_rsa)"`. Add
  `--private-key-file` and `--password-stdin` variants.
- **2.8 Warn when TLS verification is disabled via config.** `server.insecure_skip_verify`
  from a config file (including a CWD config) silently disables verification
  (`client.go:62-67`); only the `--insecure` flag is an explicit choice. Emit a warning once
  per invocation whatever the source.
- **2.9 Token-cache lifecycle gaps.** `context rename` orphans a still-valid cached token;
  `context delete` removes the cache without server-side revocation (contrast `logout`,
  which revokes). Rename should move the cache file; delete should attempt revocation.
- **2.10 Tighten existing config file perms on write.** `os.WriteFile(path, out, 0600)`
  leaves a pre-existing 0644 file at 0644 (`config.go:478`) — `login --save-password` into it
  stores the password world-readable. `os.Chmod(path, 0600)` after write.
- **2.11 `getCacheDir` ignores `os.UserHomeDir` error** (`client.go:330`) — on failure the
  token lands in a relative `.cache/` under the CWD. Return an error instead.

---

## 3. CI/CD Friendliness

- **3.1 Distinct exit codes.** Everything exits 1 (`cmd/root.go:85-89`) — auth failure,
  not-found, cancelled, task-failed, and wait-timeout are indistinguishable. Map sentinel
  errors (`errCancelled` exists already) to documented codes in `Execute()`.
- **3.2 Machine-readable mutations.** `task run` prints the new task ID only in a styled
  stderr message (`cmd/task_run.go:86`); same for every create command. Honor `--json` by
  printing the created resource/task to stdout so pipelines can capture IDs.
- **3.3 `apply --dry-run` drift gate.** Exits 0 whether the plan is empty or full
  (`cmd/apply.go:87-95`), plan is human text on stderr. Add `--detailed-exitcode`
  (0 in-sync / 2 changes / 1 error) and a `--json` plan output for GitOps loops.
- **3.4 `--quiet` flag** suppressing `style.Success/Info` (keep Error/Warning) — today the
  only silencer is `2>/dev/null`, which also hides real errors.
- **3.5 `--wait-timeout` defaults to 0 = wait forever** (`cmd/task_run.go:187`) — pick a
  sane default or warn when waiting unbounded in non-TTY mode. Consider opt-in
  retry/backoff for transient 5xx (only 401 is retried today).
- **3.6 Ephemeral-token leak on username/password auth.** Each cookie login mints a
  server-side API token that is never revoked; on a read-only filesystem the ignored cache
  write (`client.go:146-150`) means every command mints another. Document
  `SEMCTL_API_TOKEN` as the CI path; consider revoking ephemeral tokens at exit.
- **3.7 Own CI hardening** (`.github/workflows/`): `go test -race -cover`, govulncheck,
  dependabot, pin golangci-lint version (currently `latest`), cross-compile darwin/windows
  in CI (`goreleaser build --snapshot`), gate release.yml on lint too. goreleaser: consider
  homebrew tap / docker image / SBOM+signing if distributing publicly.

---

## 4. UX Polish

- **4.1 Dynamic shell completion.** Zero `ValidArgsFunction` in the repo. Highest value:
  context names for `context use` (purely local), project names for `-p`, field names for
  update commands, template/task IDs.
- **4.2 API error translation.** Raw go-swagger errors reach users
  (`[GET /project/{project_id}/…] …NotFound (status 404): {…}`). One `errors.As` helper
  mapping 404 → "repository 5 not found in project 3", 401 → auth hint, 400 → server
  message. Include the server/context identity in API errors for multi-context users.
- **4.3 Consistency fixes:**
  - `project update` takes no positional ID while every other update does — accept one.
  - Name resolution works for `-p` but nowhere else (`project show myproj` fails) — accept
    names in positional resource args.
  - Unified `--output table|json|yaml` (keep `--json`/`--yaml` as aliases; error on
    conflict — today `--json --yaml` silently picks JSON). Note `export -o` means *file*.
  - Accept kebab-case in `field=value` args (create flags are kebab, update fields snake).
  - Standardize update-arg validation on the friendly `no fields to update` message
    (user/runner use bare cobra `requires at least 2 arg(s)`).
  - Usage brackets: `context use [name]` → `<name>` (args are required).
- **4.4 Missing confirmations:** `task stop` (kills a running deployment), `runner
  deactivate`, `runner clear-cache`, and the secret-overwriting `key update` path.
- **4.5 Help text:** `template update` Long omits the `arguments` field and its
  unknown-field error doesn't list valid fields (every sibling does); document
  `user show me`; add a root-level Example showing the login → project list → task run flow.
- **4.6 Form improvements:** populate repository/inventory/environment selects from the API
  in `template create` (the two-stage `key create` form is the pattern to copy); note in
  forms that more options exist as flags; add a form to `task run`; optional pre-filled
  forms for update commands when no `field=value` args are given.
- **4.7 Table rendering:** wrap/truncate long cells (`WrapNone` today produces enormous
  lines for env JSON); render nil ints as `-`/empty, not `0` (nil `MaxParallelTasks`
  currently indistinguishable from explicit 0).

---

## 5. Architecture / Tech Debt

- **5.1 Generic command helpers before Phase 12.** All business logic lives in 74 `RunE`
  closures; list/show/delete are byte-for-byte the same shape (cmd coverage: 12.4%).
  Two or three generics helpers (`runList`, `runDelete`) would remove ~half of `cmd/` and
  stop the copy-paste drift that produced bug 1.4. Do this before adding token/event
  commands.
- **5.2 Shared `TemplateRequest` builder.** Request building is triplicated
  (`cmd/template_create.go`, `cmd/template_update.go`, twice in
  `internal/apply/executor.go`) with two different SurveyVars/Vaults/EnvironmentIds/
  TaskParams preservation mechanisms — the invariant that already bit once. One shared
  builder used by both cmd and executor.
- **5.3 Apply sharp edges:**
  - Unresolvable name refs return `0` silently (`reconcile.go:842`) — a typo'd
    `ssh_key: my-kye` creates a repo with `SSHKeyID: 0`. Return errors naming the reference.
  - On partial failure the executor attempts dependents of failed creates. Track failed
    labels and skip dependents; add `--fail-fast`.
  - Surface field-level diffs in the plan — `needsUpdate` already computes them but returns
    bool; return changed field names into `Plan.Description`.
  - `yaml.KnownFields(true)` for apply files — a typo'd `enviroment_variables:` is silently
    dropped today, meaning secrets silently don't get applied.
  - Compare variable-group JSON semantically (unmarshal both sides) — string comparison
    causes false-positive updates on key order/whitespace (`reconcile.go:709-722`).
  - Document that merge semantics can't unset a field (empty = keep); consider a
    `field: null` convention if apply should be a full source of truth.
- **5.4 Unify config ownership under yaml.v3.** Viper lowercases keys on read while
  yaml.v3 writes verbatim — a context named `Prod` lists as `prod` and a save can create a
  duplicate `prod:` key Viper then merges unpredictably. Writes destroy comments and aren't
  atomic (no temp+rename, no lock) — concurrent `login`s can corrupt the file. Env overrides
  are already manual `os.Getenv`, so Viper earns little here. Normalize context names
  (lowercase at the boundary, as apply does) and write atomically.
- **5.5 `context.Context` + signal handling.** No `ExecuteContext`/`signal.NotifyContext`
  anywhere; Ctrl-C kills mid-apply with no resumability note and can't cancel in-flight
  HTTP. Adopt first in the `task run --wait` poll loop and the apply executor loop.
- **5.6 Server version awareness.** Client targets 2.18.20; no version detection exists.
  Runner/user-options commands 404 raw on older servers, and a schedules-list 404 aborts an
  entire apply plan even for configs without schedules. Fetch `GET /api/info` once per
  session (see §9), warn on mismatch, degrade gracefully.
- **5.7 httptest coverage for executor/reconciler.** `internal/apply/executor.go` has zero
  test coverage and reconcile's API paths are untested — the riskiest code rides on manual
  container smoke tests. Point the go-swagger transport at an `httptest.Server` with canned
  Semaphore JSON; exercises real request bodies (would have caught the SurveyVars-wipe bug
  class). Cheaper than introducing client interfaces.
- **5.8 `internal/output` calls `os.Exit(1)`** (`json.go:15`, `yaml.go:16`) — return errors
  and let `RunE` propagate.

---

## 6. Schedule Commands

Cron-based task scheduling. Project-scoped. Apply/export already reconcile schedules
declaratively; these are the imperative commands.

### API Endpoints (via `apiClient.Schedule`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GetProjectProjectIDSchedules` | `GET /project/{project_id}/schedules` | List schedules (spec-patched via `scripts/patch-spec.py`; returns `ScheduleWithTpl` incl. `tpl_name`) |
| `PostProjectProjectIDSchedules` | `POST /project/{project_id}/schedules` | Create schedule |
| `GetProjectProjectIDSchedulesScheduleID` | `GET /project/{project_id}/schedules/{schedule_id}` | Get schedule |
| `PutProjectProjectIDSchedulesScheduleID` | `PUT /project/{project_id}/schedules/{schedule_id}` | Update schedule |
| `DeleteProjectProjectIDSchedulesScheduleID` | `DELETE /project/{project_id}/schedules/{schedule_id}` | Delete schedule |

### Models

**`models.Schedule`** fields: `ID`, `Name`, `ProjectID`, `TemplateID`, `CronFormat`,
`Active` (bool), `Type` (enum: `""`, `"run_at"`), `RunAt` (datetime), `TaskParams` (*TaskPrams)

### Files to create

- `cmd/schedule.go` — parent command (alias `sched`)
- `cmd/schedule_list.go` — table: ID, Name, Template (tpl_name), Cron, Active
- `cmd/schedule_show.go` — Field/Value table
- `cmd/schedule_create.go` — flags: `--name`, `--template-id` (required), `--cron-format`, `--active`, `--type`, `--run-at`
- `cmd/schedule_update.go` — `<id> [field=value...]`: name, template_id, cron_format, active, type, run_at
- `cmd/schedule_delete.go` — standard confirmation + delete

---

## 7. Token Commands

API token management for the logged-in user. Uses `apiClient.Authentication`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GetUserTokens` | `GET /user/tokens` | List tokens |
| `PostUserTokens` | `POST /user/tokens` | Create token |
| `DeleteUserTokensAPITokenID` | `DELETE /user/tokens/{api_token_id}` | Expire/delete token |

**`models.APIToken`** fields: `ID` (string — not int64 like other resources), `UserID`,
`Created`, `Expired` (bool)

### Files to create

- `cmd/token.go` — parent command
- `cmd/token_list.go` — table: ID, User ID, Created, Expired
- `cmd/token_create.go` — prints the new token to stdout (pipeable, like `runner token`)
- `cmd/token_delete.go` — string ID; standard confirmation

### Notes
- The token ID *is* the token — treat list/create output accordingly (stdout, maskable).
- Not project-scoped (user-level). No update endpoint.
- Nice pairing with 2.9/3.6: a `token prune` or revoke-on-exit story for CI-minted tokens.

---

## 8. Event Commands

Read-only event listing. Global scope. Uses `apiClient.Operations`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GetEvents` | `GET /events` | Get events for user's projects |
| `GetEventsLast` | `GET /events/last` | Get last 200 events |

**`models.Event`** fields: `Description`, `ObjectID`, `ObjectType`, `ProjectID`, `UserID`

### Files to create

- `cmd/event.go` — parent command
- `cmd/event_list.go` — `GetEventsLast` by default, `--all` for `GetEvents`; table:
  Project ID, User ID, Object Type, Object ID, Description

---

## 9. Server Info Command

`cmd/info.go` — `semctl info`, calls `apiClient.Operations.GetInfo`, displays Semaphore
server version and configuration. Doubles as the fetch point for version awareness (5.6).

---

## 10. Remaining Interactive-Form Work

Create-command forms shipped (all 7 creates + `user create` + login/logout/user password).
Still pending:

- `task run` form (template select from API, message, git_branch, debug/dry_run/diff toggles)
- Update commands: optionally launch a pre-filled form when no `field=value` args are given
- See 4.6 for form quality improvements (API-driven selects, coverage notes)

---

## Suggested Order

1. **§1 Confirmed bugs** — small, independent, all reproduced (1.1/1.2 are one-liners).
2. **§2.1–2.4 security items** — design-level; 2.3 is a one-liner, 2.1 needs a cache-format change.
3. **§3.1–3.3 CI/CD** — exit codes, `--json` mutations, apply drift gate (biggest scripting wins).
4. **§5.1 generic helpers**, then **§6–§9 new commands** on top of them.
5. **§5.7 httptest coverage** alongside any apply work (§5.3).
6. The rest of §4/§5 opportunistically.
