package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/launch"
	"github.com/raphaelneumann/claudectx/internal/profile"
	"github.com/raphaelneumann/claudectx/internal/store"
)

// version is overridden at release time via -ldflags -X.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "claudectx [profile]",
	Short: "Switch the Claude Code profile for your terminal",
	Long: "claudectx manages multiple isolated Claude Code profiles (one per account).\n\n" +
		"With the shell integration (`claudectx shell-init`):\n" +
		"  claudectx              pick a profile for THIS terminal\n" +
		"  claudectx <name>       switch THIS terminal to <name>\n" +
		"  claudectx default      revert this terminal to the default profile\n" +
		"  claudectx set default <name>   change the DEFAULT profile (new terminals)\n\n" +
		"A plain `claude` then runs in the terminal's profile (or the default).",
	Version: version,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return printSwitchHint(args[0])
		}
		return printStatus()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// defaultProfile returns the persisted default profile name (state.json), or "".
func defaultProfile() string {
	if st, err := store.Load(); err == nil {
		return st.LastUsed
	}
	return ""
}

// integrationStatus reports whether the shell integration is installed in a known
// rc file, and which file.
func integrationStatus() (installed bool, rc string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, ""
	}
	var candidates []string
	if z := os.Getenv("ZDOTDIR"); z != "" {
		candidates = append(candidates, filepath.Join(z, ".zshrc"))
	}
	candidates = append(candidates,
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
	)
	for _, f := range candidates {
		if data, err := os.ReadFile(f); err == nil && strings.Contains(string(data), "claudectx shell-init") {
			return true, f
		}
	}
	return false, ""
}

// integrationGuidance auto-detects whether the integration is installed and returns
// the appropriate next step (restart vs. install).
func integrationGuidance() string {
	if installed, rc := integrationStatus(); installed {
		return fmt.Sprintf("shell integration is installed in %s — open a new terminal (or `source %s`)", rc, rc)
	}
	return "enable it:  claudectx shell-init --install   (then restart your shell)"
}

// printStatus is shown for bare `claudectx` when the shell integration is not active
// in this shell. It shows the profiles and how to switch.
func printStatus() error {
	infos, err := profile.List()
	if err != nil {
		return err
	}
	if len(infos) == 0 {
		fmt.Println("claudectx — no profiles yet. Create one:  claudectx add <name>")
		return nil
	}

	def := defaultProfile()
	fmt.Println("claudectx — switch Claude Code profiles")
	fmt.Println()
	fmt.Println("Profiles (* = default):")
	for _, in := range infos {
		marker := "  "
		if in.Name == def {
			marker = "* "
		}
		status := "no credential"
		if in.HasCredential {
			status = "logged in"
		}
		fmt.Printf("  %s%-18s %s\n", marker, in.Name, status)
	}
	fmt.Println()
	fmt.Printf("To switch THIS terminal's profile, %s,\n", integrationGuidance())
	fmt.Println("then:  claudectx <name>")
	fmt.Println()
	fmt.Println("Other commands:")
	fmt.Println("  claudectx use <name>          switch and launch claude now")
	fmt.Println("  claudectx set default <name>  change the default profile")
	fmt.Println("  claudectx --help              all commands")
	return nil
}

// printSwitchHint is shown for `claudectx <name>` without the shell integration.
func printSwitchHint(name string) error {
	if !profile.Exists(name) {
		return fmt.Errorf("profile %q does not exist (run `claudectx add %s`)", name, name)
	}
	fmt.Printf("To switch this terminal to %q, %s,\n", name, integrationGuidance())
	fmt.Printf("then:  claudectx %s\n", name)
	fmt.Printf("Or launch it now:  claudectx use %s\n", name)
	return nil
}

// pickProfile shows the interactive selector (rendered to stderr so stdout stays
// clean for capture) and returns the chosen profile name, or "" if aborted.
func pickProfile() (string, error) {
	infos, err := profile.List()
	if err != nil {
		return "", err
	}
	if len(infos) == 0 {
		return "", errors.New("no profiles yet — create one with `claudectx add <name>`")
	}

	st, _ := store.Load()
	options := make([]huh.Option[string], 0, len(infos))
	for _, in := range infos {
		status := "no credential"
		if in.HasCredential {
			status = "logged in"
		}
		options = append(options, huh.NewOption(fmt.Sprintf("%s  (%s)", in.Name, status), in.Name))
	}

	choice := st.LastUsed
	if !profile.Exists(choice) {
		choice = infos[0].Name
	}

	sel := huh.NewSelect[string]().
		Title("Switch this terminal to which Claude Code profile?").
		Options(options...).
		Value(&choice)
	if err := huh.NewForm(huh.NewGroup(sel)).WithOutput(os.Stderr).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", nil
		}
		return "", err
	}
	return choice, nil
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// useProfile records the selection and execs claude in the profile. On success it
// does not return.
func useProfile(name string, args []string) error {
	_ = store.SetLastUsed(name) // best-effort; exec replaces the process next
	return launch.Exec(name, args)
}
