# semctl Roadmap

Pending features and implementation details for future development.

## 1. Interactive Forms for Create/Update Commands

Add `huh` form-based interactive mode to all create and update commands, using the existing `shouldAutoInteractive()` + `newForm()` helpers in `cmd/form.go`.

### Pattern

```go
// In each create command, after reading flags:
inputsMissing := name == "" || otherRequired == ""
interactive, err := shouldAutoInteractive(cmd, inputsMissing)
if err != nil { return err }
if interactive {
    form := newForm(huh.NewGroup(
        huh.NewInput().Title("Name").Value(&name),
        // ... other fields
    ).Title("Create <Resource>"))
    if err := form.Run(); err != nil { return err }
}
```

### Commands to add interactive mode

| Command | Required fields | Optional fields |
|---------|----------------|-----------------|
| `project create` | name | type, alert, alert_chat, max_parallel_tasks |
| `template create` | name, repository_id | type, app, playbook, git_branch, description, environment_id, inventory_id, build_template_id, view_id, autorun |
| `task run` | template_id | message, git_branch, arguments, environment, limit, playbook, debug, dry_run, diff |
| `key create` | name, type | login, password, private_key, passphrase (conditional on type) |
| `inventory create` | name, type | inventory, ssh_key_id, become_key_id, repository_id |
| `repo create` | name, git_url | git_branch, ssh_key_id |
| `env create` | name | json_vars, env, password |

### Notes
- Key create needs conditional fields: show login/password for `login_password` type, show login/private_key/passphrase for `ssh` type
- Template create could offer a select dropdown for type (e.g., "", "deploy", "build") and app (e.g., "ansible", "terraform", "bash")
- Update commands could optionally launch an interactive form pre-filled with current values when no `field=value` args are provided

---

## 2. Schedule Commands

Cron-based task scheduling. Project-scoped resource.

### API Endpoints (via `apiClient.Schedule`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `PostProjectProjectIDSchedules` | `POST /project/{project_id}/schedules` | Create schedule |
| `GetProjectProjectIDSchedulesScheduleID` | `GET /project/{project_id}/schedules/{schedule_id}` | Get schedule |
| `PutProjectProjectIDSchedulesScheduleID` | `PUT /project/{project_id}/schedules/{schedule_id}` | Update schedule |
| `DeleteProjectProjectIDSchedulesScheduleID` | `DELETE /project/{project_id}/schedules/{schedule_id}` | Delete schedule |

**Note:** No list endpoint in the generated client. Schedules may need to be fetched via project events or a different mechanism. Verify against Semaphore docs.

### Models

**`models.Schedule`** fields: `ID`, `Name`, `ProjectID`, `TemplateID`, `CronFormat`, `Active` (bool), `Type` (enum: `""`, `"run_at"`), `RunAt` (datetime), `TaskParams` (*TaskPrams)

**`models.ScheduleRequest`** fields: same as Schedule

**`models.TaskPrams`** fields: `Arguments`, `Environment`, `GitBranch`, `Message`, `Params` (embedded AnsibleTaskParams + TerraformTaskParams)

### Files to create

**`cmd/schedule.go`** — Parent command
```go
var scheduleCmd = &cobra.Command{
    Use:     "schedule",
    Short:   "Manage schedules",
    Aliases: []string{"sched"},
}
func init() { rootCmd.AddCommand(scheduleCmd) }
```

**`cmd/schedule_show.go`** — `semctl schedule show <id>`
- Calls `GetProjectProjectIDSchedulesScheduleID`
- Field/Value table: ID, Name, TemplateID, CronFormat, Active, Type, RunAt

**`cmd/schedule_create.go`** — `semctl schedule create`
- Flags: `--name` (required), `--template-id` (required), `--cron-format`, `--active` (default true), `--type` (default ""), `--run-at`
- Body: `&models.ScheduleRequest{Name, TemplateID, CronFormat, Active, Type, RunAt, ProjectID}`

