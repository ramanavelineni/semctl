package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/output"
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
	items, err := projectNameIDs(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, p := range items {
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
	client.SetTimeout(completionAPITimeout)
	items, err := templateNameIDs(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	for _, t := range items {
		out = append(out, fmt.Sprintf("%d\t%s", t.ID, t.Name))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeResourceNames returns a ValidArgsFunction completing the first
// positional argument with resource names (IDs shown as descriptions),
// backed by the same fetchers resolveIDOrName uses.
func completeResourceNames(list func(*cobra.Command) ([]nameID, error)) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cfgFile, _ := cmd.Flags().GetString("config")
		if err := config.Load(cfgFile); err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		client.SetTimeout(completionAPITimeout)
		items, err := list(cmd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		out := make([]string, 0, len(items))
		for _, it := range items {
			out = append(out, fmt.Sprintf("%s\tID %d", it.Name, it.ID))
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

// boolUpdateFields are field=value fields that take true/false.
var boolUpdateFields = map[string]bool{
	"alert":                   true,
	"autorun":                 true,
	"suppress_success_alerts": true,
	"admin":                   true,
	"active":                  true,
}

// completeUpdateFields returns a ValidArgsFunction for `<id|name> field=value...`
// update commands. idRequired marks commands whose first positional is the
// resource (all except project update, where it is optional); names backs
// the first-arg name completion.
func completeUpdateFields(names func(*cobra.Command) ([]nameID, error), idRequired bool, fields ...string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if idRequired && len(args) == 0 {
			return completeResourceNames(names)(cmd, args, toComplete)
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
		if !idRequired && len(args) == 0 {
			// Optional leading resource arg: names are as valid as fields here.
			nameComps, _ := completeResourceNames(names)(cmd, args, toComplete)
			out = append(out, nameComps...)
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
	_ = rootCmd.RegisterFlagCompletionFunc("output", cobra.FixedCompletions(
		[]string{string(output.FormatTable), string(output.FormatJSON), string(output.FormatYAML)},
		cobra.ShellCompDirectiveNoFileComp))

	// Positional <id|name> completion for show/delete-style commands.
	projectShowCmd.ValidArgsFunction = completeResourceNames(projectNameIDs)
	projectDeleteCmd.ValidArgsFunction = completeResourceNames(projectNameIDs)
	templateShowCmd.ValidArgsFunction = completeResourceNames(templateNameIDs)
	templateDeleteCmd.ValidArgsFunction = completeResourceNames(templateNameIDs)
	inventoryShowCmd.ValidArgsFunction = completeResourceNames(inventoryNameIDs)
	inventoryDeleteCmd.ValidArgsFunction = completeResourceNames(inventoryNameIDs)
	repoShowCmd.ValidArgsFunction = completeResourceNames(repoNameIDs)
	repoDeleteCmd.ValidArgsFunction = completeResourceNames(repoNameIDs)
	envShowCmd.ValidArgsFunction = completeResourceNames(envNameIDs)
	envDeleteCmd.ValidArgsFunction = completeResourceNames(envNameIDs)
	keyDeleteCmd.ValidArgsFunction = completeResourceNames(keyNameIDs)
	scheduleShowCmd.ValidArgsFunction = completeResourceNames(scheduleNameIDs)
	scheduleDeleteCmd.ValidArgsFunction = completeResourceNames(scheduleNameIDs)
	runnerShowCmd.ValidArgsFunction = completeResourceNames(runnerNameIDs)
	runnerDeleteCmd.ValidArgsFunction = completeResourceNames(runnerNameIDs)
	runnerActivateCmd.ValidArgsFunction = completeResourceNames(runnerNameIDs)
	runnerDeactivateCmd.ValidArgsFunction = completeResourceNames(runnerNameIDs)
	runnerClearCacheCmd.ValidArgsFunction = completeResourceNames(runnerNameIDs)
	runnerTokenCmd.ValidArgsFunction = completeResourceNames(runnerNameIDs)
	userShowCmd.ValidArgsFunction = completeResourceNames(userNameIDs)
	userDeleteCmd.ValidArgsFunction = completeResourceNames(userNameIDs)
	userPasswordCmd.ValidArgsFunction = completeResourceNames(userNameIDs)

	// Context subcommand args (rename completes only the old name).
	contextUseCmd.ValidArgsFunction = completeContextNames
	contextDeleteCmd.ValidArgsFunction = completeContextNames
	contextRenameCmd.ValidArgsFunction = completeContextNames

	// task run --template-id
	_ = taskRunCmd.RegisterFlagCompletionFunc("template-id", completeTemplateIDs)

	// field=value completion for update commands.
	projectUpdateCmd.ValidArgsFunction = completeUpdateFields(projectNameIDs, false,
		"name", "type", "alert", "alert_chat", "max_parallel_tasks")
	templateUpdateCmd.ValidArgsFunction = completeUpdateFields(templateNameIDs, true,
		"name", "description", "type", "app", "playbook", "git_branch", "arguments",
		"repository_id", "environment_id", "inventory_id", "build_template_id",
		"view_id", "autorun", "suppress_success_alerts")
	envUpdateCmd.ValidArgsFunction = completeUpdateFields(envNameIDs, true,
		"name", "json", "env", "password")
	keyUpdateCmd.ValidArgsFunction = completeUpdateFields(keyNameIDs, true,
		"name", "type", "login", "password", "private_key", "passphrase")
	inventoryUpdateCmd.ValidArgsFunction = completeUpdateFields(inventoryNameIDs, true,
		"name", "type", "inventory", "ssh_key_id", "become_key_id", "repository_id")
	repoUpdateCmd.ValidArgsFunction = completeUpdateFields(repoNameIDs, true,
		"name", "git_url", "git_branch", "ssh_key_id")
	userUpdateCmd.ValidArgsFunction = completeUpdateFields(userNameIDs, true,
		"username", "name", "email", "admin", "alert")
	runnerUpdateCmd.ValidArgsFunction = completeUpdateFields(runnerNameIDs, true,
		"name", "active", "max_parallel_tasks", "tags", "webhook")
	scheduleUpdateCmd.ValidArgsFunction = completeUpdateFields(scheduleNameIDs, true,
		"name", "template_id", "cron_format", "active", "type", "run_at")
}
