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
# Log in to a Semaphore server
semctl login --server localhost:3000 --username admin --password changeme

# List projects
semctl project list

# Set a default project
semctl project list --json   # find your project ID
# Then use -p flag or set defaults.project_id in config

# List templates in a project
semctl -p 1 template list

# Run a task
semctl -p 1 task run --template-id 5 --message "Deploy v2.1"

# Check task output
semctl -p 1 task output 42
```

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
semctl export -p 1 -o exported.yaml       # export existing project to YAML
semctl validate -f project.yaml           # validate config file offline
```

Resources are matched by name and created, updated, or deleted as needed. Environment variables (`${VAR}`) are expanded before parsing. Set `state: absent` on any resource to delete it, or `project_state: absent` to delete the entire project and all its resources.

- [Example configuration](docs/example.yaml)
- [Full schema reference](docs/apply-schema.md)

## Multi-Server Contexts

Manage multiple Semaphore servers without re-entering credentials:

```bash
semctl login --server prod.example.com --username admin --password secret
semctl context rename default prod

semctl login --server staging.example.com --username admin --password secret
semctl context rename default staging

semctl context use prod       # switch to production
semctl context list           # see all contexts
semctl --context staging project list   # one-off context override
```

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
    auth:
      username: "admin"
      password: "changeme"
      # api_token: ""    # alternative to username/password
defaults:
  project_id: 1
```

Environment variables with `SEMCTL_` prefix are also supported.

## Global Flags

```
-p, --project int32    project ID for project-scoped commands
-c, --config string    path to config file
    --context string   use a specific context
    --json             output as JSON
    --yaml             output as YAML
    --no-color         disable colored output
-y, --yes              auto-confirm prompts
```

## Building from Source

```bash
git clone https://github.com/ramanavelineni/semctl.git
cd semctl
make build    # produces ./semctl binary
make test     # run tests
make install  # install to $GOPATH/bin
```
