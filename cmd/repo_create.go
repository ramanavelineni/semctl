package cmd

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/internal/output"
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

		interactive, err := shouldAutoInteractive(cmd, name == "" || gitURL == "")
		if err != nil {
			return err
		}
		if interactive {
			sshKeyIDStr := ""
			if sshKeyID != 0 {
				sshKeyIDStr = strconv.FormatInt(sshKeyID, 10)
			}
			if err := runForm(newForm(
				huh.NewGroup(
					huh.NewInput().Title("Repository name").Value(&name).
						Validate(requireValue("name")),
					huh.NewInput().Title("Git URL").Value(&gitURL).
						Validate(requireValue("git URL")),
					huh.NewInput().Title("Git branch").Value(&gitBranch),
					huh.NewInput().Title("SSH key ID").
						Description("semctl key list shows available keys").
						Value(&sshKeyIDStr).
						Validate(optionalInt("SSH key ID")),
				).Title("New repository"),
			)); err != nil {
				return err
			}
			sshKeyID = parseOptionalInt(sshKeyIDStr)
		}

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
		// Machine-readable resource on stdout so pipelines can capture the ID.
		if output.GetFormat() != output.FormatTable {
			output.Print(r, nil, nil)
		}
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
