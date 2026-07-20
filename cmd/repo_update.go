package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var repoUpdateCmd = &cobra.Command{
	Use:   "update <id|name> [field=value...]",
	Short: "Update a repository",
	Long:  `Update a repository. Fields: name, git_url, git_branch, ssh_key_id.`,
	Args:  cobra.MinimumNArgs(1),
	Example: `  semctl repo update 1 name="Renamed Repo"
  semctl repo update 2 git_branch=develop ssh_key_id=3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := resolveIDOrName(cmd, args[0], "repository", repoNameIDs)
		if err != nil {
			return err
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
			interactive, ferr := shouldAutoInteractive(cmd, true)
			if ferr != nil {
				return ferr
			}
			if !interactive {
				return fmt.Errorf("no fields to update — provide field=value pairs")
			}
			if err := repoUpdateForm(cmd, req); err != nil {
				return err
			}
		}

		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("invalid argument %q — expected field=value", arg)
			}
			key = strings.ReplaceAll(key, "-", "_") // accept kebab-case like the create flags
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

// repoUpdateForm edits req in place, pre-filled with the current values.
func repoUpdateForm(cmd *cobra.Command, req *models.RepositoryRequest) error {
	keyOpts, err := nameIDOptions(cmd, keyNameIDs, true)
	if err != nil {
		return err
	}
	return runForm(newForm(
		huh.NewGroup(
			huh.NewInput().Title("Name").Value(&req.Name).
				Validate(requireValue("name")),
			huh.NewInput().Title("Git URL").Value(&req.GitURL).
				Validate(requireValue("git URL")),
			huh.NewInput().Title("Git branch").Value(&req.GitBranch),
			huh.NewSelect[int64]().Title("SSH key").Options(keyOpts...).Value(&req.SSHKeyID),
		).Title("Edit repository"),
	))
}

func init() {
	repoCmd.AddCommand(repoUpdateCmd)
}
