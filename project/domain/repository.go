package domain

import (
	"context"
)

// MentionRepository は返信監視対象メンションの永続化を担当します
type MentionRepository interface {
	// Save はメンション監視対象を保存します
	// 既存レコードがある場合でも成功し、CreatedAt は初回作成時のみ設定される実装を推奨します。
	// 同一キー(team:channel:ts:user)の既存レコードがある場合は上書きします
	// バリデーションエラー時は domain.ErrInvalid を返します
	Save(ctx context.Context, m *Mention) error

	// Find は指定キーのメンション監視対象を取得します。
	// 見つかった場合は (obj!=nil, err=nil) を返します。存在しない場合は ErrNotFound。
	// 存在しない場合は domain.ErrNotFound を返します
	Find(ctx context.Context, teamID, channelID, messageTS, userID string) (*Mention, error)

	// MarkReminded は10分後リマインド完了フラグを立てます
	// すでにフラグが立っている場合は何もせずに成功を返します（冪等）
	// 対象レコードが存在しない場合は domain.ErrNotFound を返します
	MarkReminded(ctx context.Context, teamID, channelID, messageTS, userID string) error

	// MarkEscalated は30分後エスカレーション完了フラグを立てます
	// すでにフラグが立っている場合は何もせずに成功を返します（冪等）
	// 対象レコードが存在しない場合は domain.ErrNotFound を返します
	MarkEscalated(ctx context.Context, teamID, channelID, messageTS, userID string) error
}

// TenantRepository はワークスペース設定の永続化を担当します
type TenantRepository interface {
	// Get は指定されたチームIDのワークスペース設定を取得します
	// 存在しない場合は domain.ErrNotFound を返します
	Get(ctx context.Context, teamID string) (*Tenant, error)

	// UpsertBotTokenSecret はBotトークンのシークレット名を保存します
	// レコードが存在しない場合は新規作成し、ある場合は上書きします
	// CreatedAtが未設定の場合は現在時刻で初期化されます
	// バリデーションエラー時は domain.ErrInvalid を返します
	UpsertBotTokenSecret(ctx context.Context, teamID, secretName string) error

	// SetManager は上長のSlackユーザーIDを設定します
	// managerUserIDがnilの場合は上長設定を解除します
	// レコードが存在しない場合は domain.ErrNotFound を返します
	SetManager(ctx context.Context, teamID string, managerUserID *string) error
}
