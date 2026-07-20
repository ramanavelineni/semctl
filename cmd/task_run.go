package cmd

import (
	"fmt"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/internal/style"
	apiclientpkg "github.com/ramanavelineni/semctl/pkg/semapi/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/task"
	"github.com/spf13/cobra"
)

// taskPollInterval is how often --wait/--follow polls the task status.
const taskPollInterval = 2 * time.Second

var taskRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a task",
	Long: `Start a task from a template.

By default the task is started and semctl returns immediately. Use --wait to
block until the task finishes (exit code reflects the task result), or
--follow to also stream the task output while waiting. Both are intended for
CI pipelines that need to gate on task success.`,
	Example: `  semctl task run --template-id 1
  semctl task run --template-id 1 --message "Deploy v1.2" --git-branch main
  semctl task run --template-id 1 --wait
  semctl task run --template-id 1 --follow --wait-timeout 30m`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		templateID, _ := cmd.Flags().GetInt64("template-id")
		message, _ := cmd.Flags().GetString("message")
		gitBranch, _ := cmd.Flags().GetString("git-branch")
		arguments, _ := cmd.Flags().GetString("arguments")
		environment, _ := cmd.Flags().GetString("environment")
		limit, _ := cmd.Flags().GetString("limit")
		playbook, _ := cmd.Flags().GetString("playbook")
		debug, _ := cmd.Flags().GetBool("debug")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		diff, _ := cmd.Flags().GetBool("diff")
		wait, _ := cmd.Flags().GetBool("wait")
		follow, _ := cmd.Flags().GetBool("follow")
		waitTimeout, _ := cmd.Flags().GetDuration("wait-timeout")

		if follow {
			wait = true
		}

		interactive, err := shouldAutoInteractive(cmd, templateID == 0)
		if err != nil {
			return err
		}
		if interactive {
			tplOpts, err := nameIDOptions(cmd, templateNameIDs, false)
			if err != nil {
				return err
			}
			if len(tplOpts) == 0 {
				return fmt.Errorf("no templates in this project — create one with 'semctl template create'")
			}
			var toggles []string
			if debug {
				toggles = append(toggles, "debug")
			}
			if dryRun {
				toggles = append(toggles, "dry-run")
			}
			if diff {
				toggles = append(toggles, "diff")
			}
			if err := runForm(newForm(
				huh.NewGroup(
					huh.NewSelect[int64]().Title("Template").Options(tplOpts...).Value(&templateID),
					huh.NewInput().Title("Message (optional)").Value(&message),
					huh.NewInput().Title("Git branch (optional)").Value(&gitBranch),
					huh.NewMultiSelect[string]().Title("Options").
						Options(
							huh.NewOption("debug", "debug"),
							huh.NewOption("dry-run", "dry-run"),
							huh.NewOption("diff", "diff"),
						).Value(&toggles),
				).Title("Run task").Description(moreFlagsNote),
			)); err != nil {
				return err
			}
			debug, dryRun, diff = false, false, false
			for _, tgl := range toggles {
				switch tgl {
				case "debug":
					debug = true
				case "dry-run":
					dryRun = true
				case "diff":
					diff = true
				}
			}
		}

		if templateID == 0 {
			return fmt.Errorf("--template-id is required")
		}

		// Unbounded waits hang CI jobs until the runner's own timeout kills
		// them; keep working but say so.
		if wait && waitTimeout == 0 && !style.IsStdinTTY() {
			style.Warning("--wait without --wait-timeout blocks indefinitely if the task hangs; set --wait-timeout in CI.")
		}

		body := task.PostProjectProjectIDTasksBody{
			TemplateID:  templateID,
			Message:     message,
			GitBranch:   gitBranch,
			Arguments:   arguments,
			Environment: environment,
			Limit:       limit,
			Playbook:    playbook,
			Debug:       debug,
			DryRun:      dryRun,
			Diff:        diff,
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := task.NewPostProjectProjectIDTasksParams()
		params.ProjectID = int64(pid)
		params.Task = body

		resp, err := apiClient.Task.PostProjectProjectIDTasks(params, nil)
		if err != nil {
			return fmt.Errorf("failed to run task: %w", err)
		}

		t := resp.GetPayload()
		style.Success(fmt.Sprintf("Started task %d (template: %d)", t.ID, t.TemplateID))

		// Machine-readable task on stdout so pipelines can capture the ID.
		// Must not return early: --json combines with --wait/--follow.
		if output.GetFormat() != output.FormatTable {
			if err := output.Print(t, nil, nil); err != nil {
				return err
			}
		}

		if !wait {
			return nil
		}

		return waitForTask(apiClient, int64(pid), t.ID, follow, waitTimeout)
	},
}

