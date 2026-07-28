package main

import (
	"fmt"

	"github.com/SamyBaouche/tfguard/internal/ui"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the tfguard version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			out := cmd.OutOrStdout()
			style := ui.NewStyle(out)
			fmt.Fprintf(out, "%s %s\n", style.Cyan("tfguard"), style.Bold(Version))
		},
	}
}
