package secret

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// Manager は Secret Manager を通じてシークレットを取得するクライアントです
type Manager struct {
	client    *secretmanager.Client
	projectID string
}

// NewManager は Secret Manager のマネージャーを初期化します
func NewManager(ctx context.Context, projectID string) (*Manager, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("secret manager: クライアント初期化失敗: %w", err)
	}

	return &Manager{
		client:    client,
		projectID: projectID,
	}, nil
}

// GetSecret は指定されたシークレット名から最新版のシークレット値を取得します
func (m *Manager) GetSecret(ctx context.Context, secretName string) (string, error) {
	// リクエスト作成
	// リソース名形式: projects/{project_id}/secrets/{secret_name}/versions/latest
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", m.projectID, secretName)

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	// シークレットにアクセス
	result, err := m.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("secret manager: シークレット取得失敗 (name=%s): %w", secretName, err)
	}

	// ペイロードからシークレット値を抽出
	secret := string(result.Payload.Data)
	if secret == "" {
		return "", fmt.Errorf("secret manager: シークレット値が空です (name=%s)", secretName)
	}

	return secret, nil
}

// Close は Secret Manager クライアントを閉じます
func (m *Manager) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}
