package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"slack-bot/project/domain"
	"slack-bot/project/dto"
	"slack-bot/project/infrastructure/httpsec"
)

// CommandsHandler は Slack スラッシュコマンドを処理します
type CommandsHandler struct {
	signingSecret    string
	tenantRepository domain.TenantRepository
	slackPort        SlackPort // ユーザー情報取得用
}

// SlackPort は Slack API 操作の最小インターフェース
type SlackPort interface {
	// GetUserID はユーザーメールアドレスまたはユーザー名から ID を取得
	GetUserID(ctx context.Context, teamID, userNameOrEmail string) (string, error)
}

// NewCommandsHandler はコマンドハンドラーを作成します
func NewCommandsHandler(signingSecret string, tenantRepository domain.TenantRepository, slackPort SlackPort) *CommandsHandler {
	return &CommandsHandler{
		signingSecret:    signingSecret,
		tenantRepository: tenantRepository,
		slackPort:        slackPort,
	}
}

// ServeHTTP は Slack スラッシュコマンド受信エンドポイントです
func (h *CommandsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// body を読み込む（署名検証用）
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"response_type":"ephemeral","text":"リクエスト読み込み失敗"}`)
		return
	}

	// Slack 署名検証
	if err := httpsec.VerifySlackSignature(h.signingSecret,
		r.Header.Get("X-Slack-Signature"),
		r.Header.Get("X-Slack-Request-Timestamp"),
		string(bodyBytes)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"response_type":"ephemeral","text":"署名検証失敗"}`)
		return
	}

	// form パース（bodyBytesから再構築）
	values := parseFormFromBytes(bodyBytes)

	var cmd dto.SlackCommandRequest
	cmd.Token = values.Get("token")
	cmd.TeamID = values.Get("team_id")
	cmd.ChannelID = values.Get("channel_id")
	cmd.UserID = values.Get("user_id")
	cmd.Command = values.Get("command")
	cmd.Text = values.Get("text")

	// コマンド実行
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")

	switch cmd.Command {
	case "/_set_manager":
		h.handleSetManager(w, ctx, cmd)
	case "/_unset_manager":
		h.handleUnsetManager(w, ctx, cmd)
	case "/_get_manager":
		h.handleGetManager(w, ctx, cmd)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"response_type":"ephemeral","text":"不明なコマンド: %s"}`, cmd.Command)
	}
}

// handleSetManager は /_set_manager コマンドを処理
func (h *CommandsHandler) handleSetManager(w http.ResponseWriter, ctx context.Context, cmd dto.SlackCommandRequest) {
	log.Printf("/_set_manager called: TeamID=%s, Text=%s", cmd.TeamID, cmd.Text)

	if cmd.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"response_type":"ephemeral","text":"使用方法: /_set_manager @ユーザー名"}`)
		return
	}

	// @ユーザー名 から ID を抽出
	userRef := strings.TrimPrefix(strings.TrimSpace(cmd.Text), "@")
	log.Printf("GetUserID: TeamID=%s, userRef=%s", cmd.TeamID, userRef)

	userID, err := h.slackPort.GetUserID(ctx, cmd.TeamID, userRef)
	if err != nil {
		log.Printf("GetUserID error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"response_type":"ephemeral","text":"ユーザー検索失敗: %v"}`, err)
		return
	}

	log.Printf("Found userID: %s", userID)

	// Tenant 更新
	if err := h.tenantRepository.SetManager(ctx, cmd.TeamID, &userID); err != nil {
		log.Printf("SetManager error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err == domain.ErrTenantNotRegistered {
			fmt.Fprint(w, `{"response_type":"ephemeral","text":"このワークスペースは登録されていません"}`)
		} else {
			fmt.Fprintf(w, `{"response_type":"ephemeral","text":"上長設定に失敗しました: %v"}`, err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"response_type":"ephemeral","text":"上長を <@%s> に設定しました"}`, userID)
}

// handleUnsetManager は /_unset_manager コマンドを処理
func (h *CommandsHandler) handleUnsetManager(w http.ResponseWriter, ctx context.Context, cmd dto.SlackCommandRequest) {
	// Tenant から上長を削除
	if err := h.tenantRepository.SetManager(ctx, cmd.TeamID, nil); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"response_type":"ephemeral","text":"上長削除失敗"}`)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"response_type":"ephemeral","text":"上長設定を削除しました"}`)
}

// handleGetManager は /_get_manager コマンドを処理
func (h *CommandsHandler) handleGetManager(w http.ResponseWriter, ctx context.Context, cmd dto.SlackCommandRequest) {
	tenant, err := h.tenantRepository.Get(ctx, cmd.TeamID)
	if err != nil {
		if err == domain.ErrTenantNotRegistered {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"response_type":"ephemeral","text":"このワークスペースは登録されていません"}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"response_type":"ephemeral","text":"テナント取得に失敗しました"}`)
		return
	}

	if tenant.ManagerUserID == nil {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"response_type":"ephemeral","text":"上長が設定されていません"}`)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"response_type":"ephemeral","text":"現在の上長: <@%s>"}`, *tenant.ManagerUserID)
}

// parseFormFromBytes はバイト列からURLエンコードされたフォームをパースします
func parseFormFromBytes(b []byte) formValues {
	values := make(formValues)
	for _, pair := range strings.Split(string(b), "&") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key, _ := url.QueryUnescape(parts[0])
			val, _ := url.QueryUnescape(parts[1])
			values[key] = append(values[key], val)
		}
	}
	return values
}

// formValues はurl.Valuesと同じインターフェースを提供
type formValues map[string][]string

func (v formValues) Get(key string) string {
	if vals, ok := v[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}
