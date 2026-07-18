# semctl Roadmap

Pending features, fixes, and hardening work. Updated 2026-07-12 after a full code review
(security / UX / CI-CD / architecture). The confirmed-bug batch (login -s shadowing,
--no-color no-op, typo subcommands exiting 0, empty-list exit inconsistency, key update
secret wipe, form stdin TTY gate, huh abort mapping, go-openapi DEBUG dumping, export
perms) shipped on fix/cli-bug-batch and has been removed from this list.

---

## 1. Security Hardening

The design-level trio (token-server binding, post-parse `${VAR}` expansion, context-name
validation) shipped on fix/security-batch. Remaining items:

- **1.1 Env-credential redirect via CWD config (partial mitigation shipped).** The
  server-binding fix refuses a cached token / re-login when a `--server`/`SEMCTL_SERVER`
  override points away from the context's configured server, and warns when env credentials
  are used with a working-directory config. Fully closing the residual path (a CWD
  `semctl.yaml` that changes the resolved server while `SEMCTL_AUTH_USERNAME/PASSWORD` are
  set — the login itself still goes to the config's server) needs opt-in trust for CWD
  configs (direnv-style: prompt once, remember the path).
- **1.2 `env show --json` prints the password the table masks** (`cmd/env_show.go:46-54`).
  Pick one policy; redact in all formats.
- **1.3 Non-argv secret input for key/env commands.** `key create --private-key/--passphrase`
  and `env create --password` only accept secrets via argv (ps/shell-history leak); the
  key create Example even teaches `--private-key "$(cat ~/.ssh/id_rsa)"`. Add
  `--private-key-file` and `--password-stdin` variants.
- **1.4 Warn when TLS verification is disabled via config.** `server.insecure_skip_verify`
  from a config file (including a CWD config) silently disables verification
  (`client.go:62-67`); only the `--insecure` flag is an explicit choice. Emit a warning once
  per invocation whatever the source.
- **1.5 Token-cache lifecycle gaps.** `context rename` orphans a still-valid cached token;
  `context delete` removes the cache without server-side revocation (contrast `logout`,
  which revokes). Rename should move the cache file; delete should attempt revocation.
- **1.6 Tighten existing config file perms on write.** `os.WriteFile(path, out, 0600)`
  leaves a pre-existing 0644 file at 0644 (`config.go:478`) — `login --save-password` into it
  stores the password world-readable. `os.Chmod(path, 0600)` after write.
- **1.7 `getCacheDir` ignores `os.UserHomeDir` error** (`client.go:330`) — on failure the
  token lands in a relative `.cache/` under the CWD. Return an error instead.

---

## 2. CI/CD Friendliness

The trio of distinct exit codes (documented in `semctl --help` and README), `--json` output
for `task run`/create commands, and the `apply --detailed-exitcode` + JSON plan drift gate
shipped on feat/cicd-friendliness. Remaining items:

- **2.1 `--quiet` flag** suppressing `style.Success/Info` (keep Error/Warning) — today the
  only silencer is `2>/dev/null`, which also hides real errors.
- **2.2 `--wait-timeout` defaults to 0 = wait forever** (`cmd/task_run.go:187`) — pick a
  sane default or warn when waiting unbounded in non-TTY mode. Consider opt-in
  retry/backoff for transient 5xx (only 401 is retried today).
- **2.3 Ephemeral-token leak on username/password auth.** Each cookie login mints a
  server-side API token that is never revoked; on a read-only filesystem the ignored cache
  write (`client.go:146-150`) means every command mints another. Document
  `SEMCTL_API_TOKEN` as the CI path; consider revoking ephemeral tokens at exit.
- **2.4 Own CI hardening** (`.github/workflows/`): `go test -race -cover`, govulncheck,
  dependabot, pin golangci-lint version (currently `latest`), cross-compile darwin/windows
  in CI (`goreleaser build --snapshot`), gate release.yml on lint too. goreleaser: consider
  homebrew tap / docker image / SBOM+signing if distributing publicly.

---

## 3. UX Polish

- **3.1 Dynamic shell completion.** Zero `ValidArgsFunction` in the repo. Highest value:
  context names for `context use` (purely local), project names for `-p`, field names for
  update commands, template/task IDs.
- **3.2 API error translation.** Raw go-swagger errors reach users
  (`[GET /project/{project_id}/…] …NotFound (status 404): {…}`). One `errors.As` helper
  mapping 404 → "repository 5 not found in project 3", 401 → auth hint, 400 → server
  message. Include the server/context identity in API errors for multi-context users.
- **3.3 Consistency fixes:**
  - `project update` takes no positional ID while every other update does — accept one.
  - Name resolution works for `-p` but nowhere else (`project show myproj` fails) — accept
    names in positional resource args.
  - Unified `--output table|json|yaml` (keep `--json`/`--yaml` as aliases; error on
    conflict — today `--json --yaml` silently picks JSON). Note `export -o` means *file*.
  - Accept kebab-case in `field=value` args (create flags are kebab, update fields snake).
  - Standardize update-arg validation on the friendly `no fields to update` message
    (user/runner use bare cobra `requires at least 2 arg(s)`).
  - Usage brackets: `context use [name]` → `<name>` (args are required).
- **3.4 Missing confirmations:** `task stop` (kills a running deployment), `runner
  deactivate`, `runner clear-cache`.
- **3.5 Help text:** `template update` Long omits the `arguments` field and its
  unknown-field error doesn't list valid fields (every sibling does); document
  `user show me`; add a root-level Example showing the login → project list → task run flow.
- **3.6 Form improvements:** populate repository/inventory/environment selects from the API
  in `template create` (the two-stage `key create` form is the pattern to copy); note in
  forms that more options exist as flags; add a form to `task run`; optional pre-filled
  forms for update commands when no `field=value` args are given.
- **3.7 Table rendering:** wrap/truncate long cells (`WrapNone` today produces enormous
  lines for env JSON); render nil ints as `-`/empty, not `0` (nil `MaxParallelTasks`
  currently indistinguishable from explicit 0).

---

## 4. Architecture / Tech Debt

Generic command helpers shipped on refactor/generic-cmd-helpers: `runList`/`runShow`/
`runDelete` + `parseIDArg` in `cmd/resource_helpers.go`, all 27 list/show/delete commands
migrated (~360 lines removed). httptest coverage for the apply executor/reconciler
shipped on test/apply-httptest (fake in-memory Semaphore server; apply package 29%→65%).
Remaining:

- **4.1 Shared `TemplateRequest` builder.** Request building is triplicated
  (`cmd/template_create.go`, `cmd/template_update.go`, twice in
  `internal/apply/executor.go`) with two different SurveyVars/Vaults/EnvironmentIds/
  TaskParams preservation mechanisms — the invariant that already bit once. One shared
  builder used by both cmd and executor.
- **4.2 Apply sharp edges:**
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
- **4.3 Unify config ownership under yaml.v3.** Viper lowercases keys on read while
  yaml.v3 writes verbatim — a context named `Prod` lists as `prod` and a save can create a
  duplicate `prod:` key Viper then merges unpredictably. Writes destroy comments and aren't
  atomic (no temp+rename, no lock) — concurrent `login`s can corrupt the file. Env overrides
  are already manual `os.Getenv`, so Viper earns little here. Normalize context names
  (lowercase at the boundary, as apply does) and write atomically.
- **4.4 `context.Context` + signal handling.** No `ExecuteContext`/`signal.NotifyContext`
  anywhere; Ctrl-C kills mid-apply with no resumability note and can't cancel in-flight
  HTTP. Adopt first in the `task run --wait` poll loop and the apply executor loop.
- **4.5 `internal/output` calls `os.Exit(1)`** (`json.go:15`, `yaml.go:16`) — return errors
  and let `RunE` propagate.

---

Schedule, token, event, and info commands (old §5–§8) shipped on
feat/new-resource-commands — all built on the cmd/resource_helpers.go pattern and
live-tested against v2.18.20 (`semctl info` also unlocks version awareness, §4.5).

## 5. Remaining Interactive-Form Work

Create-command forms shipped (all 7 creates + `user create` + login/logout/user password).
Still pending:

- `task run` form (template select from API, message, git_branch, debug/dry_run/diff toggles)
- Update commands: optionally launch a pre-filled form when no `field=value` args are given
- See 3.6 for form quality improvements (API-driven selects, coverage notes)

---

## Suggested Order

1. The rest of §1/§2/§3/§4/§5 opportunistically — no single item is blocking.
