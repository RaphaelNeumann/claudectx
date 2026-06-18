package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/paths"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Print the profile for the current shell",
	Long: "Reports which profile $CLAUDE_CONFIG_DIR points at. Because profiles are\n" +
		"per-process (set via the env var), this reflects THIS shell only.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cd := os.Getenv("CLAUDE_CONFIG_DIR")
		if cd == "" {
			fmt.Println("default (CLAUDE_CONFIG_DIR unset — Claude Code uses ~/.claude)")
			return nil
		}
		abs, err := filepath.Abs(cd)
		if err != nil {
			return err
		}
		pd, err := paths.ProfilesDir()
		if err != nil {
			return err
		}
		absPd, err := filepath.Abs(pd)
		if err != nil {
			return err
		}
		if filepath.Dir(abs) == absPd {
			fmt.Println(filepath.Base(abs))
			return nil
		}
		fmt.Printf("%s (external CLAUDE_CONFIG_DIR, not a claudectx profile)\n", cd)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
