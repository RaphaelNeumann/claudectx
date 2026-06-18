package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/paths"
	"github.com/raphaelneumann/claudectx/internal/profile"
)

var sharedCmd = &cobra.Command{
	Use:   "shared",
	Short: "Manage the shared agents/skills/commands layer",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return sharedList()
	},
}

var sharedPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the shared layer directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		sh, err := paths.SharedDir()
		if err != nil {
			return err
		}
		fmt.Println(sh)
		return nil
	},
}

var sharedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List shared agents, skills, and commands",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return sharedList()
	},
}

func sharedList() error {
	if err := profile.EnsureShared(); err != nil {
		return err
	}
	sh, err := paths.SharedDir()
	if err != nil {
		return err
	}
	for _, sub := range paths.SharedSubdirs {
		dir := filepath.Join(sh, sub)
		entries, _ := os.ReadDir(dir)
		fmt.Printf("%s/ (%d)\n", sub, len(entries))
		for _, e := range entries {
			fmt.Printf("  %s\n", e.Name())
		}
	}
	fmt.Printf("\nshared dir: %s\n", sh)
	return nil
}

func init() {
	sharedCmd.AddCommand(sharedPathCmd, sharedListCmd)
	rootCmd.AddCommand(sharedCmd)
}
