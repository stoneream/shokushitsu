---
title: shokushitsu features
---

- `--json` 出力追加（`summary` から開始
- 異常終了後の復帰ロジック（未終了セッション検出と再開/終了確認
- `doctor` コマンド追加（未終了セッションや不整合データを検知
- `recent audit` コマンド追加（最近使用アイテムの棚卸し
  - `project_name + task_name` ごとの `last_used_at`, `use_count` を表
  - `--stale-days` で未使用期間の閾値を指定して棚卸しする
  - `ended_at IS NULL` は集計対象外にする
- ログとエラーメッセージの整備
- CI/CD
