package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
		fmt.Println("plan: not implemented")
		return nil
	},
}
