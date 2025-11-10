package domain

import (
	"fmt"
	"strings"
)

// ワークスペース（Slackチーム）ごとの設定
type Tenant struct {
	// TeamID はSlackワークスペースのID
	TeamID string

	// ManagerUserID は上長のSlackユーザーID。
	// nilの場合は上長未設定を表します
	ManagerUserID *string

	// BotTokenSecretName はSecret Managerに保存されたBotトークンのシークレット名
	BotTokenSecretName string

	// CreatedAt はレコードの作成日時（Unix秒）
	CreatedAt int64
}

// 返信待ちの監視対象メンション構造体
type Mention struct {
	// TeamID はSlackワークスペースのID
	TeamID string

	// ChannelID はメンションが発生したチャンネルのID
	ChannelID string

	// MessageTS はメンションを含む親メッセージのタイムスタンプ
	MessageTS string

	// MentionedUserID は返信を期待されているユーザーのID
	MentionedUserID string

	// CreatedAt はレコードの作成日時（Unix秒）
	CreatedAt int64

	// Reminded は10分後の初回リマインドが完了したかどうか
	Reminded bool

	// Escalated は30分後の再リマインド＆上長通知が完了したかどうか
	Escalated bool
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
