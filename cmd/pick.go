package cmd

import (
	"github.com/spf13/cobra"
)

var pickCmd = &cobra.Command{
	Use:     "pick",
	Aliases: []string{"switch"},
	Short:   "Interactively choose the current profile and launch it",
	Long: "Opens the interactive picker, saves the choice as the current profile, and\n" +
		"launches claude in it. Bare `claudectx` then goes straight to that profile.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPicker()
	},
}

func init() {
	rootCmd.AddCommand(pickCmd)
}
