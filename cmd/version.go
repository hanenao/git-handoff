package cmd

import (
	"github.com/hanenao/git-handoff/version"
	"github.com/spf13/cobra"
)

func newVersionCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print git-ho version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return printLine(options.stdout, "%s", version.Version)
		},
	}
}
