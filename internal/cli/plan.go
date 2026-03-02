package cli

import (
	"fmt"

	"github.com/pstradowski/beads-plan/internal/config"
	"github.com/spf13/cobra"
)

var (
	profileFlag string
	dryRunFlag  bool
)

var planCmd = &cobra.Command{
	Use:   "plan <change-name>",
	Short: "Create beads epic from OpenSpec change",
	Long:  "Read OpenSpec artifacts and create a nested beads epic with sub-epics, tasks, dependencies, tier assignments, and parallelism analysis.",
	Args:  cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := checkBd(); err != nil {
			return err
		}
		return checkOpenSpec()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		changeName := args[0]

		cfg, err := config.LoadDefault()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		profileName := config.ActiveProfile(cfg, profileFlag)
		var profile *config.Profile
		if profileName != "" {
			profile, err = config.GetProfile(cfg, profileName)
			if err != nil {
				return err
			}
		}

		_ = changeName
		_ = profile

		fmt.Println("plan: not implemented")
		return nil
	},
}

func init() {
	planCmd.Flags().StringVar(&profileFlag, "profile", "", "Provider profile for tier-to-model mapping")
	planCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Preview planned structure without creating beads")
}
