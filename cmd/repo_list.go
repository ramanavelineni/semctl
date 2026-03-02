package cmd

import (
	"fmt"
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/spf13/cobra"
)

var repoListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List repositories",
	Example: "  semctl repo list",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := repository.NewGetProjectProjectIDRepositoriesParams()
		params.ProjectID = int64(pid)

		resp, err := apiClient.Repository.GetProjectProjectIDRepositories(params, nil)
		if err != nil {
			return fmt.Errorf("failed to list repositories: %w", err)
		}

		items := resp.GetPayload()

		if output.GetFormat() != output.FormatTable {
			output.Print(items, nil, nil)
			return nil
		}

		headers := []string{"ID", "Name", "Git URL", "Git Branch", "SSH Key ID"}
		var rows [][]string
		for _, r := range items {
			rows = append(rows, []string{
				strconv.FormatInt(r.ID, 10),
				r.Name,
				r.GitURL,
				r.GitBranch,
				strconv.FormatInt(r.SSHKeyID, 10),
			})
		}

		if len(rows) == 0 {
			return fmt.Errorf("no repositories found")
		}

		output.PrintTable(headers, rows)
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)
}
