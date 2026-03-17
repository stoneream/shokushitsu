package command

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/stoneream/shokushitsu/internal/appenv"
	"github.com/stoneream/shokushitsu/internal/config"
	"github.com/stoneream/shokushitsu/internal/storage/sqlite"
	hometui "github.com/stoneream/shokushitsu/internal/tui/home"
	tracktui "github.com/stoneream/shokushitsu/internal/tui/track"
)

const japaneseHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}使い方:
  {{if .Runnable}}{{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}{{.CommandPath}} [コマンド]{{end}}{{if gt (len .Aliases) 0}}

別名:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

例:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

利用可能なコマンド:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

フラグ:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

共通フラグ:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

追加ヘルプ:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

詳細は "{{.CommandPath}} [コマンド] --help" を参照してください。{{end}}
`

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	trackCmd := newTrackCmd()
	summaryCmd := newSummaryCmd()
	versionCmd := newVersionCmd()

	cmd := &cobra.Command{
		Use:           "shoku",
		Long:          "shokushitsu は作業時間の記録と集計を行うためのCLIツールです。",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := appenv.ConfigPath()
			if err != nil {
				return err
			}

			return config.EnsureFile(configPath)
		},
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for {
				summaryDates, err := loadSummaryDates(context.Background())
				if err != nil {
					return err
				}

				result, err := hometui.Run(summaryDates)
				if err != nil {
					return err
				}

				switch result.Action {
				case hometui.ActionTrack:
					trackResult, err := runTrack(cmd.Context())
					if err != nil {
						return err
					}
					if trackResult.Message != "" {
						fmt.Fprintln(cmd.OutOrStdout(), trackResult.Message)
					}
					if trackResult.Action == tracktui.ActionReturnHome {
						continue
					}
					return nil
				case hometui.ActionSummary:
					if err := summaryCmd.Flags().Set("date", result.SummaryDate); err != nil {
						return err
					}
					return executeCommand(summaryCmd)
				case hometui.ActionQuit:
					return nil
				default:
					return fmt.Errorf("unknown action: %s", result.Action)
				}
			}
		},
	}
	cmd.SetHelpTemplate(japaneseHelpTemplate)
	cmd.SetUsageTemplate(japaneseHelpTemplate)
	cmd.PersistentFlags().BoolP("help", "h", false, "ヘルプを表示します")

	cmd.SetHelpCommand(&cobra.Command{
		Use:   "help [コマンド名]",
		Short: "コマンドのヘルプを表示します",
		Long:  "指定したコマンドの詳細なヘルプを表示します。",
		Args:  cobra.MaximumNArgs(1),
		Run: func(helpCmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	})

	cmd.AddCommand(
		trackCmd,
		summaryCmd,
		versionCmd,
	)

	return cmd
}

func executeCommand(c *cobra.Command) error {
	if c.RunE != nil {
		return c.RunE(c, nil)
	}

	if c.Run != nil {
		c.Run(c, nil)
		return nil
	}

	return fmt.Errorf("command %q has no runnable handler", c.Name())
}

func loadSummaryDates(ctx context.Context) ([]string, error) {
	dbPath, err := appenv.DBPath()
	if err != nil {
		return nil, err
	}

	store, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = store.Close()
	}()

	dates, err := store.ListSessionStartedDates(ctx, time.Local)
	if err != nil {
		return nil, err
	}

	formatted := make([]string, 0, len(dates))
	for _, d := range dates {
		formatted = append(formatted, d.Format(summaryDateLayout))
	}

	return formatted, nil
}