**`cmd/schedule_update.go`** — `semctl schedule update <id> [field=value...]`
- Fetch current via GET, apply field=value overrides
- Fields: name, template_id, cron_format, active, type, run_at

**`cmd/schedule_delete.go`** — `semctl schedule delete <id>`
- Standard confirmation + delete pattern

### Verification
```bash
go build ./... && go vet ./...
```

---

## 3. User Commands (Admin)

User management. Global scope (not project-scoped). Requires admin privileges.

### API Endpoints (via `apiClient.User`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GetUsers` | `GET /users` | List all users |
| `GetUsersUserID` | `GET /users/{user_id}/` | Get user by ID |
| `GetUser` | `GET /user/` | Get logged-in user (self) |
| `PostUsers` | `POST /users` | Create user |
| `PutUsersUserID` | `PUT /users/{user_id}/` | Update user |
| `PostUsersUserIDPassword` | `POST /users/{user_id}/password` | Change password |
| `DeleteUsersUserID` | `DELETE /users/{user_id}/` | Delete user |

### Models

**`models.User`** fields: `ID`, `Name`, `Username`, `Email`, `Admin` (bool), `Alert` (bool), `External` (bool), `Created`

**`models.UserRequest`** fields: `Name`, `Username`, `Email`, `Password` (strfmt.Password), `Admin` (bool), `Alert` (bool), `External` (bool)

### Files to create

**`cmd/user.go`** — Parent command

**`cmd/user_list.go`** — `semctl user list`
- Calls `GetUsers`
- Table: ID, Name, Username, Email, Admin, Created

**`cmd/user_show.go`** — `semctl user show <id>`
- Calls `GetUsersUserID`
- Field/Value table: ID, Name, Username, Email, Admin, Alert, External, Created

**`cmd/user_whoami.go`** — `semctl user whoami`
- Calls `GetUser` (logged-in user)
- Field/Value table: same as show

**`cmd/user_create.go`** — `semctl user create`
- Flags: `--name` (required), `--username` (required), `--email` (required), `--password` (required), `--admin` (bool), `--alert` (bool)
- Body: `&models.UserRequest{...}`

**`cmd/user_update.go`** — `semctl user update <id> [field=value...]`
- Fetch current via `GetUsersUserID`, apply overrides
- Fields: name, username, email, admin, alert, external

**`cmd/user_password.go`** — `semctl user password <id>`
- Flags: `--password` (required, or prompt from stdin)
- Calls `PostUsersUserIDPassword`

**`cmd/user_delete.go`** — `semctl user delete <id>`
- Standard confirmation + delete pattern

### Verification
```bash
go build ./... && go vet ./...
```

---

## 4. Token Commands

API token management for the logged-in user. Uses `apiClient.Authentication`.

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GetUserTokens` | `GET /user/tokens` | List tokens |
| `PostUserTokens` | `POST /user/tokens` | Create token |
| `DeleteUserTokensAPITokenID` | `DELETE /user/tokens/{api_token_id}` | Expire/delete token |

### Models

**`models.APIToken`** fields: `ID` (string), `UserID`, `Created`, `Expired` (bool)

### Files to create

**`cmd/token.go`** — Parent command

**`cmd/token_list.go`** — `semctl token list`
- Calls `GetUserTokens`
- Table: ID, User ID, Created, Expired

**`cmd/token_create.go`** — `semctl token create`
- Calls `PostUserTokens`
- Prints the new token ID on success
- No flags needed (API creates with defaults)

**`cmd/token_delete.go`** — `semctl token delete <token_id>`
- `token_id` is a string (not int64)
- Calls `DeleteUserTokensAPITokenID`
- Standard confirmation pattern

### Notes
- Token ID is a string, not int64 (unlike other resources)
- No update endpoint exists
- `token_list` and `token_create` are not project-scoped (user-level)

### Verification
```bash
go build ./... && go vet ./...
```

---

## 5. Event Commands

Read-only event listing. Global scope.

### API Endpoints (via `apiClient.Operations`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GetEvents` | `GET /events` | Get events for user's projects |
| `GetEventsLast` | `GET /events/last` | Get last 200 events |

