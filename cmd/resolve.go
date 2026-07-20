package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/inventory"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/key_store"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/runner"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/schedule"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/user"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

// Per-resource name→ID listings backing resolveIDOrName and positional-arg
// completion. Each fetcher authenticates itself (auth errors returned
// unwrapped so exit-code mapping sees the sentinels) and wraps only the API
// call. Resources without names (tasks) or with string IDs (tokens) have no
// fetcher — their commands keep parseIDArg / raw args.

func projectNameIDs(_ *cobra.Command) ([]nameID, error) {
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	resp, err := apiClient.Project.GetProjects(project.NewGetProjectsParams(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	out := make([]nameID, 0, len(resp.GetPayload()))
	for _, p := range resp.GetPayload() {
		out = append(out, nameID{ID: p.ID, Name: p.Name})
	}
	return out, nil
}

func templateNameIDs(cmd *cobra.Command) ([]nameID, error) {
	pid, err := getProjectID(cmd)
	if err != nil {
		return nil, err
	}
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	params := template.NewGetProjectProjectIDTemplatesParams()
	params.ProjectID = int64(pid)
	resp, err := apiClient.Template.GetProjectProjectIDTemplates(params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	out := make([]nameID, 0, len(resp.GetPayload()))
	for _, t := range resp.GetPayload() {
		out = append(out, nameID{ID: t.ID, Name: t.Name})
	}
	return out, nil
}

func inventoryNameIDs(cmd *cobra.Command) ([]nameID, error) {
	pid, err := getProjectID(cmd)
	if err != nil {
		return nil, err
	}
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	params := inventory.NewGetProjectProjectIDInventoryParams()
	params.ProjectID = int64(pid)
	resp, err := apiClient.Inventory.GetProjectProjectIDInventory(params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventories: %w", err)
	}
	out := make([]nameID, 0, len(resp.GetPayload()))
	for _, i := range resp.GetPayload() {
		out = append(out, nameID{ID: i.ID, Name: i.Name})
	}
	return out, nil
}

func repoNameIDs(cmd *cobra.Command) ([]nameID, error) {
	pid, err := getProjectID(cmd)
	if err != nil {
		return nil, err
	}
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	params := repository.NewGetProjectProjectIDRepositoriesParams()
	params.ProjectID = int64(pid)
	resp, err := apiClient.Repository.GetProjectProjectIDRepositories(params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	out := make([]nameID, 0, len(resp.GetPayload()))
	for _, r := range resp.GetPayload() {
		out = append(out, nameID{ID: r.ID, Name: r.Name})
	}
	return out, nil
}

func envNameIDs(cmd *cobra.Command) ([]nameID, error) {
	pid, err := getProjectID(cmd)
	if err != nil {
		return nil, err
	}
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	params := variable_group.NewGetProjectProjectIDEnvironmentParams()
	params.ProjectID = int64(pid)
	resp, err := apiClient.VariableGroup.GetProjectProjectIDEnvironment(params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	out := make([]nameID, 0, len(resp.GetPayload()))
	for _, e := range resp.GetPayload() {
		out = append(out, nameID{ID: e.ID, Name: e.Name})
	}
	return out, nil
}

func keyNameIDs(cmd *cobra.Command) ([]nameID, error) {
	pid, err := getProjectID(cmd)
	if err != nil {
		return nil, err
	}
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	params := key_store.NewGetProjectProjectIDKeysParams()
	params.ProjectID = int64(pid)
	resp, err := apiClient.KeyStore.GetProjectProjectIDKeys(params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	out := make([]nameID, 0, len(resp.GetPayload()))
	for _, k := range resp.GetPayload() {
		out = append(out, nameID{ID: k.ID, Name: k.Name})
	}
	return out, nil
}

func scheduleNameIDs(cmd *cobra.Command) ([]nameID, error) {
	pid, err := getProjectID(cmd)
	if err != nil {
		return nil, err
	}
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	params := schedule.NewGetProjectProjectIDSchedulesParams()
	params.ProjectID = int64(pid)
	resp, err := apiClient.Schedule.GetProjectProjectIDSchedules(params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	out := make([]nameID, 0, len(resp.GetPayload()))
	for _, s := range resp.GetPayload() {
		out = append(out, nameID{ID: s.ID, Name: s.Name})
	}
	return out, nil
}

// runnerNameIDs follows runnerScope: global unless --project is explicit.
func runnerNameIDs(cmd *cobra.Command) ([]nameID, error) {
	pid, projectScoped, err := runnerScope(cmd)
	if err != nil {
		return nil, err
	}
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	var items []*models.Runner
	if projectScoped {
		params := runner.NewGetProjectProjectIDRunnersParams()
		params.ProjectID = pid
		resp, err := apiClient.Runner.GetProjectProjectIDRunners(params, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list runners: %w", err)
		}
		items = resp.GetPayload()
	} else {
		resp, err := apiClient.Runner.GetRunners(runner.NewGetRunnersParams(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list runners: %w", err)
		}
		items = resp.GetPayload()
	}
	out := make([]nameID, 0, len(items))
	for _, r := range items {
		out = append(out, nameID{ID: r.ID, Name: r.Name})
	}
	return out, nil
}

// userNameIDs resolves by login username, not display name.
func userNameIDs(_ *cobra.Command) ([]nameID, error) {
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, err
	}
	resp, err := apiClient.User.GetUsers(user.NewGetUsersParams(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	out := make([]nameID, 0, len(resp.GetPayload()))
	for _, u := range resp.GetPayload() {
		out = append(out, nameID{ID: u.ID, Name: u.Username})
	}
	return out, nil
}
