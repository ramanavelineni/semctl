package cmd

import (
	"strconv"

	"github.com/ramanavelineni/semctl/internal/client"
	"github.com/ramanavelineni/semctl/pkg/semapi/client/operations"
	"github.com/ramanavelineni/semctl/pkg/semapi/models"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:     "info",
	Short:   "Show Semaphore server information",
	Long:    `Show Semaphore server information: version, ansible version, and configuration.`,
	Example: "  semctl info",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient, err := client.NewAuthenticatedClient()
		if err != nil {
			return err
		}

		return runShow("server info",
			func() (*models.InfoType, error) {
				resp, err := apiClient.Operations.GetInfo(operations.NewGetInfoParams(), nil)
				if err != nil {
					return nil, err
				}
				return resp.GetPayload(), nil
			},
			func(i *models.InfoType) [][]string {
				return [][]string{
					{"Version", i.Version},
					{"Ansible", i.Ansible},
					{"Web Host", i.WebHost},
					{"Git Client", i.GitClient},
					{"Schedule Timezone", i.ScheduleTimezone},
					{"Remote Runners", strconv.FormatBool(i.UseRemoteRunner)},
				}
			})
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
