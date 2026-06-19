package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	completionInstallDir string
	completionDoInstall  bool
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate or install shell completion",
	Long: "Print a shell completion script to stdout, or install it with --install\n" +
		"(or the `install` subcommand).\n\n" +
		"  claudectx completion zsh             # print the zsh script\n" +
		"  claudectx completion zsh --install   # install it for zsh\n" +
		"  claudectx completion install         # install for your current shell",
	Args:                  cobra.MaximumNArgs(1),
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		if completionDoInstall {
			return installCompletion(args[0], completionInstallDir)
		}
		return writeCompletion(args[0], os.Stdout)
	},
}

var completionInstallCmd = &cobra.Command{
	Use:   "install [bash|zsh|fish]",
	Short: "Install the completion script for your shell",
	Long: "Write the completion script to the standard location for the shell\n" +
		"(detected from $SHELL when not given) and print how to activate it.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := ""
		if len(args) == 1 {
			shell = args[0]
		} else {
			shell = detectShell()
		}
		if shell == "" {
			return fmt.Errorf("could not detect shell from $SHELL; pass one, e.g. `claudectx completion install zsh`")
		}
		return installCompletion(shell, completionInstallDir)
	},
}

// writeCompletion renders the completion script for shell to w.
func writeCompletion(shell string, w io.Writer) error {
	switch shell {
	case "bash":
		return rootCmd.GenBashCompletionV2(w, true)
	case "zsh":
		return rootCmd.GenZshCompletion(w)
	case "fish":
		return rootCmd.GenFishCompletion(w, true)
	case "powershell":
		return rootCmd.GenPowerShellCompletionWithDesc(w)
	default:
		return fmt.Errorf("unsupported shell %q (bash|zsh|fish|powershell)", shell)
	}
}

// detectShell maps $SHELL to a supported shell name, or "" if unknown.
func detectShell() string {
	base := filepath.Base(os.Getenv("SHELL"))
	switch {
	case strings.Contains(base, "bash"):
		return "bash"
	case strings.Contains(base, "zsh"):
		return "zsh"
	case strings.Contains(base, "fish"):
		return "fish"
	}
	return ""
}

// installCompletion writes the completion script to the standard per-shell path
// (or dirOverride) and prints activation instructions.
func installCompletion(shell, dirOverride string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}

	var path, note string
	switch shell {
	case "bash":
		dir := dirOverride
		if dir == "" {
			dir = filepath.Join(dataHome, "bash-completion", "completions")
		}
		path = filepath.Join(dir, "claudectx")
		note = "Requires bash-completion (e.g. `brew install bash-completion@2`). Restart your shell."
	case "zsh":
		dir := dirOverride
		if dir == "" {
			dir = filepath.Join(dataHome, "zsh", "site-functions")
		}
		path = filepath.Join(dir, "_claudectx")
		note = "Ensure the directory is on your fpath — add to ~/.zshrc before `compinit`:\n" +
			"  fpath=(" + dir + " $fpath)\n" +
			"  autoload -Uz compinit && compinit\n" +
			"Then restart your shell."
	case "fish":
		dir := dirOverride
		if dir == "" {
			dir = filepath.Join(configHome, "fish", "completions")
		}
		path = filepath.Join(dir, "claudectx.fish")
		note = "fish loads this automatically. Restart your shell."
	case "powershell":
		return fmt.Errorf("powershell auto-install is unsupported; run: claudectx completion powershell >> $PROFILE")
	default:
		return fmt.Errorf("unsupported shell %q (bash|zsh|fish)", shell)
	}

	var buf bytes.Buffer
	if err := writeCompletion(shell, &buf); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("installed %s completion → %s\n", shell, path)
	fmt.Println(note)
	return nil
}

func init() {
	// Replace cobra's default completion command with ours (adds install support).
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	const dirUsage = "install directory (default: the shell's standard location)"
	completionCmd.Flags().BoolVar(&completionDoInstall, "install", false, "install the script instead of printing it")
	completionCmd.Flags().StringVar(&completionInstallDir, "dir", "", dirUsage)
	completionInstallCmd.Flags().StringVar(&completionInstallDir, "dir", "", dirUsage)
	completionCmd.AddCommand(completionInstallCmd)
	rootCmd.AddCommand(completionCmd)
}