// waitForTask polls a task until it reaches a terminal status. With follow,
// new output lines are streamed to stdout as they appear. A non-success
// terminal status is returned as an error so the process exits non-zero.
func waitForTask(apiClient *apiclientpkg.Semapi, projectID, taskID int64, follow bool, waitTimeout time.Duration) error {
	deadline := time.Time{}
	if waitTimeout > 0 {
		deadline = time.Now().Add(waitTimeout)
	}

	printed := 0
	consecutiveErrors := 0

	for {
		status, err := fetchTaskStatus(apiClient, projectID, taskID)
		if err != nil {
			// Tolerate transient poll failures (network blips, restarts).
			consecutiveErrors++
			if consecutiveErrors >= 3 {
				return fmt.Errorf("polling task %d failed repeatedly: %w", taskID, err)
			}
		} else {
			consecutiveErrors = 0

			if follow {
				printed = printNewTaskOutput(apiClient, projectID, taskID, printed)
			}

			switch status {
			case "success":
				style.Success(fmt.Sprintf("Task %d finished: %s", taskID, status))
				return nil
			case "error", "stopped":
				return withExitCode(fmt.Errorf("task %d finished with status %q", taskID, status), exitTaskFailed)
			}
		}

		if !deadline.IsZero() && time.Now().After(deadline) {
			return withExitCode(fmt.Errorf("timed out after %s waiting for task %d (task is still running server-side)", waitTimeout, taskID), exitWaitTimeout)
		}

		time.Sleep(taskPollInterval)
	}
}

func fetchTaskStatus(apiClient *apiclientpkg.Semapi, projectID, taskID int64) (string, error) {
	params := task.NewGetProjectProjectIDTasksTaskIDParams()
	params.ProjectID = projectID
	params.TaskID = taskID

	resp, err := apiClient.Task.GetProjectProjectIDTasksTaskID(params, nil)
	if err != nil {
		return "", err
	}
	return resp.GetPayload().Status, nil
}

// printNewTaskOutput fetches the task output and prints any lines beyond
// alreadyPrinted, returning the new count. Fetch errors are ignored — the
// next poll retries.
func printNewTaskOutput(apiClient *apiclientpkg.Semapi, projectID, taskID int64, alreadyPrinted int) int {
	params := task.NewGetProjectProjectIDTasksTaskIDOutputParams()
	params.ProjectID = projectID
	params.TaskID = taskID

	resp, err := apiClient.Task.GetProjectProjectIDTasksTaskIDOutput(params, nil)
	if err != nil {
		return alreadyPrinted
	}

	lines := resp.GetPayload()
	for i := alreadyPrinted; i < len(lines); i++ {
		fmt.Println(lines[i].Output)
	}
	return len(lines)
}

func init() {
	taskCmd.AddCommand(taskRunCmd)

	taskRunCmd.Flags().Int64("template-id", 0, "template ID (required)")
	taskRunCmd.Flags().String("message", "", "task message")
	taskRunCmd.Flags().String("git-branch", "", "git branch override")
	taskRunCmd.Flags().String("arguments", "", "extra arguments (JSON)")
	taskRunCmd.Flags().String("environment", "", "environment override (JSON)")
	taskRunCmd.Flags().String("limit", "", "limit hosts")
	taskRunCmd.Flags().String("playbook", "", "playbook override")
	taskRunCmd.Flags().Bool("debug", false, "enable debug mode")
	taskRunCmd.Flags().Bool("dry-run", false, "dry run mode")
	taskRunCmd.Flags().Bool("diff", false, "show diff")
	taskRunCmd.Flags().Bool("wait", false, "wait for the task to finish; exit non-zero if it fails")
	taskRunCmd.Flags().Bool("follow", false, "stream task output while waiting (implies --wait)")
	taskRunCmd.Flags().Duration("wait-timeout", 0, "maximum time to wait with --wait/--follow (0 = no limit)")
}
