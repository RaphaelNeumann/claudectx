package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/launch"
	"github.com/raphaelneumann/claudectx/internal/profile"
	"github.com/raphaelneumann/claudectx/internal/store"
)

// version is overridden at release time via -ldflags -X.
var version = "dev"

// profileEnv overrides the current profile for a single invocation, without
// changing the saved default.
const profileEnv = "CLAUDECTX_PROFILE"

var rootCmd = &cobra.Command{
	Use:   "claudectx",
	Short: "Switch between isolated Claude Code profiles",
	Long: "claudectx manages multiple isolated Claude Code profiles (one per account)\n" +
		"and launches `claude` into the chosen one via CLAUDE_CONFIG_DIR.\n\n" +
		"With no arguments it launches the current profile, resolved in order:\n" +
		"  --profile flag → CLAUDECTX_PROFILE env → saved current → picker (first run).\n" +
		"Use `claudectx use <name>` to change the saved current, or `claudectx pick`\n" +
		"to choose interactively.",
	Version: version,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, _ := cmd.Flags().GetString("profile")
		return runRoot(p)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.Flags().String("profile", "", "launch this profile for one invocation (overrides the saved default)")
}

// resolveCurrent returns the profile that bare `claudectx` would launch and its
// source: "env" (CLAUDECTX_PROFILE), "saved" (persisted default), or "" (none).
func resolveCurrent() (name, source string) {
	if p := strings.TrimSpace(os.Getenv(profileEnv)); p != "" {
		return p, "env"
	}
	if st, err := store.Load(); err == nil && st.LastUsed != "" {
		return st.LastUsed, "saved"
	}
	return "", ""
}

// runRoot handles the no-argument invocation: launch the current profile, or fall
// back to the picker on first run. flagProfile is the --profile value (one-shot
// override; highest precedence, does not persist).
func runRoot(flagProfile string) error {
	if p := strings.TrimSpace(flagProfile); p != "" {
		if !profile.Exists(p) {
			return fmt.Errorf("--profile %q: profile does not exist (run `claudectx add %s`)", p, p)
		}
		return launch.Exec(p, nil)
	}
	name, source := resolveCurrent()
	switch source {
	case "env":
		if !profile.Exists(name) {
			return fmt.Errorf("%s=%q: profile does not exist (run `claudectx add %s`)", profileEnv, name, name)
		}
		return launch.Exec(name, nil) // transient override — do not persist
	case "saved":
		if profile.Exists(name) {
			return launch.Exec(name, nil) // already the saved default — no need to re-persist
		}
		// saved profile was removed; fall through to the picker
	}
	return runPicker()
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

// runPicker shows an interactive profile selector, defaulting to the last used.
func runPicker() error {
	infos, err := profile.List()
	if err != nil {
		return err
	}
	if len(infos) == 0 {
		return errors.New("no profiles yet — create one with `claudectx add <name>`")
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

	form := huh.NewSelect[string]().
		Title("Select a Claude Code profile").
		Options(options...).
		Value(&choice)

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return err
	}
	return useProfile(choice, nil)
}
