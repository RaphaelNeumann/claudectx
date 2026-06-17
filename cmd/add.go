package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rneumann/claudectx/internal/profile"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new profile",
	Long: "Creates profiles/<name>/ and links the shared agents/skills/commands into\n" +
		"it. Claude Code creates its own state (and credential slot) on first run.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := profile.Add(name); err != nil {
			return err
		}
		fmt.Printf("Created profile %q.\nRun `claudectx use %s` and log in to that account.\n", name, name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
