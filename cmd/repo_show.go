package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/spf13/cobra"
)

var repoShowCmd = &cobra.Command{
	Use:     "show <id>",
	Short:   "Show repository details",
	Args:    cobra.ExactArgs(1),
	Example: "  semctl repo show 1",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid repository ID: %w", err)
		}

		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := repository.NewGetProjectProjectIDRepositoriesRepositoryIDParams()
		params.ProjectID = int64(pid)
		params.RepositoryID = id

		resp, err := apiClient.Repository.GetProjectProjectIDRepositoriesRepositoryID(params, nil)
		if err != nil {
			return fmt.Errorf("failed to get repository: %w", err)
		}

		r := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(r, nil, nil)
			return nil
		}

		headers := []string{"Field", "Value"}
		rows := [][]string{
			{"ID", strconv.FormatInt(r.ID, 10)},
			{"Name", r.Name},
			{"Git URL", r.GitURL},
			{"Git Branch", r.GitBranch},
			{"SSH Key ID", strconv.FormatInt(r.SSHKeyID, 10)},
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoShowCmd)
}
