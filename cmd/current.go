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
	Short: "Print the current profile",
	Long: "Reports the profile that bare `claudectx` would launch — from\n" +
		"CLAUDECTX_PROFILE if set, otherwise the saved default. If $CLAUDE_CONFIG_DIR\n" +
		"is set in this shell (e.g. inside a `use` session), that is shown too.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, source := resolveCurrent()
		switch source {
		case "env":
			if profile.Exists(name) {
				fmt.Printf("current: %s (from %s)\n", name, profileEnv)
			} else {
				fmt.Printf("current: %s (from %s — NOT FOUND)\n", name, profileEnv)
			}
		case "saved":
			if profile.Exists(name) {
				fmt.Printf("current: %s (saved default)\n", name)
			} else {
				fmt.Printf("current: %s (saved default — NOT FOUND)\n", name)
			}
		default:
			fmt.Println("current: none set — run `claudectx pick` or `claudectx use <name>`")
		}

		// Also report the profile this shell is actively running, if any.
		if cd := os.Getenv("CLAUDE_CONFIG_DIR"); cd != "" {
			fmt.Printf("this shell: %s\n", shellProfile(cd))
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
