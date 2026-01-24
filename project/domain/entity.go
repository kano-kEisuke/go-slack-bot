package domain

import (
	"fmt"
	"strings"
)

// ワークスペース（Slackチーム）ごとの設定
type Tenant struct {
	// TeamID はSlackワークスペースのID
	TeamID string `firestore:"team_id"`

	// ManagerUserID は上長のSlackユーザーID。
	// nilの場合は上長未設定を表す
	ManagerUserID *string `firestore:"manager_user_id"`

	// BotTokenSecretName はSecret Managerに保存されたBotトークンのシークレット名
	BotTokenSecretName string `firestore:"bot_token_secret_name"`

	// CreatedAt はレコードの作成日時（Unix秒）
	CreatedAt int64 `firestore:"created_at"`
}

// 返信待ちの監視対象メンション構造体
type Mention struct {
	// TeamID はSlackワークスペースのID
	TeamID string `firestore:"team_id"`

	// ChannelID はメンションが発生したチャンネルのID
	ChannelID string `firestore:"channel_id"`

	// MessageTS はメンションを含む親メッセージのタイムスタンプ
	MessageTS string `firestore:"message_ts"`

	// MentionedUserID は返信を期待されているユーザーのID
	MentionedUserID string `firestore:"mentioned_user_id"`

	// CreatedAt はレコードの作成日時（Unix秒）
	CreatedAt int64 `firestore:"created_at"`

	// Reminded は10分後の初回リマインドが完了したかどうか
	Reminded bool `firestore:"reminded"`

	// Escalated は30分後の再リマインド＆上長通知が完了したかどうか
	Escalated bool `firestore:"escalated"`
}

// MentionKey は監視対象メンションの一意キーを生成します
func MentionKey(teamID, channelID, messageTS, userID string) string {
	return fmt.Sprintf("%s:%s:%s:%s", teamID, channelID, messageTS, userID)
}

// Validate はTenantの必須項目を検証します
func (t Tenant) Validate() error {
	if strings.TrimSpace(t.TeamID) == "" {
		return fmt.Errorf("%w: TeamIDは必須項目です", ErrInvalid)
	}
	if strings.TrimSpace(t.BotTokenSecretName) == "" {
		return fmt.Errorf("%w: BotTokenSecretNameは必須項目です", ErrInvalid)
	}
	if t.CreatedAt <= 0 {
		return fmt.Errorf("%w: CreatedAtは0より大きい必要があります", ErrInvalid)
	}
	return nil
}

// Validate はMentionの必須項目を検証します
func (m Mention) Validate() error {
	if strings.TrimSpace(m.TeamID) == "" {
		return fmt.Errorf("%w: TeamIDは必須項目です", ErrInvalid)
	}
	if strings.TrimSpace(m.ChannelID) == "" {
		return fmt.Errorf("%w: ChannelIDは必須項目です", ErrInvalid)
	}
	if strings.TrimSpace(m.MessageTS) == "" {
		return fmt.Errorf("%w: MessageTSは必須項目です", ErrInvalid)
	}
	if strings.TrimSpace(m.MentionedUserID) == "" {
		return fmt.Errorf("%w: MentionedUserIDは必須項目です", ErrInvalid)
	}
	if m.CreatedAt <= 0 {
		return fmt.Errorf("%w: CreatedAtは0より大きい必要があります", ErrInvalid)
	}
	return nil
}
