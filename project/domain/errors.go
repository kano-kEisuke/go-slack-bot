package domain

import "errors"

// ドメインエラー定義
var (
	// ErrInvalid は不正な値が設定された場合のエラー
	ErrInvalid = errors.New("ドメイン: 不正な値です")

	// ErrNotFound は要求されたリソースが見つからない場合のエラー
	ErrNotFound = errors.New("ドメイン: リソースが見つかりません")

	// Tenant 関連エラー
	// ErrTenantNotRegistered はテナントが登録されていない場合のエラー
	ErrTenantNotRegistered = errors.New("ドメイン: テナントが登録されていません")

	// ErrBotTokenNotFound は Bot トークンが Secret Manager に見つからない場合のエラー
	ErrBotTokenNotFound = errors.New("ドメイン: Bot トークンが見つかりません")

	// Mention 関連エラー
	// ErrMentionNotFound は監視対象のメンションが見つからない場合のエラー
	ErrMentionNotFound = errors.New("ドメイン: メンション記録が見つかりません")

	// ErrInvalidMentionState は不正なメンション状態の場合のエラー
	ErrInvalidMentionState = errors.New("ドメイン: メンション状態が不正です")

	// Slack API エラー
	// ErrSlackAPIFailed は Slack API 呼び出しが失敗した場合のエラー
	ErrSlackAPIFailed = errors.New("ドメイン: Slack API 呼び出し失敗")

	// ErrUserNotFound はユーザーが見つからない場合のエラー
	ErrUserNotFound = errors.New("ドメイン: ユーザーが見つかりません")

	// ErrChannelNotFound はチャンネルが見つからない場合のエラー
	ErrChannelNotFound = errors.New("ドメイン: チャンネルが見つかりません")

	// ErrMessageNotFound はメッセージが見つからない場合のエラー
	ErrMessageNotFound = errors.New("ドメイン: メッセージが見つかりません")

	// ErrInsufficientPermission は権限不足の場合のエラー
	ErrInsufficientPermission = errors.New("ドメイン: 権限不足です")

	// Task キューイングエラー
	// ErrTaskEnqueueFailed はタスク登録に失敗した場合のエラー
	ErrTaskEnqueueFailed = errors.New("ドメイン: タスク登録失敗")

	// Secret Manager エラー
	// ErrSecretNotFound はシークレットが見つからない場合のエラー
	ErrSecretNotFound = errors.New("ドメイン: シークレットが見つかりません")

	// ErrSecretAccessFailed はシークレットアクセスに失敗した場合のエラー
	ErrSecretAccessFailed = errors.New("ドメイン: シークレットアクセス失敗")

	// Database エラー
	// ErrDatabaseError はデータベース操作に失敗した場合のエラー
	ErrDatabaseError = errors.New("ドメイン: データベースエラー")

	// Firestore トランザクションエラー
	// ErrTransactionFailed はトランザクション実行に失敗した場合のエラー
	ErrTransactionFailed = errors.New("ドメイン: トランザクション失敗")
)
