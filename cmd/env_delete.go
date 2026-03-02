package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/variable_group"
	"github.com/spf13/cobra"
)

var envDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete an environment",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl env delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid environment ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		autoConfirm, _ := cmd.Flags().GetBool("yes")
		if !autoConfirm {
			fmt.Fprintf(os.Stderr, "Delete environment %d? [y/N] ", id)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				style.Info("Cancelled.")
				return nil
			}
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := variable_group.NewDeleteProjectProjectIDEnvironmentEnvironmentIDParams()
		params.ProjectID = int64(pid)
		params.EnvironmentID = id

		_, err = apiClient.VariableGroup.DeleteProjectProjectIDEnvironmentEnvironmentID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to delete environment: %w", err)
		}

		style.Success(fmt.Sprintf("Deleted environment %d", id))
		return nil
	},
}

func init() {
	envCmd.AddCommand(envDeleteCmd)
}
