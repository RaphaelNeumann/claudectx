package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/paths"
	"github.com/raphaelneumann/claudectx/internal/profile"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Print the default profile and this terminal's profile",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default profile (persisted).
		def := defaultProfile()
		switch {
		case def == "":
			fmt.Println("default profile: none — set one with `claudectx set default <name>`")
		case profile.Exists(def):
			fmt.Printf("default profile: %s\n", def)
		default:
			fmt.Printf("default profile: %s (NOT FOUND)\n", def)
		}

		// This terminal's profile (CLAUDE_CONFIG_DIR override, if any).
		if cd := os.Getenv("CLAUDE_CONFIG_DIR"); cd != "" {
			fmt.Printf("this terminal:   %s\n", shellProfile(cd))
		} else {
			fmt.Println("this terminal:   (following the default)")
		}
		return nil
	},
}

// shellProfile maps a CLAUDE_CONFIG_DIR value back to a claudectx profile name, or
// describes it if it's external.
func shellProfile(cd string) string {
	abs, err := filepath.Abs(cd)
	if err != nil {
		return cd
	}
	pd, err := paths.ProfilesDir()
	if err != nil {
		return cd
	}
	if absPd, err := filepath.Abs(pd); err == nil && filepath.Dir(abs) == absPd {
		return filepath.Base(abs)
	}
	return cd + " (external, not a claudectx profile)"
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
