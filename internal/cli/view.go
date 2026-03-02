package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view <epic-id>",
	Short: "Generate tasks.md from beads epic",
	Long:  "Read a beads epic recursively and produce an OpenSpec-compatible tasks.md with checkbox status, bead IDs, and tier tags.",
	Args:  cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return checkBd()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("view: not implemented")
		return nil
	},
}
