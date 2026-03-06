package command

import "github.com/spf13/cobra"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "バージョン情報を表示します",
		Long:  "shoku CLIのバージョン情報を表示します。",
		Run:   runNotImplemented("version"),
	}
}
