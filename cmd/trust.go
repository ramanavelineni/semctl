package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramanavelineni/semctl/internal/config"
	"github.com/ramanavelineni/semctl/internal/style"
	"github.com/spf13/cobra"
)

var trustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Trust the config file in the current directory",
	Long: `Trust the semctl.yaml (or .semctl.yaml) file in the current directory.

A config file found in the working directory can redirect semctl — and any
credentials from the environment — to a server you never chose, so it is
ignored until trusted. Trust is bound to the file's content: editing the
file requires trusting it again.`,
	Example: `  semctl trust            # trust ./semctl.yaml (or ./.semctl.yaml)
  semctl trust --revoke   # stop trusting it
  semctl trust --list     # show all trusted config files`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if list, _ := cmd.Flags().GetBool("list"); list {
			paths, err := config.TrustedConfigs()
			if err != nil {
				return err
			}
			if len(paths) == 0 {
				printEmptyList("trusted config files")
				return nil
			}
			for _, p := range paths {
				fmt.Println(p)
			}
			return nil
		}

		path := config.FindCWDConfig()
		if path == "" {
			return fmt.Errorf("no semctl.yaml or .semctl.yaml in the current directory")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		if revoke, _ := cmd.Flags().GetBool("revoke"); revoke {
			removed, err := config.RevokeTrust(path)
			if err != nil {
				return err
			}
			if !removed {
				style.Info(fmt.Sprintf("%s was not trusted; nothing to revoke.", abs))
				return nil
			}
			style.Success(fmt.Sprintf("Revoked trust for %s.", abs))
			return nil
		}

		if err := config.TrustConfig(path); err != nil {
			return err
		}
		style.Success(fmt.Sprintf("Trusted %s — it now loads automatically until its content changes.", abs))
		return nil
	},
}

// warnIfUntrustedSkipped surfaces a CWD config that Load ignored. Root's
// PersistentPreRunE calls it for most commands; commands that load config
// themselves (login, logout, context) call it after their own Load.
func warnIfUntrustedSkipped() {
	if p := config.SkippedUntrustedConfig(); p != "" {
		style.Warning(fmt.Sprintf("Ignoring untrusted config %s — run 'semctl trust' in this directory to use it, or pass it explicitly with --config.", p))
	}
}

// promptTrustCWDConfig is the interactive consent gate behind
// config.TrustPrompt. Non-TTY runs never trust implicitly — CI trusts via
// 'semctl trust' or passes --config explicitly.
func promptTrustCWDConfig(path string, changed bool) bool {
	if !style.IsStdinTTY() {
		return false
	}
	if changed {
		fmt.Fprintf(os.Stderr, "Config %s has changed since it was trusted.\n", path)
	} else {
		fmt.Fprintf(os.Stderr, "Found config %s in the current directory.\nA working-directory config can redirect semctl — and any credentials it uses — to its own server.\n", path)
	}
	fmt.Fprintf(os.Stderr, "Trust this file? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func init() {
	rootCmd.AddCommand(trustCmd)
	trustCmd.Flags().Bool("revoke", false, "stop trusting the current directory's config file")
	trustCmd.Flags().Bool("list", false, "list all trusted config files")
	trustCmd.MarkFlagsMutuallyExclusive("revoke", "list")
}
