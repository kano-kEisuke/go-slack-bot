package store

import (
	"context"
	"fmt"
	"time"

	"slack-bot/project/domain"
	"slack-bot/project/infrastructure/config"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isNotFound は Firestore の NotFound エラーを判定するヘルパー関数です
func isNotFound(err error) bool {
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.NotFound
}

// FirestoreRepo は domain.MentionRepository と domain.TenantRepository の Firestore 実装です
type FirestoreRepo struct {
	cli         *firestore.Client
	tenantsCol  string
	mentionsCol string
}

// NewFirestoreRepo は Firestore リポジトリを初期化します
func NewFirestoreRepo(ctx context.Context, cfg *config.Config) (*FirestoreRepo, error) {
	client, err := firestore.NewClient(ctx, cfg.FirestoreProjectID)
	if err != nil {
		return nil, fmt.Errorf("firestore: クライアント初期化失敗: %w", err)
	}

	return &FirestoreRepo{
		cli:         client,
		tenantsCol:  cfg.CollectionTenants,
		mentionsCol: cfg.CollectionMentions,
	}, nil
}

// ===== MentionRepository 実装 =====

// Save はメンション監視対象を保存します（新規作成または上書き）
func (repo *FirestoreRepo) Save(ctx context.Context, m *domain.Mention) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("firestore: Save検証失敗: %w", err)
	}

	docID := mentionDocID(m.TeamID, m.ChannelID, m.MessageTS, m.MentionedUserID)
	docRef := repo.cli.Collection(repo.mentionsCol).Doc(docID)

	// Firestore保存用のマップ
	data := map[string]interface{}{
		"team_id":           m.TeamID,
		"channel_id":        m.ChannelID,
		"message_ts":        m.MessageTS,
		"mentioned_user_id": m.MentionedUserID,
		"created_at":        m.CreatedAt,
		"reminded":          m.Reminded,
		"escalated":         m.Escalated,
	}

	if _, err := docRef.Set(ctx, data, firestore.MergeAll); err != nil {
		return fmt.Errorf("firestore: メンション保存失敗 (docID=%s): %w", docID, err)
	}

	return nil
}

// Find は指定キーのメンション監視対象を取得します
func (repo *FirestoreRepo) Find(ctx context.Context, teamID, channelID, messageTS, userID string) (*domain.Mention, error) {
	docID := mentionDocID(teamID, channelID, messageTS, userID)
	docRef := repo.cli.Collection(repo.mentionsCol).Doc(docID)

	snapshot, err := docRef.Get(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("firestore: メンション取得失敗 (docID=%s): %w", docID, err)
	}

	// Firestore ドキュメントから domain.Mention へ写経
	var m domain.Mention
	if err := snapshot.DataTo(&m); err != nil {
		return nil, fmt.Errorf("firestore: メンション構造体変換失敗: %w", err)
	}

	return &m, nil
}

// MarkReminded は10分後リマインド完了フラグを立てます
func (repo *FirestoreRepo) MarkReminded(ctx context.Context, teamID, channelID, messageTS, userID string) error {
	docID := mentionDocID(teamID, channelID, messageTS, userID)
	docRef := repo.cli.Collection(repo.mentionsCol).Doc(docID)

	// 単一フィールド更新
	_, err := docRef.Update(ctx, []firestore.Update{
		{Path: "reminded", Value: true},
	})
	if err != nil {
		if isNotFound(err) {
			// ドキュメントが存在しない場合は ErrNotFound を返す
			return domain.ErrNotFound
		}
		return fmt.Errorf("firestore: Reminded フラグ更新失敗 (docID=%s): %w", docID, err)
	}

	return nil
}