### Models

**`models.Event`** fields: `Description`, `ObjectID`, `ObjectType`, `ProjectID`, `UserID`

### Files to create

**`cmd/event.go`** — Parent command

**`cmd/event_list.go`** — `semctl event list`
- Calls `GetEventsLast` by default
- Flag: `--all` to use `GetEvents` instead
- Table: Project ID, User ID, Object Type, Object ID, Description

### Notes
- Read-only resource (no create/update/delete)
- Event model is simple (no timestamps in model, might be in response headers)
- Uses `apiClient.Operations` (not a dedicated Event client)

### Verification
```bash
go build ./... && go vet ./...
```

---

## 6. Server Info Command

Utility command using the operations client.

### Files to create

**`cmd/info.go`** — `semctl info`
- Calls `apiClient.Operations.GetInfo`
- Displays Semaphore server version and configuration

---

## 7. Apply Command — Declarative Resource Management

Apply a YAML manifest to create, update, or delete resources in Semaphore UI. Inspired by `kubectl apply`.

### Usage

```bash
semctl apply -f resources.yaml              # apply from file
semctl apply -f resources.yaml --dry-run    # preview changes without applying
semctl apply -f resources.yaml --delete     # delete resources defined in the file
```

### Manifest Format

```yaml
project_id: 1

keys:
  - name: "Deploy Key"
    type: ssh
    login: deploy
    private_key: |
      -----BEGIN OPENSSH PRIVATE KEY-----
      ...

repositories:
  - name: "Main Repo"
    git_url: "git@github.com:org/app.git"
    git_branch: main
    ssh_key: "Deploy Key"    # reference by name

inventories:
  - name: "Production Hosts"
    type: static-yaml
    inventory: |
      all:
        hosts:
          10.0.0.1:
    ssh_key: "Deploy Key"

environments:
  - name: "Production"
    json: '{"db_host": "10.0.0.5"}'
    password: "vault-pass"

templates:
  - name: "Deploy App"
    type: deploy
    app: ansible
    playbook: deploy.yml
    repository: "Main Repo"       # reference by name
    inventory: "Production Hosts"
    environment: "Production"
    autorun: false

schedules:
  - name: "Nightly Deploy"
    template: "Deploy App"        # reference by name
    cron_format: "0 2 * * *"
    active: true
```

### Implementation Details

**`cmd/apply.go`** — `semctl apply`
- Flag: `-f, --file` (required) — path to YAML manifest
- Flag: `--dry-run` — print planned actions without executing
- Flag: `--delete` — delete resources defined in the manifest

**Core logic (`internal/apply/apply.go`):**
1. Parse YAML manifest into typed structs
2. Resolve `project_id` from manifest or `--project` flag
3. For each resource type (in dependency order):
   - Fetch existing resources from API
   - Match by name (primary identifier for idempotency)
   - Determine action: create (new), update (changed), skip (unchanged)
4. Execute actions, printing status per resource:
   ```
   key "Deploy Key"              created (id: 5)
   repo "Main Repo"              unchanged
   inventory "Production Hosts"  updated (id: 3)
   template "Deploy App"         created (id: 12)
   ```

**Resource processing order** (respects dependencies):
1. Keys (no dependencies)
2. Repositories (depend on keys via `ssh_key`)
3. Inventories (depend on keys, optionally repositories)
4. Environments (no dependencies)
5. Templates (depend on repositories, inventories, environments)
6. Schedules (depend on templates)

**Name-based references:**
- Resources reference each other by `name` instead of ID (e.g., `ssh_key: "Deploy Key"`)
- Resolver maps names to IDs after fetching/creating upstream resources
- Error if a referenced name doesn't exist and isn't defined in the manifest

