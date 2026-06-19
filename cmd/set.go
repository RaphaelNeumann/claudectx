package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/profile"
	"github.com/raphaelneumann/claudectx/internal/store"
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Change persistent claudectx settings",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var setDefaultCmd = &cobra.Command{
	Use:   "default <name>",
	Short: "Set the default profile (used by new terminals and plain `claude`)",
	Long: "Persists <name> as the default profile. New terminals — and any `claude`\n" +
		"with the shell integration but no per-terminal override — use it.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !profile.Exists(name) {
			return fmt.Errorf("profile %q does not exist (run `claudectx add %s`)", name, name)
		}
		if err := store.SetLastUsed(name); err != nil {
			return err
		}
		fmt.Printf("default profile set to %q\n", name)
		return nil
	},
}

func init() {
	setCmd.AddCommand(setDefaultCmd)
	rootCmd.AddCommand(setCmd)
}
