package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/raphaelneumann/claudectx/internal/profile"
	"github.com/raphaelneumann/claudectx/internal/store"
)

var (
	activeStyle = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List profiles",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		infos, err := profile.List()
		if err != nil {
			return err
		}
		if len(infos) == 0 {
			fmt.Println("No profiles yet. Create one with `claudectx add <name>`.")
			return nil
		}
		st, _ := store.Load()
		for _, in := range infos {
			marker := " "
			name := in.Name
			if in.Name == st.LastUsed {
				marker = "*"
				name = activeStyle.Render(name)
			}
			status := dimStyle.Render("no credential")
			if in.HasCredential {
				status = "logged in"
			}
			fmt.Printf("%s %-24s %s\n", marker, name, status)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
