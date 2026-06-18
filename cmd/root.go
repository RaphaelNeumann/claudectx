package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/rneumann/claudectx/internal/launch"
	"github.com/rneumann/claudectx/internal/profile"
	"github.com/rneumann/claudectx/internal/store"
)

// version is overridden at release time via -ldflags -X.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "claudectx",
	Short: "Switch between isolated Claude Code profiles",
	Long: "claudectx manages multiple isolated Claude Code profiles (one per account)\n" +
		"and launches `claude` into the chosen one via CLAUDE_CONFIG_DIR.\n\n" +
		"With no arguments it opens an interactive picker.",
	Version:       version,
	Args:          cobra.NoArgs,
	RunE:          func(cmd *cobra.Command, args []string) error { return runPicker() },
	SilenceUsage:  true,
	SilenceErrors: true,
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
