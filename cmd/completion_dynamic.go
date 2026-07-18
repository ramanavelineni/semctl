package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/template"
	"github.com/spf13/cobra"
)

// Completion runs in its own short-lived process on every TAB press: config
// is NOT loaded by PersistentPreRunE (__complete is in the skip list), so
// every function here loads it best-effort itself, fails silently, and — for
// anything touching the API — first drops the HTTP timeout so a dead server
// can never hang the user's shell.

const completionAPITimeout = 2 * time.Second

// completeContextNames completes context names from the local config file.
func completeContextNames(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfgFile, _ := cmd.Flags().GetString("config")
	if err := config.Load(cfgFile); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return config.ListContexts(), cobra.ShellCompDirectiveNoFileComp
}

// completeProjectNames completes project names for the -p/--project flag.
func completeProjectNames(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cfgFile, _ := cmd.Flags().GetString("config")
	if err := config.Load(cfgFile); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client.SetTimeout(completionAPITimeout)
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	resp, err := apiClient.Project.GetProjects(project.NewGetProjectsParams(), nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, p := range resp.GetPayload() {
		names = append(names, p.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeTemplateIDs completes template IDs (with names as descriptions)
// for task run --template-id.
func completeTemplateIDs(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cfgFile, _ := cmd.Flags().GetString("config")
	if err := config.Load(cfgFile); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	pid, err := getProjectID(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client.SetTimeout(completionAPITimeout)
	apiClient, err := client.NewAuthenticatedClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	params := template.NewGetProjectProjectIDTemplatesParams()
	params.ProjectID = int64(pid)
	resp, err := apiClient.Template.GetProjectProjectIDTemplates(params, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	for _, t := range resp.GetPayload() {
		out = append(out, fmt.Sprintf("%d\t%s", t.ID, t.Name))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// boolUpdateFields are field=value fields that take true/false.
var boolUpdateFields = map[string]bool{
	"alert":                   true,
	"autorun":                 true,
	"suppress_success_alerts": true,
	"admin":                   true,
	"active":                  true,
}

// completeUpdateFields returns a ValidArgsFunction for `[<id>] field=value...`
// update commands. hasIDArg marks commands whose first positional is the
// resource ID (all except project update).
func completeUpdateFields(hasIDArg bool, fields ...string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if hasIDArg && len(args) == 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		// Value side: offer true/false for boolean fields, nothing otherwise.
		if name, _, ok := strings.Cut(toComplete, "="); ok {
			if boolUpdateFields[name] {
				return []string{name + "=true", name + "=false"}, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		out := make([]string, 0, len(fields))
		for _, f := range fields {
			out = append(out, f+"=")
		}
		return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}
}

// registerDynamicCompletions wires flag and argument completion. Called from
// Execute, NOT init: this file's init would run before root.go's and
// task_run.go's (alphabetical order), when the flags don't exist yet and
// RegisterFlagCompletionFunc silently fails.
func registerDynamicCompletions() {
	// Persistent flags on root.
	_ = rootCmd.RegisterFlagCompletionFunc("context", func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completeContextNames(cmd, nil, "")
	})
	_ = rootCmd.RegisterFlagCompletionFunc("project", completeProjectNames)

	// Context subcommand args (rename completes only the old name).
	contextUseCmd.ValidArgsFunction = completeContextNames
	contextDeleteCmd.ValidArgsFunction = completeContextNames
	contextRenameCmd.ValidArgsFunction = completeContextNames

	// task run --template-id
	_ = taskRunCmd.RegisterFlagCompletionFunc("template-id", completeTemplateIDs)

	// field=value completion for update commands.
	projectUpdateCmd.ValidArgsFunction = completeUpdateFields(false,
		"name", "type", "alert", "alert_chat", "max_parallel_tasks")
	templateUpdateCmd.ValidArgsFunction = completeUpdateFields(true,
		"name", "description", "type", "app", "playbook", "git_branch", "arguments",
		"repository_id", "environment_id", "inventory_id", "build_template_id",
		"view_id", "autorun", "suppress_success_alerts")
	envUpdateCmd.ValidArgsFunction = completeUpdateFields(true,
		"name", "json", "env", "password")
	keyUpdateCmd.ValidArgsFunction = completeUpdateFields(true,
		"name", "type", "login", "password", "private_key", "passphrase")
	inventoryUpdateCmd.ValidArgsFunction = completeUpdateFields(true,
		"name", "type", "inventory", "ssh_key_id", "become_key_id", "repository_id")
	repoUpdateCmd.ValidArgsFunction = completeUpdateFields(true,
		"name", "git_url", "git_branch", "ssh_key_id")
	userUpdateCmd.ValidArgsFunction = completeUpdateFields(true,
		"username", "name", "email", "admin", "alert")
	runnerUpdateCmd.ValidArgsFunction = completeUpdateFields(true,
		"name", "active", "max_parallel_tasks", "tags", "webhook")
	scheduleUpdateCmd.ValidArgsFunction = completeUpdateFields(true,
		"name", "template_id", "cron_format", "active", "type", "run_at")
}
