# semctl Roadmap

Pending features, fixes, and hardening work. Updated 2026-07-12 after a full code review
(security / UX / CI-CD / architecture). The confirmed-bug batch (login -s shadowing,
--no-color no-op, typo subcommands exiting 0, empty-list exit inconsistency, key update
secret wipe, form stdin TTY gate, huh abort mapping, go-openapi DEBUG dumping, export
perms) shipped on fix/cli-bug-batch and has been removed from this list.

---

## 1. Security Hardening

The design-level trio (token-server binding, post-parse `${VAR}` expansion, context-name
validation) shipped on fix/security-batch; polish batch 1 closed env-show redaction,
the TLS-disabled warning, token-cache lifecycle on rename/delete, config chmod, and
the cache-dir home error. Remaining items:

- **1.1 Env-credential redirect via CWD config (partial mitigation shipped).** The
  server-binding fix refuses a cached token / re-login when a `--server`/`SEMCTL_SERVER`
  override points away from the context's configured server, and warns when env credentials
  are used with a working-directory config. Fully closing the residual path (a CWD
  `semctl.yaml` that changes the resolved server while `SEMCTL_AUTH_USERNAME/PASSWORD` are
  set — the login itself still goes to the config's server) needs opt-in trust for CWD
  configs (direnv-style: prompt once, remember the path).

---

## 3. UX Polish

- **3.1 Remaining consistency decisions.** Name resolution in positional args
  (`project show myproj`) and a unified `--output table|json|yaml` flag (today
  `--json --yaml` silently picks JSON; note `export -o` means *file*). The mechanical
  items (positional ID for project update, kebab-case field=value, standardized
  update-arg validation, usage brackets) shipped in polish batch A.
- **3.2 Form improvements:** populate repository/inventory/environment selects from the API
  in `template create` (the two-stage `key create` form is the pattern to copy); note in
  forms that more options exist as flags; add a form to `task run`; optional pre-filled
  forms for update commands when no `field=value` args are given.

---

## 4. Architecture / Tech Debt

Generic command helpers shipped on refactor/generic-cmd-helpers: `runList`/`runShow`/
`runDelete` + `parseIDArg` in `cmd/resource_helpers.go`, all 27 list/show/delete commands
migrated (~360 lines removed). httptest coverage for the apply executor/reconciler
shipped on test/apply-httptest (fake in-memory Semaphore server; apply package 29%→65%).
Remaining:

- **4.1 Unify config ownership under yaml.v3.** Viper lowercases keys on read while
  yaml.v3 writes verbatim — a context named `Prod` lists as `prod` and a save can create a
  duplicate `prod:` key Viper then merges unpredictably. Writes destroy comments and aren't
  atomic (no temp+rename, no lock) — concurrent `login`s can corrupt the file. Env overrides
  are already manual `os.Getenv`, so Viper earns little here. Normalize context names
  (lowercase at the boundary, as apply does) and write atomically.
- **4.2 `context.Context` + signal handling.** No `ExecuteContext`/`signal.NotifyContext`
  anywhere; Ctrl-C kills mid-apply with no resumability note and can't cancel in-flight
  HTTP. Adopt first in the `task run --wait` poll loop and the apply executor loop.

---

Schedule, token, event, and info commands (old §5–§8) shipped on
feat/new-resource-commands — all built on the cmd/resource_helpers.go pattern and
live-tested against v2.18.20 (`semctl info` also unlocks version awareness, §4.5).

## 5. Remaining Interactive-Form Work

Create-command forms shipped (all 7 creates + `user create` + login/logout/user password).
Still pending:

- `task run` form (template select from API, message, git_branch, debug/dry_run/diff toggles)
- Update commands: optionally launch a pre-filled form when no `field=value` args are given
- See 3.2 for form quality improvements (API-driven selects, coverage notes)

---

## Suggested Order

1. The rest of §1/§2/§3/§4/§5 opportunistically — no single item is blocking.
