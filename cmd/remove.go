package cmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/profile"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Delete a profile",
	Long: "Deletes profiles/<name>/ and its isolated Claude Code state. The shared\n" +
		"agents/skills/commands are not affected. The Keychain credential slot is\n" +
		"left as-is.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !profile.Exists(name) {
			return fmt.Errorf("profile %q does not exist", name)
		}
		if !removeForce {
			confirm := false
			err := huh.NewConfirm().
				Title(fmt.Sprintf("Delete profile %q and its local Claude Code state?", name)).
				Description("Shared agents/skills are kept. Keychain credentials are left in place.").
				Value(&confirm).
				Run()
			if err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					fmt.Println("aborted")
					return nil
				}
				return err
			}
			if !confirm {
				fmt.Println("aborted")
				return nil
			}
		}
		if err := profile.Remove(name); err != nil {
			return err
		}
		fmt.Printf("Removed profile %q.\n", name)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "skip confirmation")
	rootCmd.AddCommand(removeCmd)
}
