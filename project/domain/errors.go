package domain

import "errors"

// ドメインエラー定義
var (
	// ErrInvalid は不正な値が設定された場合のエラー
	ErrInvalid = errors.New("ドメイン: 不正な値です")

	// ErrNotFound は要求されたリソースが見つからない場合のエラー
	ErrNotFound = errors.New("ドメイン: リソースが見つかりません")
)
