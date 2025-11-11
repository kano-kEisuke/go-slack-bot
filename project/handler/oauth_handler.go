package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"slack-bot/project/domain"
	"slack-bot/project/dto"
	"slack-bot/project/infrastructure/config"
	"slack-bot/project/infrastructure/secret"
)

// OAuthHandler は Slack OAuth フロー（インストール完了）を処理します
type OAuthHandler struct {
	cfg              *config.Config
	tenantRepository domain.TenantRepository
	secretManager    *secret.Manager
}

// NewOAuthHandler は OAuth ハンドラーを作成します
func NewOAuthHandler(cfg *config.Config, tenantRepository domain.TenantRepository, secretManager *secret.Manager) *OAuthHandler {
	return &OAuthHandler{
		cfg:              cfg,
		tenantRepository: tenantRepository,
		secretManager:    secretManager,
	}
}

// ServeHTTP は OAuth コールバック処理 (/oauth_redirect)
func (h *OAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// クエリパラメータから code を取得
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "code パラメータが不足しています", http.StatusBadRequest)
		return
	}

	// Slack OAuth token 交換 API を呼び出す
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tokenResp, err := h.exchangeToken(ctx, code)
	if err != nil {
		http.Error(w, fmt.Sprintf("トークン交換失敗: %v", err), http.StatusBadRequest)
		return
	}

	if !tokenResp.OK {
		http.Error(w, fmt.Sprintf("OAuth エラー: %s", tokenResp.Error), http.StatusBadRequest)
		return
	}

	// Secret Manager にトークンを保存
	secretName := fmt.Sprintf("slack_token_%s", tokenResp.Team.ID)
	if err := h.secretManager.PutSecret(ctx, secretName, tokenResp.AccessToken); err != nil {
		http.Error(w, fmt.Sprintf("トークン保存失敗: %v", err), http.StatusInternalServerError)
		return
	}

	// Tenant として登録
	if err := h.tenantRepository.UpsertBotTokenSecret(ctx, tokenResp.Team.ID, secretName); err != nil {
		http.Error(w, fmt.Sprintf("テナント登録失敗: %v", err), http.StatusInternalServerError)
		return
	}

	// インストール成功画面を表示
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>インストール成功</title>
    <style>
        body { font-family: sans-serif; margin: 40px; }
        .success { color: green; font-size: 18px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="success">✓ Slack Reminder Bot がインストールされました！</div>
    <p>チャンネルで @Bot をメンションして、返信監視を開始できます。</p>
    <p>管理者は <code>/_set_manager @上長</code> でエスカレーション先を設定してください。</p>
</body>
</html>
	`))
}

// exchangeToken は OAuth code をトークンに交換します
func (h *OAuthHandler) exchangeToken(ctx context.Context, code string) (*dto.SlackTokenResponse, error) {
	// OAuth API エンドポイント
	url := "https://slack.com/api/oauth.v2.access"

	// リクエストボディ
	reqBody := dto.SlackTokenRequest{
		ClientID:     h.cfg.SlackClientID,
		ClientSecret: h.cfg.SlackClientSecret,
		Code:         code,
		RedirectURI:  h.cfg.OAuthRedirectURL,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("リクエスト JSON 化失敗: %w", err)
	}

	// POST リクエスト
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("リクエスト作成失敗: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("リクエスト送信失敗: %w", err)
	}
	defer resp.Body.Close()

	// レスポンス解析
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンス本体読み込み失敗: %w", err)
	}

	var tokenResp dto.SlackTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("レスポンス JSON 解析失敗: %w", err)
	}

	return &tokenResp, nil
}
