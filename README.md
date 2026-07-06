# semctl

A command-line interface for managing [Semaphore UI](https://semaphoreui.com) via its REST API.

## Installation

### From source

```bash
go install github.com/ramanavelineni/semctl@latest
```

### From release

Download the latest binary from [Releases](https://github.com/ramanavelineni/semctl/releases) for your platform (Linux, macOS, Windows).

## Quick Start

```bash
# Log in to a Semaphore server (prompts for anything missing, incl. password)
semctl login --server localhost:3000 --username admin

# List projects
semctl project list

# Reference projects by ID or by name
semctl -p 1 template list
semctl -p "My Project" template list

# Run a task and wait for the result (exit code reflects task success)
semctl -p 1 task run --template-id 5 --message "Deploy v2.1" --wait

# Stream task output while it runs
semctl -p 1 task run --template-id 5 --follow

# Check output of a past task
semctl -p 1 task output 42
```

The password is used once to create an API token and is **not stored** (pass
`--save-password` to opt in to automatic re-login). For scripting, prefer
`--password-stdin` over `--password` so the password stays out of shell
history.

## Commands

| Command | Description |
|---------|-------------|
| `login` | Log in to a Semaphore server |
| `logout` | Log out and remove saved credentials |
| `context` | Manage multiple server contexts |
| `project` | List, show, create, delete projects |
| `template` | Manage task templates (list, show, create, update, delete) |
| `task` | Run, stop, list, and view task output |
| `key` | Manage SSH keys and login credentials |
| `inventory` | Manage host inventories |
| `repo` | Manage Git repositories |
| `env` | Manage variable groups (environments) |
| `apply` | Apply declarative configuration files |
| `export` | Export project state to YAML/JSON |
| `validate` | Validate configuration files offline |
| `version` | Show CLI version |
| `completion` | Generate shell completions (bash, zsh, fish, powershell) |

## Declarative Configuration

Define your entire Semaphore project as code and apply it:

```bash
semctl apply -f project.yaml              # apply a config file
semctl apply -f project.yaml --dry-run    # preview changes without applying
semctl apply -f ./config/                 # apply all files in a directory
semctl apply -f project.yaml --skip-schedules  # re-apply without duplicating schedules
semctl export -p 1 -o exported.yaml       # export existing project to YAML
semctl validate -f project.yaml           # validate config file offline
```

Resources are matched by name and created, updated, or deleted as needed. Updates merge over existing state — fields you omit keep their server-side values. `${VAR}` environment references are expanded before parsing; referencing an unset variable is an error (use `$${VAR}` for a literal). Set `state: absent` on any resource to delete it, or `project_state: absent` to delete the entire project and all its resources.

Note: the Semaphore API has no schedule list endpoint, so schedules cannot be reconciled — each apply creates them anew. Use `--skip-schedules` on repeat applies.

- [Example configuration](docs/example.yaml)
- [Full schema reference](docs/apply-schema.md)

## CI/CD Usage

semctl runs configless in CI — no config file or login step needed. Create an
API token once (Semaphore UI → User Settings → API Tokens, or take the token
`semctl login` caches) and set:

```yaml
# GitHub Actions example
env:
  SEMCTL_SERVER: sem.example.com:3000
  SEMCTL_SCHEME: https
  SEMCTL_API_TOKEN: ${{ secrets.SEMAPHORE_TOKEN }}

steps:
  - run: semctl apply -f semaphore/ --yes --skip-schedules
  - run: semctl -p "My Project" task run --template-id 5 --wait --wait-timeout 30m
```

- `--wait` / `--follow` make `task run` block until the task finishes; the exit
  code reflects the task result, so pipelines can gate on it.
- Prompts never hang in CI: without a TTY, any command that would ask for
  confirmation fails fast unless `--yes` is passed, and a declined or cancelled
  confirmation always exits non-zero.
- `SEMCTL_AUTH_USERNAME` / `SEMCTL_AUTH_PASSWORD` are supported as an
  alternative to a token (semctl logs in and creates a token per run).
- All HTTP requests have a 30s timeout by default; tune with `--timeout`.

## Multi-Server Contexts

Manage multiple Semaphore servers without re-entering credentials:

```bash
semctl login --server prod.example.com --username admin
semctl context rename default prod

semctl login --server staging.example.com --username admin
semctl context rename default staging

semctl context use prod       # switch to production
semctl context list           # see all contexts
semctl --context staging project list   # one-off context override
```

`semctl logout` revokes the context's API token server-side and removes local
credentials.

## Output Formats

All list/show commands support table, JSON, and YAML output:

```bash
semctl project list              # table (default)
semctl project list --json       # JSON
semctl project list --yaml       # YAML
semctl project show 1 --json     # single resource as JSON
```

## Configuration

Config file locations (searched in order):
1. `./semctl.yaml` (current directory)
2. `~/.config/semctl/config.yaml`

```yaml
current_context: "default"
contexts:
  default:
    server:
      host: "127.0.0.1"
      port: 3000
      scheme: "http"
      # insecure_skip_verify: true      # skip TLS verification (not recommended)
      # ca_cert: /path/to/ca.pem        # custom CA for TLS verification
    auth:
      username: "admin"
      # api_token: ""    # set explicitly, or let 'semctl login' cache one
      # password: ""     # only stored by 'semctl login --save-password'
defaults:
  project_id: 1
```

`semctl login` stores the server and username here and caches the API token
under `~/.cache/semctl/tokens/`; the password is not written unless you pass
`--save-password`.

### Environment variables

| Variable | Meaning |
|----------|---------|
| `SEMCTL_SERVER` | Server `host:port` (overrides context config) |
| `SEMCTL_SCHEME` | `http` or `https` |
| `SEMCTL_API_TOKEN` | API token (highest-precedence auth) |
| `SEMCTL_AUTH_USERNAME` / `SEMCTL_AUTH_PASSWORD` | Username/password auth |

Precedence: command-line flags > environment variables > config file.

## Global Flags

```
-p, --project string    project ID or name for project-scoped commands
-c, --config string     path to config file
    --context string    use a specific context
-s, --server string     override server host:port for this invocation
    --timeout duration  HTTP request timeout (default 30s)
    --insecure          skip TLS certificate verification (not recommended)
    --ca-cert string    path to a CA certificate for TLS verification
    --json              output as JSON
    --yaml              output as YAML
    --no-color          disable colored output
-y, --yes               auto-confirm prompts
```

## Compatibility

| semctl | Semaphore UI | Notes |
|--------|--------------|-------|
| v0.1.0 | v2.16.51     | Initial release |
| v0.2.0 | v2.16.51     | Declarative apply, variable groups, validate |
| v0.3.0 | v2.16.51     | Security & CI/CD hardening: env var auth, token-only login, `task run --wait`, strict apply semantics. Breaking: `--project` is now string (ID or name), cancelled prompts exit non-zero, unset `${VAR}` in apply configs is an error |

The API client is generated from the Semaphore UI OpenAPI spec. Each semctl release is tested against the listed Semaphore UI version. Older or newer versions of Semaphore UI may work but are not guaranteed.

## Building from Source

```bash
git clone https://github.com/ramanavelineni/semctl.git
cd semctl
make build    # produces ./semctl binary
make test     # run tests
make install  # install to $GOPATH/bin
```
