# Apply Configuration Schema Reference

This document describes every field available in `semctl apply` configuration files.

Configuration files can be written in YAML (`.yaml`, `.yml`) or JSON (`.json`). All examples below use YAML.

## Environment Variable Expansion

`${VAR_NAME}` references in string values (and in variable-map keys) are expanded from the environment after the file is parsed.

```yaml
keys:
  - name: my-key
    type: ssh
    ssh:
      private_key: "${SSH_PRIVATE_KEY}"
```

Rules:

- Only the braced `${VAR_NAME}` form is expanded. Bare `$WORD` text (common in Ansible arguments, cron entries, and passwords) is left untouched.
- Referencing a variable that is **not set** is an error in `semctl apply` — values are never silently replaced with empty strings. `semctl validate` treats unset variables as empty but prints a warning, so config files can be validated offline without secrets present.
- To produce a literal `${VAR_NAME}` in a value, escape it as `$${VAR_NAME}`.
- Unknown keys are a **parse error** (strict field checking) — a typo like `enviroment_variables:` fails loudly instead of being dropped silently.
- Updates **merge over** existing state: an empty/omitted field keeps the server-side value. Consequently apply cannot *unset* a field (e.g. clear a template description or detach an ssh key) — do that in the UI or with `semctl <resource> update`.
- Expansion happens **after parsing** and only inside string values, so an environment value can never change the document structure (a value containing YAML syntax stays an inert string). Consequently, `${VAR}` cannot be used for numeric fields like `ssh_key_id` — use the name-reference fields (`ssh_key: "${KEY_NAME}"`) instead.

---

## Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project` | `string` | Yes | Name of the Semaphore project. If it does not exist, it will be created. |
| `project_state` | `string` | No | Set to `"absent"` to delete the project and all its resources. When omitted or empty, the project is created or updated. |
| `keys` | [`[]KeyEntry`](#keyentry) | No | List of access keys (SSH keys, login credentials). |
| `variable_groups` | [`[]VariableGroupEntry`](#variablegroupentry) | No | List of variable groups (maps to "Environments" in the Semaphore API). |
| `repositories` | [`[]RepoEntry`](#repoentry) | No | List of Git repositories. |
| `inventories` | [`[]InventoryEntry`](#inventoryentry) | No | List of inventories (static, file-based, or Terraform). |
| `templates` | [`[]TemplateEntry`](#templateentry) | No | List of task templates. |
| `schedules` | [`[]ScheduleEntry`](#scheduleentry) | No | List of cron schedules for templates, reconciled by name. |

### Minimal example

```yaml
project: my-project
```

### Full example structure

```yaml
project: my-project
keys: [...]
variable_groups: [...]
repositories: [...]
inventories: [...]
templates: [...]
schedules: [...]
```

### Project deletion example

```yaml
project: my-project
project_state: absent
```

When `project_state: absent` is set, semctl discovers all existing child resources (templates, inventories, repositories, variable groups, keys) in the project and deletes them in dependency order before deleting the project itself.

---

## Resource State

Every resource entry supports an optional `state` field. When set to `"absent"`, the resource is deleted if it exists and skipped if it does not. When omitted or empty, the resource is created or updated.

```yaml
keys:
  - name: old-key
    state: absent
```

---

## KeyEntry

Represents an access key in Semaphore. Keys are used by repositories, inventories, and other resources for authentication.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | Yes | Unique name of the key within the project. |
| `type` | `string` | Yes (unless `state: absent`) | Key type. Valid values: `none`, `ssh`, `login_password`. |
| `state` | `string` | No | Set to `"absent"` to delete. |
| `ssh` | [`SSHKeyData`](#sshkeydata) | No | SSH key credentials. Required when `type: ssh`. |
| `login_password` | [`LoginPasswordData`](#loginpassworddata) | No | Login/password credentials. Required when `type: login_password`. |

### Example

```yaml
keys:
  - name: git-ssh-key
    type: ssh
    ssh:
      private_key: "${GIT_SSH_PRIVATE_KEY}"

  - name: vault-credentials
    type: login_password
    login_password:
      login: admin
      password: "${VAULT_PASSWORD}"

  - name: empty-key
    type: none
```

---

## SSHKeyData

SSH key credentials, used when a key has `type: ssh`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `login` | `string` | No | SSH login username. |
| `private_key` | `string` | No | PEM-encoded private key content. Providing this field triggers an update on every apply since the API never returns secret values. |
| `passphrase` | `string` | No | Passphrase for the private key, if encrypted. |

### Example

```yaml
ssh:
  login: ramana
  private_key: "${HOST_SSH_PRIVATE_KEY}"
```

---

## LoginPasswordData

Login/password credentials, used when a key has `type: login_password`.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `login` | `string` | No | Login username. |
| `password` | `string` | No | Password. Providing this field triggers an update on every apply since the API never returns secret values. |

### Example

```yaml
login_password:
  login: admin
  password: "${ADMIN_PASSWORD}"
```

---

## VariableGroupEntry

Represents a variable group in Semaphore (called "Environment" in the API). Variable groups hold extra variables, environment variables, and secrets that are passed to task templates at runtime.

In the Semaphore UI, a variable group has two tabs:

- **Variables tab** -- contains "Extra variables" and "Environment variables"
- **Secrets tab** -- contains secret "Extra variables" and secret "Environment variables"

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `group_name` | `string` | Yes | Unique name of the variable group within the project. |
| `state` | `string` | No | Set to `"absent"` to delete. |
| `variables` | `map[string]string` | No | Extra variables (key-value pairs). Displayed under the **Variables** tab in the UI. Passed as extra vars to Ansible (e.g., `--extra-vars`). |
| `environment_variables` | `map[string]string` | No | Environment variables (key-value pairs). Displayed under the **Variables** tab in the UI. Set as OS-level environment variables during task execution. |
| `secrets` | `map[string]string` | No | Secret extra variables (key-value pairs). Displayed under the **Secrets** tab in the UI as "Extra variables". Values are encrypted at rest and never returned by the API. Always re-applied on every update. |
| `secret_environment_variables` | `map[string]string` | No | Secret environment variables (key-value pairs). Displayed under the **Secrets** tab in the UI as "Environment variables". Values are encrypted at rest and never returned by the API. Always re-applied on every update. |

### How fields map to the Semaphore UI

```
Variable Group
+-- Variables tab
|   +-- Extra variables          <-- "variables"
|   +-- Environment variables    <-- "environment_variables"
+-- Secrets tab
    +-- Extra variables          <-- "secrets"
    +-- Environment variables    <-- "secret_environment_variables"
```

### Example

```yaml
variable_groups:
  - group_name: production
    variables:
      ansible_user: deploy
      cluster_name: prod
    environment_variables:
      ANSIBLE_HOST_KEY_CHECKING: "False"
      ANSIBLE_TIMEOUT: "30"
    secrets:
      ansible_vault_password: "${ANSIBLE_VAULT_PASSWORD}"
    secret_environment_variables:
      AWS_SECRET_ACCESS_KEY: "${AWS_SECRET_ACCESS_KEY}"
```

---

## RepoEntry

Represents a Git repository that templates use as their source of playbooks, roles, and other files.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | Yes | Unique name of the repository within the project. |
| `state` | `string` | No | Set to `"absent"` to delete. |
| `git_url` | `string` | Yes (unless `state: absent`) | Git clone URL. Supports SSH (`git@...`) and HTTPS (`https://...`) formats. |
| `git_branch` | `string` | No | Default branch to checkout. |
| `ssh_key` | `string` | No | Name of a key defined in the `keys` section to use for Git authentication. Resolved by case-insensitive name match. |
| `ssh_key_id` | `integer` | No | Explicit Semaphore key ID. Takes precedence over `ssh_key` if both are set. |

### Cross-referencing keys

Use `ssh_key` to reference a key by its name. The name is matched case-insensitively against keys defined in the same file or already existing in the project.

```yaml
keys:
  - name: git-ssh-key
    type: ssh
    ssh:
      private_key: "${GIT_SSH_PRIVATE_KEY}"

repositories:
  - name: ansible
    git_url: "git@github.com:org/ansible.git"
    git_branch: main
    ssh_key: git-ssh-key     # references the key above
```

Alternatively, use `ssh_key_id` to reference a key by its numeric Semaphore ID:

```yaml
repositories:
  - name: ansible
    git_url: "git@github.com:org/ansible.git"
    ssh_key_id: 42
```

---

## InventoryEntry

Represents an Ansible inventory. Inventories can be defined inline (static), reference a file in a repository, or point to a Terraform workspace.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | Yes | Unique name of the inventory within the project. |
| `state` | `string` | No | Set to `"absent"` to delete. |
| `type` | `string` | Yes (unless `state: absent`) | Inventory type. Valid values: `static`, `static-yaml`, `file`, `terraform-workspace`. |
| `inventory` | `string` | No | Inventory content or path. For `static`/`static-yaml`: inline inventory content. For `file`: path to the inventory file within the repository. |
| `ssh_key` | `string` | No | Name of a key for SSH connections to inventory hosts. |
| `ssh_key_id` | `integer` | No | Explicit Semaphore key ID for SSH connections. Takes precedence over `ssh_key`. |
| `become_key` | `string` | No | Name of a key for privilege escalation (sudo). |
| `become_key_id` | `integer` | No | Explicit Semaphore key ID for privilege escalation. Takes precedence over `become_key`. |
| `repository` | `string` | No | Name of a repository (for `file` type inventories). |
| `repository_id` | `integer` | No | Explicit Semaphore repository ID. Takes precedence over `repository`. |

### Inventory types

- **`static`** -- INI-format inventory defined inline in the `inventory` field.
- **`static-yaml`** -- YAML-format inventory defined inline in the `inventory` field.
- **`file`** -- Path to an inventory file inside a linked repository. Requires `repository` or `repository_id`.
- **`terraform-workspace`** -- Terraform workspace-based dynamic inventory.

### Example: file-based inventory

```yaml
inventories:
  - name: homelab-file
    type: file
    inventory: inventories/homelab/hosts
    ssh_key: host-ssh-key
    become_key: host-ssh-key
    repository: ansible
```

### Example: static inline inventory

```yaml
inventories:
  - name: homelab-static
    type: static
    inventory: |
      [webservers]
      web01
      web02

      [databases]
      db01
    ssh_key: host-ssh-key
    become_key: host-ssh-key
```

---

## TemplateEntry

Represents a task template in Semaphore. Templates define what playbook to run, which repository, inventory, and variable group to use, and various execution options.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | Yes | Unique name of the template within the project. |
| `state` | `string` | No | Set to `"absent"` to delete. |
| `type` | `string` | No | Template type (e.g., `""`, `"build"`, `"deploy"`). |
| `app` | `string` | No | Application type (e.g., `"ansible"`, `"terraform"`, `"tofu"`, `"bash"`, `"powershell"`). |
| `playbook` | `string` | No | Playbook filename or script path to execute. |
| `description` | `string` | No | Human-readable description of the template. |
| `git_branch` | `string` | No | Override the repository's default branch for this template. |
| `arguments` | `string` | No | Additional CLI arguments passed to the task runner (JSON array string, e.g., `'["--tags", "setup"]'`). |
| `start_version` | `string` | No | Starting version string for deploy-type templates. |
| `autorun` | `boolean` | No | If `true`, automatically run this template when its repository is updated. Omitted: keeps the existing value on update, `false` on create. |
| `suppress_success_alerts` | `boolean` | No | If `true`, suppress notifications on successful task completion. Default: `false`. |
| `allow_override_args_in_task` | `boolean` | No | If `true`, allow users to override arguments when manually running a task. Default: `false`. |
| `repository` | `string` | No | Name of a repository defined in the `repositories` section. |
| `repository_id` | `integer` | No | Explicit Semaphore repository ID. Takes precedence over `repository`. |
| `variable_group` | `string` | No | Name of a variable group defined in the `variable_groups` section. |
| `environment` | `string` | No | Alias for `variable_group`. Cannot be used together with `variable_group`. |
| `environment_id` | `integer` | No | Explicit Semaphore environment ID. Takes precedence over `variable_group`/`environment`. |
| `inventory` | `string` | No | Name of an inventory defined in the `inventories` section. |
| `inventory_id` | `integer` | No | Explicit Semaphore inventory ID. Takes precedence over `inventory`. |
| `build_template` | `string` | No | Name of another template to run as a build step before this template. |
| `build_template_id` | `integer` | No | Explicit Semaphore template ID for the build step. Takes precedence over `build_template`. |
| `view_id` | `integer` | No | ID of the view (tab) to display this template under in the Semaphore UI. |

### Cross-references

Templates reference other resources by name. All name lookups are case-insensitive.

```yaml
templates:
  - name: deploy-app
    app: ansible
    playbook: deploy.yaml
    description: "Deploy application to production"
    repository: ansible           # matches repositories[].name
    variable_group: production    # matches variable_groups[].group_name
    inventory: prod-hosts         # matches inventories[].name
    build_template: build-app     # matches templates[].name
    autorun: true
    allow_override_args_in_task: true
```

### ID-based references

For resources not managed in the same config file, you can reference them by their Semaphore ID:

```yaml
templates:
  - name: deploy-app
    app: ansible
    playbook: deploy.yaml
    repository_id: 10
    environment_id: 20
    inventory_id: 30
```

---

## ScheduleEntry

Represents a cron schedule attached to a template. Schedules are reconciled by name like every other resource: matched schedules are updated in place, missing ones are created, and `state: absent` deletes them.

> **Note:** Semaphore does not enforce unique schedule names, and semctl versions before v0.4.0 could not reconcile schedules (each apply created a new copy). When several schedules share a name, semctl manages the first match and the plan output flags the duplicates. To clean up duplicates: set `state: absent` (deletes **all** schedules with that name), apply, then restore the entry and apply again.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | Yes | Name of the schedule. Matching against existing schedules is case-insensitive. |
| `state` | `string` | No | Set to `"absent"` to delete all schedules with this name. |
| `cron_format` | `string` | Yes* | Cron expression (e.g., `"0 2 * * *"` for daily at 2 AM). *Not required when `state: absent`. |
| `template` | `string` | Conditional | Name of the template to schedule. Either `template` or `template_id` is required (unless `state: absent`). |
| `template_id` | `integer` | Conditional | Explicit Semaphore template ID. Takes precedence over `template`. |
| `active` | `boolean` | No | Whether the schedule is active. Omitted: keeps the existing value on update, `true` on create. |

### Example

```yaml
schedules:
  - name: nightly-backup
    cron_format: "0 2 * * *"
    template: backup
    active: true

  - name: weekly-audit
    cron_format: "0 6 * * 0"
    template: security-audit
```

---

## Resource Processing Order

Resources are processed in a specific dependency order:

### Create/Update order

1. Project
2. Keys
3. Variable Groups
4. Repositories
5. Inventories
6. Templates
7. Schedules

### Delete order (reverse)

1. Schedules
2. Templates
3. Inventories
4. Repositories
5. Variable Groups
6. Keys
7. Project

This ensures that dependent resources are created before the resources that reference them, and deleted after.

---

## Reconciliation Behavior

| Scenario | Behavior |
|----------|----------|
| Resource exists with same config | Skipped (no API call) |
| Resource exists with different config | Updated in place |
| Resource does not exist | Created |
| Resource has `state: absent` and exists | Deleted |
| Resource has `state: absent` and does not exist | Skipped |
| Secret fields provided (private keys, passwords, secrets) | Always updated (API never returns secret values for comparison) |

### Update merge semantics

Updates are **merges**, not replacements. A field omitted from the config keeps its current server-side value:

- String fields: an empty/omitted value means "keep existing".
- Reference fields (`ssh_key`, `repository`, ...): omitted means "keep existing".
- Boolean template fields (`autorun`, `suppress_success_alerts`, `allow_override_args_in_task`): omitted means "keep existing"; an explicit `false` actively sets the value to false.
- Template **survey variables and vaults** are not managed by apply configs and are preserved as configured server-side.

Consequence: apply cannot *clear* a field back to empty — set a new value instead, or change it in the UI.

### Export placeholders

`semctl export` writes `<set-me>` in place of secret values (the API never returns them). `semctl apply` and `semctl validate` **refuse** files that still contain `<set-me>`, so an unedited export can never overwrite real keys or secrets. Replace each placeholder with a real value or an `${ENV_VAR}` reference before applying.

---

## Name Matching

All resource name lookups (both within the config file and against existing Semaphore resources) are **case-insensitive**. For example, `ssh_key: Git-SSH-Key` will match a key named `git-ssh-key`.

Duplicate names within the same resource type (case-insensitive) are rejected during validation.

---

## Multiple Config Files

You can split your configuration across multiple files and apply them together:

```bash
# Apply a single file
semctl apply -f project.yaml

# Apply multiple files
semctl apply -f keys.yaml -f templates.yaml

# Apply all .yaml/.yml/.json files in a directory
semctl apply -f ./config/
```

Each file must have the same `project` value. Resources from all files are merged before reconciliation.

---

## Full Example

```yaml
project: homelab

keys:
  - name: git-ssh-key
    type: ssh
    ssh:
      private_key: "${GIT_SSH_PRIVATE_KEY}"

  - name: host-ssh-key
    type: ssh
    ssh:
      login: ramana
      private_key: "${HOST_SSH_PRIVATE_KEY}"

variable_groups:
  - group_name: homelab
    variables:
      ansible_user: ramana
      cluster_name: homelab
    environment_variables:
      ANSIBLE_HOST_KEY_CHECKING: "False"
    secrets:
      ansible_vault_password: "${ANSIBLE_VAULT_PASSWORD}"
    secret_environment_variables:
      AWS_SECRET_ACCESS_KEY: "${AWS_SECRET_ACCESS_KEY}"

repositories:
  - name: ansible
    git_url: "git@github.com:org/ansible.git"
    git_branch: main
    ssh_key: git-ssh-key

inventories:
  - name: homelab-file
    type: file
    inventory: inventories/homelab/hosts
    ssh_key: host-ssh-key
    become_key: host-ssh-key
    repository: ansible

templates:
  - name: k8s-setup
    app: ansible
    playbook: k8s-setup.yaml
    description: "Full Kubernetes cluster bootstrap"
    repository: ansible
    variable_group: homelab
    inventory: homelab-file

schedules:
  - name: nightly-deploy
    cron_format: "0 2 * * *"
    template: k8s-setup
```
