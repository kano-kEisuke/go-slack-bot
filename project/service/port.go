package service

import "context"

// SlackPort は Slack API 呼び出しのポートです
type SlackPort interface {
	// HasUserReplied は指定されたユーザーが指定されたメッセージ以降に返信しているかを判定します
	// oldest パラメータはメッセージの検索開始点を示します（通常は親メッセージの TS）
	HasUserReplied(ctx context.Context, teamID, channelID, messageTS, userID, oldest string) (bool, error)

	// HasUserRepliedWithMention は対象ユーザーが送信元ユーザーへメンション付きで返信しているか判定します
	// userID: チェック対象のユーザー（メンションされた人）
	// parentUserID: トリガーメッセージ送信者のユーザーID（メンションした人）
	HasUserRepliedWithMention(ctx context.Context, teamID, channelID, messageTS, userID, parentUserID, oldest string) (bool, error)

	// PostThreadMessage はスレッドにメッセージを投稿します
	PostThreadMessage(ctx context.Context, teamID, channelID, messageTS, text string) error

	// PostDM は指定されたユーザーにDMを送信します
	PostDM(ctx context.Context, teamID, userID, text string) error
}

// TaskPort は Cloud Tasks へのジョブ予約のポートです
type TaskPort interface {
	// EnqueueRemind は指定時刻に CheckRemind を実行するジョブをキューに登録します
	EnqueueRemind(ctx context.Context, runAt int64, payload *TaskPayload) error

	// EnqueueEscalate は指定時刻に CheckEscalate を実行するジョブをキューに登録します
	EnqueueEscalate(ctx context.Context, runAt int64, payload *TaskPayload) error
}
