package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rneumann/claudectx/internal/profile"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a profile",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := profile.Rename(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Renamed %q -> %q.\n", args[0], args[1])
		fmt.Println("Warning: the Keychain credential slot is derived from the profile path,")
		fmt.Println("so the renamed profile will require a fresh `claude` login.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
