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

const shellInitScript = `# claudectx shell integration.
# Add to ~/.zshrc (or ~/.bashrc):  eval "$(claudectx shell-init)"
#
# Wraps ` + "`claude`" + ` so it runs in this terminal's profile (or the default).
claude() {
  if [ -z "${CLAUDE_CONFIG_DIR:-}" ]; then
    _claudectx_dir="$(command claudectx _current-dir 2>/dev/null)"
    if [ -n "$_claudectx_dir" ]; then
      CLAUDE_CONFIG_DIR="$_claudectx_dir" command claude "$@"
      return $?
    fi
  fi
  command claude "$@"
}

# Wraps ` + "`claudectx`" + ` so switching a profile affects THIS terminal:
#   claudectx          pick a profile for this terminal
#   claudectx <name>   switch this terminal to <name>
#   claudectx default  revert this terminal to the default profile
# Everything else (add/remove/list/use/set/current/...) passes through unchanged.
claudectx() {
  case "${1:-}" in
    "" | pick | switch)
      _claudectx_dir="$(command claudectx _pick-dir)" || return $?
      [ -n "$_claudectx_dir" ] || return 0
      export CLAUDE_CONFIG_DIR="$_claudectx_dir"
      echo "claudectx: this terminal now uses '$(basename "$_claudectx_dir")'."
      ;;
    default | reset)
      unset CLAUDE_CONFIG_DIR
      echo "claudectx: this terminal now follows the default profile."
      ;;
    add | remove | rm | rename | list | ls | use | set | current | add-* | shared | shell-init | completion | help | version | -*)
      command claudectx "$@"
      ;;
    *)
      _claudectx_dir="$(command claudectx _profile-dir "$1")" || return $?
      export CLAUDE_CONFIG_DIR="$_claudectx_dir"
      echo "claudectx: this terminal now uses profile '$1'."
      ;;
  esac
}
`

var (
	shellInitDoInstall bool
	shellInitRC        string
)

var shellInitCmd = &cobra.Command{
	Use:   "shell-init",
	Short: "Print or install shell integration (wraps claude to use the current profile)",
	Long: "Prints shell functions to source from ~/.zshrc (or ~/.bashrc):\n" +
		"  eval \"$(claudectx shell-init)\"\n\n" +
		"Defines a `claude` wrapper (runs claude in this terminal's profile, or the\n" +
		"default) and a `claudectx` wrapper so `claudectx <name>` switches the profile\n" +
		"for the current terminal. CLAUDE_CONFIG_DIR set in the shell always wins.\n\n" +
		"Use --install to append the `eval` line to your shell rc file automatically.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if shellInitDoInstall {
			return installShellInit(shellInitRC)
		}
		fmt.Print(shellInitScript)
		return nil
	},
}

// installShellInit appends `eval "$(claudectx shell-init)"` to the shell rc file
// (detected from $SHELL, or rcOverride), idempotently.
func installShellInit(rcOverride string) error {
	rc := rcOverride
	if rc == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		switch detectShell() {
		case "zsh":
			zdot := os.Getenv("ZDOTDIR")
			if zdot == "" {
				zdot = home
			}
			rc = filepath.Join(zdot, ".zshrc")
		case "bash":
			rc = filepath.Join(home, ".bashrc")
		case "fish":
			return fmt.Errorf("fish is not supported by shell-init (bash/zsh syntax); use bash or zsh")
		default:
			return fmt.Errorf("could not detect a supported shell from $SHELL; pass --rc <file>")
		}
	}

	const marker = "claudectx shell-init"
	if data, err := os.ReadFile(rc); err == nil && strings.Contains(string(data), marker) {
		fmt.Printf("shell integration already present in %s\n", rc)
		return nil
	}

	block := "\n# claudectx — make `claude` use the profile selected with claudectx\n" +
		"eval \"$(claudectx shell-init)\"\n"
	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(block); err != nil {
		return err
	}
	fmt.Printf("added claudectx shell integration to %s\n", rc)
	fmt.Printf("restart your shell or run: source %s\n", rc)
	return nil
}

// currentDirCmd prints the absolute config dir of the default profile, or nothing
// if none is set / it no longer exists. Used by the `claude` wrapper as the fallback
// when no per-terminal override (CLAUDE_CONFIG_DIR) is set.
var currentDirCmd = &cobra.Command{
	Use:    "_current-dir",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := defaultProfile()
		if name == "" || !profile.Exists(name) {
			return nil // print nothing → wrapper falls back to plain claude
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

// pickDirCmd runs the interactive picker and prints the chosen profile's absolute
// dir to stdout (UI renders to stderr). Used by the `claudectx` shell function for
// the bare / pick / switch forms.
var pickDirCmd = &cobra.Command{
	Use:    "_pick-dir",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := pickProfile()
		if err != nil {
			return err
		}
		if name == "" {
			return nil // aborted
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
	shellInitCmd.Flags().BoolVar(&shellInitDoInstall, "install", false, "append the integration to your shell rc instead of printing it")
	shellInitCmd.Flags().StringVar(&shellInitRC, "rc", "", "rc file to install into (default: ~/.zshrc or ~/.bashrc from $SHELL)")
	rootCmd.AddCommand(shellInitCmd, profileDirCmd, currentDirCmd, pickDirCmd)
}