**`--delete` mode:**
- Process in reverse dependency order (schedules first, keys last)
- Only deletes resources whose names appear in the manifest
- Confirmation prompt unless `--yes` is passed

### Files to create

| File | Purpose |
|------|---------|
| `cmd/apply.go` | Command definition, flag parsing, orchestration |
| `internal/apply/apply.go` | Core apply engine: parse, diff, execute |
| `internal/apply/manifest.go` | Manifest struct definitions and YAML parsing |
| `internal/apply/resolver.go` | Name-to-ID resolution across resource types |

---

## 8. Export Command — Dump All Resources to File

Export all project resources into a single YAML or JSON file, suitable for backup, version control, or re-import via `semctl apply`.

### Usage

```bash
semctl export                           # YAML to stdout (default)
semctl export -o project.yaml           # YAML to file
semctl export -o project.json --json    # JSON to file
semctl export --project 1               # specific project
```

### Output Format

The export produces a manifest compatible with `semctl apply`:

```yaml
project_id: 1

keys:
  - name: "Deploy Key"
    type: ssh
    login: deploy
    # private_key omitted (sensitive, not returned by API)

repositories:
  - name: "Main Repo"
    git_url: "git@github.com:org/app.git"
    git_branch: main
    ssh_key: "Deploy Key"

inventories:
  - name: "Production Hosts"
    type: static-yaml
    inventory: |
      all:
        hosts:
          10.0.0.1:
    ssh_key: "Deploy Key"

environments:
  - name: "Production"
    json: '{"db_host": "10.0.0.5"}'
    # password omitted (sensitive)

templates:
  - name: "Deploy App"
    type: deploy
    app: ansible
    playbook: deploy.yml
    repository: "Main Repo"
    inventory: "Production Hosts"
    environment: "Production"
    autorun: false

schedules:
  - name: "Nightly Deploy"
    template: "Deploy App"
    cron_format: "0 2 * * *"
    active: true
```

### Implementation Details

**`cmd/export.go`** — `semctl export`
- Flag: `-o, --output` — file path (default: stdout)
- Uses `--json` / `--yaml` global flags for format (default YAML)
- Uses `getProjectID(cmd)` for project scope

**Core logic:**
1. Fetch all resources in parallel (keys, repos, inventories, envs, templates, schedules)
2. Build ID-to-name maps for cross-referencing
3. Replace foreign key IDs with name references (e.g., `ssh_key_id: 5` → `ssh_key: "Deploy Key"`)
4. Omit sensitive fields (private keys, passwords) with a comment noting they're excluded
5. Marshal to YAML or JSON
6. Write to file or stdout

**Sensitive field handling:**
- Keys: omit `private_key`, `passphrase`, `password`
- Environments: omit `password`
- Add a top-level comment: `# Sensitive fields (private_key, password) are omitted. Fill in before applying.`

### Files to create

| File | Purpose |
|------|---------|
| `cmd/export.go` | Command definition, flag parsing, output writing |
| `internal/apply/export.go` | Fetch all resources, build manifest, resolve names |

### Notes
- Export and apply share the manifest struct definitions (`internal/apply/manifest.go`)
- Exported manifests are idempotent — re-applying an unchanged export is a no-op
- Foreign key resolution is the inverse of apply: IDs → names (export) vs names → IDs (apply)

---

## Implementation Order

1. **Export** (2 files) — enables backup/audit of existing Semaphore state
2. **Apply** (4 files) — declarative management, depends on shared manifest types with export
3. **Schedule** (5 files) — most useful for automation workflows
4. **User** (8 files) — admin management
5. **Token** (4 files) — lightweight, useful for auth management
6. **Event** (2 files) — read-only, quick to implement
7. **Info** (1 file) — trivial utility
8. **Interactive forms** — enhance all existing create/update commands
