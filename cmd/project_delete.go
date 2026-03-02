package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/project"
	"github.com/spf13/cobra"
)

var projectDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a project",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl project delete 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		autoConfirm, _ := cmd.Flags().GetBool("yes")
		if !autoConfirm {
			fmt.Fprintf(os.Stderr, "Delete project %d? [y/N] ", id)
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

		params := project.NewDeleteProjectProjectIDParams()
		params.ProjectID = id

		_, err = apiClient.Project.DeleteProjectProjectID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to delete project: %w", err)
		}

		style.Success(fmt.Sprintf("Deleted project %d", id))
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectDeleteCmd)
}
