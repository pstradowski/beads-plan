package cli

import (
	"github.com/spf13/cobra"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "beads-plan",
	Short: "Convert OpenSpec artifacts into beads execution plans",
	Long:  "beads-plan converts OpenSpec tasks.md into nested beads epics with complexity/tier assessment, parallelism analysis, and self-contained task descriptions for subagent execution.",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output structured JSON instead of human-readable text")
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(primeCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
