package cmd

import (
	"fmt"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/repository"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var repoCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a repository",
	Example: `  semctl repo create --name "My Repo" --git-url git@github.com:org/repo.git --ssh-key-id 1
  semctl repo create --name "My Repo" --git-url https://github.com/org/repo.git --git-branch main`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := getProjectID(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		gitURL, _ := cmd.Flags().GetString("git-url")
		gitBranch, _ := cmd.Flags().GetString("git-branch")
		sshKeyID, _ := cmd.Flags().GetInt64("ssh-key-id")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if gitURL == "" {
			return fmt.Errorf("--git-url is required")
		}

		req := &models.RepositoryRequest{
			ProjectID: int64(pid),
			Name:      name,
			GitURL:    gitURL,
			GitBranch: gitBranch,
			SSHKeyID:  sshKeyID,
		}

		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		params := repository.NewPostProjectProjectIDRepositoriesParams()
		params.ProjectID = int64(pid)
		params.Repository = req

		resp, err := apiClient.Repository.PostProjectProjectIDRepositories(params, nil)
		if err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}

		r := resp.GetPayload()
		style.Success(fmt.Sprintf("Created repository %q (ID: %d)", r.Name, r.ID))
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoCreateCmd)

	repoCreateCmd.Flags().String("name", "", "repository name (required)")
	repoCreateCmd.Flags().String("git-url", "", "git URL (required)")
	repoCreateCmd.Flags().String("git-branch", "", "default git branch")
	repoCreateCmd.Flags().Int64("ssh-key-id", 0, "SSH key ID")
}
