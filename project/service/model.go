package service

// MentionEvent はSlackメンションイベントを表します
type MentionEvent struct {
	// TeamID はSlackワークスペースのID
	TeamID string

	// ChannelID はメンションが投稿されたチャンネルのID
	ChannelID string

	// MessageTS はメッセージのタイムスタンプ
	MessageTS string

	// Text はメッセージのテキスト（メンション抽出に使用）
	Text string

	// BotUserID はBotのユーザーID（除外対象）
	BotUserID string

	// ParentUserID はメンションを投稿したユーザーID（メンション返信判定に使用）
	ParentUserID string

	// NowUnix はイベント発生時刻（Unix秒）
	NowUnix int64
}

// TaskPayload はCloud Tasksのジョブペイロードを表します
type TaskPayload struct {
	// TeamID はSlackワークスペースのID
	TeamID string

	// ChannelID はメンションが投稿されたチャンネルのID
	ChannelID string

	// MessageTS はメッセージのタイムスタンプ
	MessageTS string

	// UserID は監視対象のユーザーID
	UserID string

	// ParentUserID はメンションを投稿したユーザーID
	ParentUserID string
}
