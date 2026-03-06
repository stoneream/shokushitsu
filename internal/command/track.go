package command

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stoneream/shokushitsu/internal/appenv"
	"github.com/stoneream/shokushitsu/internal/config"
	"github.com/stoneream/shokushitsu/internal/storage/sqlite"
	tracktui "github.com/stoneream/shokushitsu/internal/tui/track"
)

func newTrackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "track",
		Short: "作業時間の計測を開始・終了します",
		Long:  "作業セッションの開始、継続、終了を管理します。",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := appenv.ConfigPath()
			if err != nil {
				return err
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			notificationSoundPath := config.ResolveNotificationSoundPath(
				configPath,
				cfg.NotificationSoundPath(),
			)

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

			message, err := tracktui.Run(context.Background(), store, notificationSoundPath)
			if err != nil {
				return err
			}
			if message != "" {
				fmt.Fprintln(cmd.OutOrStdout(), message)
			}

			return nil
		},
	}
}
