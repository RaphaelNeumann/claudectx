package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/paths"
	"github.com/raphaelneumann/claudectx/internal/profile"
)

var currentSession bool

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Print the default profile and this terminal's profile",
	Long: "Without flags, prints the default profile and this terminal's profile.\n" +
		"With --session, prints only the profile active in this terminal (the\n" +
		"CLAUDE_CONFIG_DIR override if set, else the default) — handy for a shell prompt.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if currentSession {
			if name := effectiveProfile(); name != "" {
				fmt.Println(name)
			}
			return nil
		}

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

// effectiveProfile returns the profile active in this terminal: the
// CLAUDE_CONFIG_DIR override if set, otherwise the default profile. "" if none.
func effectiveProfile() string {
	if cd := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); cd != "" {
		if name := profileNameForDir(cd); name != "" {
			return name
		}
		return filepath.Base(cd) // external dir → show its basename
	}
	if def := defaultProfile(); def != "" && profile.Exists(def) {
		return def
	}
	return ""
}

// profileNameForDir returns the claudectx profile name a config dir corresponds to,
// or "" if it is not a claudectx profile directory.
func profileNameForDir(cd string) string {
	abs, err := filepath.Abs(cd)
	if err != nil {
		return ""
	}
	pd, err := paths.ProfilesDir()
	if err != nil {
		return ""
	}
	if absPd, err := filepath.Abs(pd); err == nil && filepath.Dir(abs) == absPd {
		return filepath.Base(abs)
	}
	return ""
}

// shellProfile maps a CLAUDE_CONFIG_DIR value back to a claudectx profile name, or
// describes it if it's external.
func shellProfile(cd string) string {
	if name := profileNameForDir(cd); name != "" {
		return name
	}
	return cd + " (external, not a claudectx profile)"
}

func init() {
	currentCmd.Flags().BoolVarP(&currentSession, "session", "s", false, "print only this terminal's active profile (for shell prompts)")
	rootCmd.AddCommand(currentCmd)
}
