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

// PutSecret はシークレット値を保存または更新します
func (m *Manager) PutSecret(ctx context.Context, secretName, secretValue string) error {
	// リソース名
	name := fmt.Sprintf("projects/%s/secrets/%s", m.projectID, secretName)

	// シークレット作成リクエスト（既存の場合はスキップ）
	createReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", m.projectID),
		SecretId: secretName,
		Secret:   &secretmanagerpb.Secret{},
	}

	// 既存チェック（GetSecret でシークレット存在確認）
	_, err := m.client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{Name: name})
	if err != nil {
		// シークレットが存在しない場合、作成を試みる（エラーは無視）
		_, _ = m.client.CreateSecret(ctx, createReq)
	}

	// バージョンを追加
	addReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(secretValue),
		},
	}

	_, err = m.client.AddSecretVersion(ctx, addReq)
	if err != nil {
		return fmt.Errorf("secret manager: シークレット保存失敗 (name=%s): %w", secretName, err)
	}

	return nil
}

// Close は Secret Manager クライアントを閉じます
func (m *Manager) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}