// MarkEscalated は30分後エスカレーション完了フラグを立てます
func (repo *FirestoreRepo) MarkEscalated(ctx context.Context, teamID, channelID, messageTS, userID string) error {
	docID := mentionDocID(teamID, channelID, messageTS, userID)
	docRef := repo.cli.Collection(repo.mentionsCol).Doc(docID)

	// 単一フィールド更新
	_, err := docRef.Update(ctx, []firestore.Update{
		{Path: "escalated", Value: true},
	})
	if err != nil {
		if isNotFound(err) {
			// ドキュメントが存在しない場合は ErrNotFound を返す
			return domain.ErrNotFound
		}
		return fmt.Errorf("firestore: Escalated フラグ更新失敗 (docID=%s): %w", docID, err)
	}

	return nil
}

// ===== TenantRepository 実装 =====

// Get はテナント設定を取得します
func (repo *FirestoreRepo) Get(ctx context.Context, teamID string) (*domain.Tenant, error) {
	docID := tenantDocID(teamID)
	docRef := repo.cli.Collection(repo.tenantsCol).Doc(docID)

	snapshot, err := docRef.Get(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("firestore: テナント取得失敗 (docID=%s): %w", docID, err)
	}

	// Firestore ドキュメントから domain.Tenant へ写経
	var t domain.Tenant
	if err := snapshot.DataTo(&t); err != nil {
		return nil, fmt.Errorf("firestore: テナント構造体変換失敗: %w", err)
	}

	return &t, nil
}

// UpsertBotTokenSecret は Botトークンシークレット名を保存します
func (repo *FirestoreRepo) UpsertBotTokenSecret(ctx context.Context, teamID, secretName string) error {
	docID := tenantDocID(teamID)
	docRef := repo.cli.Collection(repo.tenantsCol).Doc(docID)

	// 既存レコードを取得（CreatedAt を保持するため）
	snapshot, err := docRef.Get(ctx)
	createdAt := time.Now().Unix()
	if err == nil {
		// 既存レコードが存在する場合は CreatedAt を保持
		var existing domain.Tenant
		if err := snapshot.DataTo(&existing); err == nil && existing.CreatedAt > 0 {
			createdAt = existing.CreatedAt
		}
	}

	// 新規・更新データ
	data := map[string]interface{}{
		"team_id":               teamID,
		"bot_token_secret_name": secretName,
		"created_at":            createdAt,
	}

	if _, err := docRef.Set(ctx, data, firestore.MergeAll); err != nil {
		return fmt.Errorf("firestore: ボットトークン保存失敗 (docID=%s): %w", docID, err)
	}

	return nil
}

// SetManager は上長ユーザーIDを設定します
func (repo *FirestoreRepo) SetManager(ctx context.Context, teamID string, managerUserID *string) error {
	docID := tenantDocID(teamID)
	docRef := repo.cli.Collection(repo.tenantsCol).Doc(docID)

	// 既存レコードを確認（存在しない場合はエラー）
	_, err := docRef.Get(ctx)
	if err != nil {
		if isNotFound(err) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("firestore: テナント確認失敗 (docID=%s): %w", docID, err)
	}

	// 上長IDを更新
	// nil の場合はフィールドを削除、値がある場合は更新
	if managerUserID == nil {
		// フィールド削除
		_, err = docRef.Update(ctx, []firestore.Update{
			{Path: "manager_user_id", Value: firestore.Delete},
		})
	} else {
		// フィールド更新
		_, err = docRef.Update(ctx, []firestore.Update{
			{Path: "manager_user_id", Value: *managerUserID},
		})
	}

	if err != nil {
		if isNotFound(err) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("firestore: 上長設定失敗 (docID=%s): %w", docID, err)
	}

	return nil
}

// Close は Firestore クライアントを閉じます
func (repo *FirestoreRepo) Close() error {
	if repo.cli != nil {
		return repo.cli.Close()
	}
	return nil
}

// ===== ヘルパー関数 =====

// mentionDocID はメンション監視対象のドキュメントID（一意キー）を生成します
// 形式: "team:channel:ts:user"
func mentionDocID(team, channel, ts, user string) string {
	return fmt.Sprintf("%s:%s:%s:%s", team, channel, ts, user)
}

// tenantDocID はテナント設定のドキュメントID を生成します
// 形式: "team"
func tenantDocID(team string) string {
	return team
}
