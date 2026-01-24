package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"slack-bot/project/dto"
	"slack-bot/project/infrastructure/httpsec"
	"slack-bot/project/service"
)

// EventsHandler は Slack Events API からのイベントを処理します
type EventsHandler struct {
	signingSecret   string
	reminderService service.ReminderService
}

// NewEventsHandler はイベントハンドラーを作成します
func NewEventsHandler(signingSecret string, reminderService service.ReminderService) *EventsHandler {
	return &EventsHandler{
		signingSecret:   signingSecret,
		reminderService: reminderService,
	}
}

// ServeHTTP は Slack イベント受信エンドポイントです
func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// リクエスト本体を読み込む
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "リクエスト本体の読み込み失敗", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// まず url_verification かどうかを確認（署名検証の前に）
	var preCheck struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(body, &preCheck); err == nil {
		if preCheck.Type == "url_verification" {
			// URL 検証に応答（署名検証をスキップ）
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(preCheck.Challenge))
			return
		}
	}

	// Slack 署名検証（url_verification 以外のリクエスト）
	signature := r.Header.Get("X-Slack-Signature")
	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	if err := httpsec.VerifySlackSignature(h.signingSecret, signature, timestamp, string(body)); err != nil {
		http.Error(w, "署名検証失敗", http.StatusUnauthorized)
		return
	}

	// JSON パース（完全版）
	var req dto.SlackEventRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "JSON パース失敗", http.StatusBadRequest)
		return
	}

	// event_callback のみ処理
	if req.Type != "event_callback" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// イベント処理
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.handleEvent(ctx, req); err != nil {
		fmt.Printf("イベント処理エラー: %v\n", err)
		// Slack側への応答は成功にして、ログだけ記録
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleEvent は個別のイベントを処理します
func (h *EventsHandler) handleEvent(ctx context.Context, req dto.SlackEventRequest) error {
	// app_mention イベント (Bot メンション) または message イベント (返信確認用)
	if req.Event.Type != "app_mention" && req.Event.Type != "message" {
		return nil
	}

	// Bot 自身のメッセージや bot_message は無視
	if req.Event.BotID != "" || req.Event.SubType == "bot_message" {
		return nil
	}

	// app_mention の場合のみメンション検知を処理
	if req.Event.Type != "app_mention" {
		return nil
	}

	// メンション検知イベントを service に渡す
	// BotUserID は Authorization から取得
	botUserID := ""
	for _, auth := range req.Authorizations {
		if auth.IsBot {
			botUserID = auth.UserID
			break
		}
	}

	event := service.MentionEvent{
		TeamID:       req.TeamID,
		ChannelID:    req.Event.Channel,
		MessageTS:    req.Event.Timestamp,
		Text:         req.Event.Text,
		BotUserID:    botUserID,
		ParentUserID: req.Event.User,
		NowUnix:      time.Now().Unix(),
	}

	return h.reminderService.OnMention(ctx, &event)
}
