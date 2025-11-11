package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"slack-bot/project/infrastructure/config"
	"slack-bot/project/service"
)

// CloudTasksClient は service.TaskPort の Cloud Tasks 実装です
type CloudTasksClient struct {
	project    string
	region     string
	audience   string // OIDC Audience (Cloud Run サービスの URL)
	svcAcct    string // Service Account メールアドレス
	httpClient *http.Client
}

// NewCloudTasksClient は Cloud Tasks クライアントを初期化します
func NewCloudTasksClient(ctx context.Context, cfg *config.Config) (*CloudTasksClient, error) {
	return &CloudTasksClient{
		project:    cfg.GcpProject,
		region:     cfg.Region,
		audience:   cfg.TasksAudience,
		svcAcct:    cfg.TasksServiceAccount,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// EnqueueRemind は10分後のリマインドタスクをキューに登録します
func (ct *CloudTasksClient) EnqueueRemind(ctx context.Context, runAtUnix int64, payload *service.TaskPayload) error {
	queueName := fmt.Sprintf("projects/%s/locations/%s/queues/%s", ct.project, ct.region, "remind-queue")
	return ct.enqueueTask(ctx, queueName, "/check/remind", runAtUnix, payload)
}

// EnqueueEscalate は30分後のエスカレーションタスクをキューに登録します
func (ct *CloudTasksClient) EnqueueEscalate(ctx context.Context, runAtUnix int64, payload *service.TaskPayload) error {
	queueName := fmt.Sprintf("projects/%s/locations/%s/queues/%s", ct.project, ct.region, "escalate-queue")
	return ct.enqueueTask(ctx, queueName, "/check/escalate", runAtUnix, payload)
}

// enqueueTask はタスクを指定されたキューに登録します
func (ct *CloudTasksClient) enqueueTask(ctx context.Context, queueName, path string, runAtUnix int64, payload *service.TaskPayload) error {
	// ペイロードを JSON に変換
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("cloudtasks: ペイロード JSON 化失敗: %w", err)
	}

	// OIDC トークンを取得
	audience := ct.audience

	// Cloud Tasks API への HTTP リクエストボディを構築
	taskBody := map[string]interface{}{
		"httpRequest": map[string]interface{}{
			"uri":        fmt.Sprintf("%s%s", audience, path),
			"body":       string(payloadBytes),
			"headers":    map[string]string{"Content-Type": "application/json"},
			"httpMethod": "POST",
			"oidcToken": map[string]interface{}{
				"serviceAccountEmail": ct.svcAcct,
				"audience":            audience,
			},
		},
		"scheduleTime": time.Unix(runAtUnix, 0).Format(time.RFC3339),
	}

	taskBodyJSON, err := json.Marshal(taskBody)
	if err != nil {
		return fmt.Errorf("cloudtasks: タスクボディ JSON 化失敗: %w", err)
	}

	// Cloud Tasks API エンドポイント
	url := fmt.Sprintf("https://cloudtasks.googleapis.com/v2/%s/tasks", queueName)

	// HTTP リクエスト作成
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(taskBodyJSON))
	if err != nil {
		return fmt.Errorf("cloudtasks: リクエスト作成失敗: %w", err)
	}

	// OIDC トークンをヘッダーに追加
	// Cloud Run から Cloud Tasks API を呼ぶ場合、ワークロード ID 連携で自動的に認証される
	req.Header.Set("Content-Type", "application/json")

	// リクエスト送信
	resp, err := ct.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cloudtasks: リクエスト送信失敗 (queue=%s, path=%s): %w", queueName, path, err)
	}
	defer resp.Body.Close()

	// レスポンスステータスチェック
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudtasks: API エラー (status=%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close は Cloud Tasks クライアントを閉じます（リソースクリーンアップ）
func (ct *CloudTasksClient) Close() error {
	if ct.httpClient != nil {
		ct.httpClient.CloseIdleConnections()
	}
	return nil
}
