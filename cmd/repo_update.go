package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var repoUpdateCmd = &cobra.Command{
	Use:   "update <id> [field=value...]",
	Short: "Update a repository",
	Long:  `Update a repository. Fields: name, git_url, git_branch, ssh_key_id.`,
	Args:  cobra.MinimumNArgs(1),
	Example: `  semctl repo update 1 name="Renamed Repo"
  semctl repo update 2 git_branch=develop ssh_key_id=3`,
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

		// Fetch current repository
		getParams := repository.NewGetProjectProjectIDRepositoriesRepositoryIDParams()
		getParams.ProjectID = int64(pid)
		getParams.RepositoryID = id
		getResp, err := apiClient.Repository.GetProjectProjectIDRepositoriesRepositoryID(getParams, nil)
		if err != nil {
			return fmt.Errorf("failed to get repository: %w", err)
		}
		r := getResp.GetPayload()

		req := &models.RepositoryRequest{
			ID:        r.ID,
			ProjectID: int64(pid),
			Name:      r.Name,
			GitURL:    r.GitURL,
			GitBranch: r.GitBranch,
			SSHKeyID:  r.SSHKeyID,
		}

		if len(args) < 2 {
			return fmt.Errorf("no fields to update — provide field=value pairs")
		}

		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid argument %q — expected field=value", arg)
			}
			switch key {
			case "name":
				req.Name = value
			case "git_url":
				req.GitURL = value
			case "git_branch":
				req.GitBranch = value
			case "ssh_key_id":
				n, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid value for ssh_key_id: %w", err)
				}
				req.SSHKeyID = n
			default:
				return fmt.Errorf("unknown field %q — valid fields: name, git_url, git_branch, ssh_key_id", key)
			}
		}

		putParams := repository.NewPutProjectProjectIDRepositoriesRepositoryIDParams()
		putParams.ProjectID = int64(pid)
		putParams.RepositoryID = id
		putParams.Repository = req

		_, err = apiClient.Repository.PutProjectProjectIDRepositoriesRepositoryID(putParams, nil)
		if err != nil {
			return fmt.Errorf("failed to update repository: %w", err)
		}

		style.Success(fmt.Sprintf("Updated repository %d", id))
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoUpdateCmd)
}
