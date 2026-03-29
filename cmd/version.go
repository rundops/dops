package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.10.0"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "do(ops) version %s\n", version)
			return nil
		},
	}
}
