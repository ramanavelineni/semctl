# semctl — CLI for Semaphore UI

## Project Overview
Go CLI tool for managing Semaphore UI via its REST API. Built with Cobra + Viper.

## Key Patterns
- `pkg/semapi/` is a generated OpenAPI client — DO NOT HAND-EDIT
- User messages → stderr via `style.Success()`, `style.Error()`, etc.
- Data output → stdout via `output.Print()`, `output.PrintTable()`
- Config writing uses `yaml.v3` directly (not Viper) for partial file updates
- Auth: username/password → cookie login → create API token → cache token
- Project-scoped commands resolve project ID via: `--project` flag → `defaults.project_id` config → error
- Commands skipped in PersistentPreRunE: login, logout, completion, version, __complete
- Interactive mode: `shouldAutoInteractive(cmd, inputsMissing)` pattern
- Update commands use `field=value` positional args pattern: fetch current resource, apply overrides, PUT back
- Delete commands use `--yes` flag with stdin `[y/N]` confirmation prompt
- All API calls pass `nil` for authInfo, relying on `transport.DefaultAuthentication`

## API Client Gotchas
- `getProjectID(cmd)` returns `int32` but API params expect `int64` — always cast with `int64(pid)`
- Environment APIs use `apiClient.VariableGroup` (not a separate Environment client)
- Token APIs use `apiClient.Authentication` (GetUserTokens, PostUserTokens, DeleteUserTokensAPITokenID)
- Event APIs use `apiClient.Operations` (GetEvents, GetEventsLast)
- Key resource has no GET-by-ID endpoint — fetch from list and filter by ID
- APIToken.ID is a `string` (not int64 like other resource IDs)
- Schedule API has no list endpoint — only get/create/update/delete by ID

## Commit Style (Mandatory)
- Use [Conventional Commits](https://www.conventionalcommits.org): `type(scope): description`
- Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `build`
- Include key changes as bullet points in the commit body
- Example: `feat(cmd): add CRUD commands for all project resources`

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
