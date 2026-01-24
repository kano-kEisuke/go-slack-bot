package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"slack-bot/project/infrastructure/config"
	"slack-bot/project/service"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CloudTasksClient は service.TaskPort の Cloud Tasks 実装です
type CloudTasksClient struct {
	client   *cloudtasks.Client
	project  string
	region   string
	audience string // OIDC Audience (Cloud Run サービスの URL)
	svcAcct  string // Service Account メールアドレス
}

// NewCloudTasksClient は Cloud Tasks クライアントを初期化します
func NewCloudTasksClient(ctx context.Context, cfg *config.Config) (*CloudTasksClient, error) {
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloudtasks: クライアント初期化失敗: %w", err)
	}

	return &CloudTasksClient{
		client:   client,
		project:  cfg.GcpProject,
		region:   cfg.Region,
		audience: cfg.TasksAudience,
		svcAcct:  cfg.TasksServiceAccount,
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

	// タスクリクエストを構築
	task := &cloudtaskspb.Task{
		MessageType: &cloudtaskspb.Task_HttpRequest{
			HttpRequest: &cloudtaskspb.HttpRequest{
				Url:        fmt.Sprintf("%s%s", ct.audience, path),
				HttpMethod: cloudtaskspb.HttpMethod_POST,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       payloadBytes,
				AuthorizationHeader: &cloudtaskspb.HttpRequest_OidcToken{
					OidcToken: &cloudtaskspb.OidcToken{
						ServiceAccountEmail: ct.svcAcct,
						Audience:            ct.audience,
					},
				},
			},
		},
		ScheduleTime: timestamppb.New(time.Unix(runAtUnix, 0)),
	}

	// タスクを作成
	req := &cloudtaskspb.CreateTaskRequest{
		Parent: queueName,
		Task:   task,
	}

	_, err = ct.client.CreateTask(ctx, req)
	if err != nil {
		return fmt.Errorf("cloudtasks: タスク作成失敗 (queue=%s, path=%s): %w", queueName, path, err)
	}

	return nil
}

// Close は Cloud Tasks クライアントを閉じます（リソースクリーンアップ）
func (ct *CloudTasksClient) Close() error {
	if ct.client != nil {
		return ct.client.Close()
	}
	return nil
}
