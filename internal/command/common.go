package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runNotImplemented(name string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s はまだ実装されていません\n", name)
	}
}
