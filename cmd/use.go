package cmd

import (
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <name> [-- claude-args...]",
	Short: "Launch claude in the named profile",
	Long: "Replaces the current process with `claude`, with CLAUDE_CONFIG_DIR set to\n" +
		"the named profile. Any arguments after the name are forwarded to claude:\n\n" +
		"  claudectx use work -- -p \"summarize the diff\"",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return useProfile(args[0], args[1:])
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
