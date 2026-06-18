package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/paths"
	"github.com/raphaelneumann/claudectx/internal/profile"
)

const shellInitScript = `# claudectx shell integration.
# Add to ~/.zshrc (or ~/.bashrc):  eval "$(claudectx shell-init)"
#
# Use the launcher form (claudectx use <name>) to run claude in a profile, OR use
# the function below to set CLAUDE_CONFIG_DIR for the *current* shell so a plain
# ` + "`claude`" + ` (and other tools) inherit it.
claudectx-use() {
  local _dir
  _dir="$(command claudectx _profile-dir "$1")" || return $?
  export CLAUDE_CONFIG_DIR="$_dir"
  echo "CLAUDE_CONFIG_DIR set to profile '$1' for this shell."
}
`

var shellInitCmd = &cobra.Command{
	Use:   "shell-init",
	Short: "Print a shell function for the env-var shim",
	Long: "Prints a shell function (claudectx-use) that exports CLAUDE_CONFIG_DIR for\n" +
		"the current shell. Source it with: eval \"$(claudectx shell-init)\"",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(shellInitScript)
		return nil
	},
}

// profileDirCmd is a hidden helper used by the shell shim to resolve a profile to
// its absolute config dir.
var profileDirCmd = &cobra.Command{
	Use:    "_profile-dir <name>",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !profile.Exists(name) {
			return fmt.Errorf("profile %q does not exist", name)
		}
		dir, err := paths.ProfileDir(name)
		if err != nil {
			return err
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		fmt.Println(abs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(shellInitCmd, profileDirCmd)
}
