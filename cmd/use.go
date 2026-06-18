package cmd

import (
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <name> [-- claude-args...]",
	Short: "Set the current profile to <name> and launch claude",
	Long: "Saves <name> as the current profile (so bare `claudectx` goes there next\n" +
		"time), then replaces the current process with `claude`, with CLAUDE_CONFIG_DIR\n" +
		"set to the profile. Arguments after the name are forwarded to claude:\n\n" +
		"  claudectx use work -- -p \"summarize the diff\"",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return useProfile(args[0], args[1:])
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
