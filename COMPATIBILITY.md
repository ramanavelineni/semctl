# Compatibility

semctl currently targets **Semaphore UI 2.18.x**. The API client
(`pkg/semapi/`) is generated from the Semaphore UI OpenAPI spec; each semctl
release is tested against the Semaphore UI version listed below. Older or
newer versions may work but are not guaranteed — semctl warns when the server
version differs from its target (`semctl info` shows the server version), and
degrades gracefully on pre-2.18 servers (schedule reconciliation is skipped
with a warning instead of failing; missing APIs get a version hint on 404).

## Release matrix

| semctl | Semaphore UI | Notes |
|--------|--------------|-------|
| v0.1.0 | v2.16.51     | Initial release |
| v0.2.0 | v2.16.51     | Declarative apply, variable groups, validate |
| v0.3.0 | v2.16.51     | Security & CI/CD hardening: env var auth, token-only login, `task run --wait`, strict apply semantics. Breaking: `--project` is now string (ID or name), cancelled prompts exit non-zero, unset `${VAR}` in apply configs is an error |
| v0.4.0 | v2.16.51     | Schedule reconciliation: apply diffs/updates/deletes schedules by name (no more duplicates on re-apply); export includes schedules |
| v0.5.0 | v2.18.20     | API client regenerated for Semaphore 2.18; template multi-variable-group assignments and task params preserved on update |
| v0.6.0 | v2.18.20     | Runner management commands (global and project-scoped) |
| v0.7.0 | v2.18.20     | User management commands, per-user options, interactive forms on all create commands |
| v0.8.0 | v2.18.20     | Schedule/token/event/info commands; CLI bug-fix batch (`login -s`, `--no-color`, exit codes on typos/empty lists, `key update` secret guard); security hardening (server-bound token cache, post-parse `${VAR}` expansion, context-name validation); distinct exit codes + `--json` for mutations; `apply --detailed-exitcode` drift gate; generic command helpers |
| v0.9.0 | v2.18.20     | Server version awareness: mismatch warning on apply, graceful schedule handling + friendly 404 hints on pre-2.18 servers; apply executor/reconciler test suite |
| v0.10.0 | v2.18.20    | Human-readable API errors with server identity; dynamic shell completion; `--quiet`; secret file/stdin input for key/env create; apply hardening (plan-time reference validation, `--fail-fast`, field-level plan diffs, strict config parsing, semantic variable comparison); table cell wrapping; CI hardening (govulncheck, race detector, pinned toolchain go1.26.5); dependency refresh (huh 1.0, go-openapi 0.27) |
| v0.11.0 | v2.18.20    | Positional args accept resource names everywhere; global `--output table\|json\|yaml` flag (conflicting formats error; `export -o` deprecated in favor of `-f`); interactive forms batch (API-driven selects in `template create`, `task run` form, pre-filled forms on update commands); config owned end-to-end by yaml.v3 (Viper dropped, atomic locked writes, corrupt configs refused); graceful Ctrl-C/SIGTERM (in-flight HTTP cancelled, apply/task-wait stop with resume notes, interrupts exit 5); working-directory configs require opt-in trust (`semctl trust`, direnv-style). Breaking: an untrusted `./semctl.yaml` is ignored until trusted (pass `--config` explicitly or run `semctl trust`); context names are now case-insensitive — a config defining both `Prod` and `prod` fails to load |

## Upgrade notes

> **Upgrading Semaphore UI to 2.18?** BoltDB support was removed (replaced by
> sqlite) — the official Docker image fails to start with
> `SEMAPHORE_DB_DIALECT=bolt`. Back up your `database.boltdb` and plan the
> migration before upgrading a 2.16 server.
