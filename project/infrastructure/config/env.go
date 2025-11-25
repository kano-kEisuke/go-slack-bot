package config

import (
	"fmt"
	"os"
	"time"
)

// Config は環境変数から読み込まれるアプリケーション設定を表します
type Config struct {
	// 基本設定
	AppBaseURL string
	GcpProject string
	Region     string

	// Firestore設定
	FirestoreProjectID string
	CollectionTenants  string
	CollectionMentions string

	// OAuth設定
	OAuthRedirectURL string
	OAuthStateSecret string // Secret Manager推奨

	// Cloud Tasks設定
	TasksQueueRemind    string
	TasksQueueEscalate  string
	TasksAudience       string
	TasksServiceAccount string

	// Slack API設定
	SlackClientID      string // Secret Manager推奨
	SlackClientSecret  string // Secret Manager推奨
	SlackSigningSecret string // Secret Manager推奨
	SecretTokenPrefix  string

	// リマインド設定
	RemindDuration   time.Duration
	EscalateDuration time.Duration
}

// NewConfig は環境変数から設定を読み込み、Config構造体を返します
func NewConfig() (*Config, error) {
	remindAfter := os.Getenv("REMIND_AFTER")
	if remindAfter == "" {
		remindAfter = "10m" // デフォルト値
	}
	remindDuration, err := time.ParseDuration(remindAfter)
	if err != nil {
		return nil, fmt.Errorf("invalid REMIND_AFTER format: %v", err)
	}

	escalateAfter := os.Getenv("ESCALATE_AFTER")
	if escalateAfter == "" {
		escalateAfter = "30m" // デフォルト値
	}
	escalateDuration, err := time.ParseDuration(escalateAfter)
	if err != nil {
		return nil, fmt.Errorf("invalid ESCALATE_AFTER format: %v", err)
	}

	config := &Config{
		// 基本設定
		AppBaseURL: mustGetEnv("APP_BASE_URL"),
		GcpProject: mustGetEnv("GCP_PROJECT"),
		Region:     mustGetEnv("REGION"),

		// Firestore設定
		FirestoreProjectID: mustGetEnv("FIRESTORE_PROJECT_ID"),
		CollectionTenants:  mustGetEnv("FS_COLLECTION_TENANTS"),
		CollectionMentions: mustGetEnv("FS_COLLECTION_MENTIONS"),

		// OAuth設定
		OAuthRedirectURL: mustGetEnv("OAUTH_REDIRECT_URL"),
		OAuthStateSecret: mustGetEnv("OAUTH_STATE_SECRET"),

		// Cloud Tasks設定
		TasksQueueRemind:    mustGetEnv("TASKS_QUEUE_REMIND"),
		TasksQueueEscalate:  mustGetEnv("TASKS_QUEUE_ESCALATE"),
		TasksAudience:       mustGetEnv("TASKS_AUDIENCE"),
		TasksServiceAccount: mustGetEnv("TASKS_SERVICE_ACCOUNT"),

		// Slack API設定
		SlackClientID:      mustGetEnv("SLACK_CLIENT_ID"),
		SlackClientSecret:  mustGetEnv("SLACK_CLIENT_SECRET"),
		SlackSigningSecret: mustGetEnv("SLACK_SIGNING_SECRET"),
		SecretTokenPrefix:  mustGetEnv("SECRET_TOKEN_PREFIX"),

		// リマインド設定
		RemindDuration:   remindDuration,
		EscalateDuration: escalateDuration,
	}

	return config, nil
}

// mustGetEnv は環境変数を取得し、存在しない場合は警告を出して空文字を返します（起動優先）
// 本番では必須値は Cloud Run の環境変数または Secret Manager で必ず設定してください。
func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		// 起動時にパニックせず、ログに警告を出す
		fmt.Fprintf(os.Stderr, "[WARN] required environment variable not set: %s\n", key)
	}
	return value
}
