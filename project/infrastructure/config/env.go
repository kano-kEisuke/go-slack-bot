package config

import (
	"context"
	"fmt"
	"os"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
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
	OAuthStateSecret string // Secret Manager から読み込み

	// Cloud Tasks設定
	TasksQueueRemind    string
	TasksQueueEscalate  string
	TasksAudience       string
	TasksServiceAccount string

	// Slack API設定
	SlackClientID      string // Secret Manager から読み込み
	SlackClientSecret  string // Secret Manager から読み込み
	SlackSigningSecret string // Secret Manager から読み込み
	SecretTokenPrefix  string

	// リマインド設定
	RemindDuration   time.Duration
	EscalateDuration time.Duration
}

// NewConfig は環境変数から設定を読み込み、Config構造体を返します
// センシティブな情報（Slack認証情報など）はSecret Managerから取得します
func NewConfig(ctx context.Context) (*Config, error) {
	gcpProject := mustGetEnv("GCP_PROJECT")

	// Secret Manager クライアントを初期化
	secretClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("Secret Manager クライアント初期化失敗: %v", err)
	}
	defer secretClient.Close()

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

	// Secret Manager から Slack 認証情報を取得
	slackSigningSecret, err := getSecretFromManager(ctx, secretClient, gcpProject, "slack-signing-secret")
	if err != nil {
		return nil, fmt.Errorf("SLACK_SIGNING_SECRET 取得失敗: %v", err)
	}

	slackClientID, err := getSecretFromManager(ctx, secretClient, gcpProject, "slack-client-id")
	if err != nil {
		return nil, fmt.Errorf("SLACK_CLIENT_ID 取得失敗: %v", err)
	}

	slackClientSecret, err := getSecretFromManager(ctx, secretClient, gcpProject, "slack-client-secret")
	if err != nil {
		return nil, fmt.Errorf("SLACK_CLIENT_SECRET 取得失敗: %v", err)
	}

	oauthStateSecret, err := getSecretFromManager(ctx, secretClient, gcpProject, "oauth-state-secret")
	if err != nil {
		return nil, fmt.Errorf("OAUTH_STATE_SECRET 取得失敗: %v", err)
	}

	config := &Config{
		// 基本設定
		AppBaseURL: mustGetEnv("APP_BASE_URL"),
		GcpProject: gcpProject,
		Region:     mustGetEnv("REGION"),

		// Firestore設定
		FirestoreProjectID: mustGetEnv("FIRESTORE_PROJECT_ID"),
		CollectionTenants:  mustGetEnv("FS_COLLECTION_TENANTS"),
		CollectionMentions: mustGetEnv("FS_COLLECTION_MENTIONS"),

		// OAuth設定
		OAuthRedirectURL: mustGetEnv("OAUTH_REDIRECT_URL"),
		OAuthStateSecret: oauthStateSecret,

		// Cloud Tasks設定
		TasksQueueRemind:    mustGetEnv("TASKS_QUEUE_REMIND"),
		TasksQueueEscalate:  mustGetEnv("TASKS_QUEUE_ESCALATE"),
		TasksAudience:       mustGetEnv("TASKS_AUDIENCE"),
		TasksServiceAccount: mustGetEnv("TASKS_SERVICE_ACCOUNT"),

		// Slack API設定（Secret Manager から取得）
		SlackClientID:      slackClientID,
		SlackClientSecret:  slackClientSecret,
		SlackSigningSecret: slackSigningSecret,
		SecretTokenPrefix:  mustGetEnv("SECRET_TOKEN_PREFIX"),

		// リマインド設定
		RemindDuration:   remindDuration,
		EscalateDuration: escalateDuration,
	}

	return config, nil
}

// getSecretFromManager は Secret Manager から指定されたシークレットを取得します
func getSecretFromManager(ctx context.Context, client *secretmanager.Client, projectID, secretName string) (string, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName)

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Secret Manager からの取得失敗 (name=%s): %w", secretName, err)
	}

	secret := string(result.Payload.Data)
	if secret == "" {
		return "", fmt.Errorf("Secret Manager のシークレット値が空です (name=%s)", secretName)
	}

	return secret, nil
}

// mustGetEnv は環境変数を取得し、存在しない場合はパニックします
func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable not set: %s", key))
	}
	return value
}
