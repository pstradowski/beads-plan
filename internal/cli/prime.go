package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Emit agent skill definition",
	Long:  "Output a SKILL.md file that teaches a coding agent how to use beads-plan, interpret metadata fields, and dispatch subagents using parallelism hints.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return checkBd()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("prime: not implemented")
		return nil
	},
}
