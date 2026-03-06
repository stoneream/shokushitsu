package command

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/stoneream/shokushitsu/internal/appenv"
	"github.com/stoneream/shokushitsu/internal/storage/sqlite"
)

const summaryDateLayout = "2006-01-02"

func newSummaryCmd() *cobra.Command {
	var dateArg string

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "作業時間のサマリーを表示します",
		Long:  "指定した期間の作業ログを集計し、結果を表示します。",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDate, err := resolveSummaryDate(dateArg)
			if err != nil {
				return err
			}

			dbPath, err := appenv.DBPath()
			if err != nil {
				return err
			}

			store, err := sqlite.Open(dbPath)
			if err != nil {
				return err
			}
			defer func() {
				_ = store.Close()
			}()

			loc := time.Local
			from := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, loc)
			to := from.AddDate(0, 0, 1)

			items, err := store.ListDailySummaryItemsByStartedRange(context.Background(), from, to)
			if err != nil {
				return err
			}

			unfinished, err := store.CountUnfinishedSessionsByStartedRange(context.Background(), from, to)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "日次サマリー: %s\n", from.Format(summaryDateLayout))
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"対象範囲: %s - %s\n\n",
				from.Format("2006-01-02 15:04:05 MST"),
				to.Format("2006-01-02 15:04:05 MST"),
			)

			var totalSeconds int64
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "終了済みセッションはありません。")
			} else {
				currentProject := ""
				for _, item := range items {
					if item.ProjectName != currentProject {
						if currentProject != "" {
							fmt.Fprintln(cmd.OutOrStdout())
						}
						currentProject = item.ProjectName
						fmt.Fprintf(cmd.OutOrStdout(), "【%s】\n", currentProject)
					}

					fmt.Fprintf(
						cmd.OutOrStdout(),
						"  %s: %s\n",
						item.TaskName,
						formatHHMMSS(item.DurationSeconds),
					)
					totalSeconds += item.DurationSeconds
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\n総作業時間: %s\n", formatHHMMSS(totalSeconds))
			if unfinished > 0 {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"警告: 未終了セッションが %d 件あります（集計対象外）。\n",
					unfinished,
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&dateArg, "date", "", "集計対象日 (YYYY-MM-DD)。未指定時は当日")
	return cmd
}

func resolveSummaryDate(raw string) (time.Time, error) {
	if raw == "" {
		now := time.Now().In(time.Local)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local), nil
	}

	d, err := time.ParseInLocation(summaryDateLayout, raw, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("--date は YYYY-MM-DD 形式で指定してください: %q", raw)
	}

	return d, nil
}

func formatHHMMSS(totalSeconds int64) string {
	if totalSeconds < 0 {
		totalSeconds = 0
	}

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
