package cmd

import (
	"github.com/spf13/cobra"
)

var pickCmd = &cobra.Command{
	Use:     "pick",
	Aliases: []string{"switch"},
	Short:   "Interactively switch this terminal's profile (same as bare `claudectx`)",
	Long: "Opens the picker to switch the current terminal's profile. Requires the\n" +
		"shell integration (`claudectx shell-init`); identical to bare `claudectx`.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printStatus()
	},
}

func init() {
	rootCmd.AddCommand(pickCmd)
}
