package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/paths"
	"github.com/raphaelneumann/claudectx/internal/profile"
)

var envsEdit bool

const envsFileHeader = `# claudectx per-profile environment for ` + "`claude`" + `.
# One KEY=VALUE per line; blank lines and lines starting with # are ignored.
# These vars are loaded into the claude process only (not your shell) when you
# launch claude in this profile.
#
# Example — Google Vertex AI:
#   CLAUDE_CODE_USE_VERTEX=1
#   CLOUD_ML_REGION=us-east5
#   ANTHROPIC_VERTEX_PROJECT_ID=my-gcp-project
`

var envsCmd = &cobra.Command{
	Use:   "envs [name]",
	Short: "Show (or --edit) the env vars loaded when claude runs in a profile",
	Long: "Each profile has an env file (claudectx.env) whose variables claudectx loads\n" +
		"into the `claude` process when you launch it in that profile — e.g.\n" +
		"CLAUDE_CODE_USE_VERTEX and its Vertex config for a work account. The vars apply\n" +
		"only to claude, not to your shell.\n\n" +
		"With no name, the active profile is used (this terminal's profile if set, else\n" +
		"the default profile).\n\n" +
		"  claudectx envs              show the active profile's env vars\n" +
		"  claudectx envs work         show the work profile's env vars\n" +
		"  claudectx envs work --edit  open work's env file in $EDITOR",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		dir, label, err := resolveEnvDir(name)
		if err != nil {
			return err
		}
		file := filepath.Join(dir, profile.EnvFileName)

		if envsEdit {
			if err := editEnvFile(file); err != nil {
				return err
			}
			entries, _ := profile.LoadEnv(dir)
			fmt.Printf("%s now has %d environment variable(s)\n", label, len(entries))
			return nil
		}

		entries, err := profile.LoadEnv(dir)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Printf("%s has no environment variables\n", label)
			fmt.Printf("add some with: claudectx envs %s --edit\n", displayName(name))
			return nil
		}
		for _, e := range entries {
			fmt.Println(e)
		}
		return nil
	},
}

// resolveEnvDir returns the absolute config dir (and a human label) for the named
// profile, or the active profile when name is empty: this terminal's CLAUDE_CONFIG_DIR
// if set, else the default profile.
func resolveEnvDir(name string) (dir, label string, err error) {
	if name != "" {
		if !profile.Exists(name) {
			return "", "", fmt.Errorf("profile %q does not exist", name)
		}
		abs, err := absProfileDir(name)
		return abs, fmt.Sprintf("profile %q", name), err
	}
	if cd := os.Getenv("CLAUDE_CONFIG_DIR"); cd != "" {
		abs, err := filepath.Abs(cd)
		return abs, fmt.Sprintf("profile %q", filepath.Base(abs)), err
	}
	def := defaultProfile()
	if def == "" || !profile.Exists(def) {
		return "", "", fmt.Errorf("no profile given and no active or default profile set (try `claudectx envs <name>`)")
	}
	abs, err := absProfileDir(def)
	return abs, fmt.Sprintf("default profile %q", def), err
}

func absProfileDir(name string) (string, error) {
	d, err := paths.ProfileDir(name)
	if err != nil {
		return "", err
	}
	return filepath.Abs(d)
}

// displayName returns the name to suggest in hints; falls back to a placeholder when
// the active profile was resolved implicitly.
func displayName(name string) string {
	if name == "" {
		return "<name>"
	}
	return name
}

// editEnvFile opens the env file in $VISUAL/$EDITOR (falling back to vi), seeding a
// commented template if the file does not exist yet.
func editEnvFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(envsFileHeader), 0o600); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	// Run via sh -c so an editor with arguments (e.g. "code -w") works; the file is
	// passed as $1.
	c := exec.Command("sh", "-c", editor+` "$1"`, "sh", path)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

func init() {
	envsCmd.Flags().BoolVar(&envsEdit, "edit", false, "open the profile's env file in $EDITOR")
	rootCmd.AddCommand(envsCmd)
}
