package cli

import (
	"fmt"
	"os"

	"github.com/pstradowski/beads-plan/internal/viewer"
	"github.com/spf13/cobra"
)

var viewOutputFlag string

var viewCmd = &cobra.Command{
	Use:   "view <epic-id>",
	Short: "Generate tasks.md from beads epic",
	Long:  "Read a beads epic recursively and produce an OpenSpec-compatible tasks.md with checkbox status, bead IDs, and tier tags.",
	Args:  cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return checkBd()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		epicID := args[0]

		reader := &viewer.BdReader{}
		tree, err := viewer.ReadEpicTree(reader, epicID)
		if err != nil {
			return fmt.Errorf("reading epic tree: %w", err)
		}

		md := viewer.RenderTasksMd(tree, viewer.DefaultRenderOptions())

		if viewOutputFlag != "" {
			if err := os.WriteFile(viewOutputFlag, []byte(md), 0644); err != nil {
				return fmt.Errorf("writing output file: %w", err)
			}
			fmt.Printf("Written to %s\n", viewOutputFlag)
		} else {
			fmt.Print(md)
		}

		return nil
	},
}

func init() {
	viewCmd.Flags().StringVarP(&viewOutputFlag, "output", "o", "", "Write output to file instead of stdout")
}
