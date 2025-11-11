package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"slack-bot/project/service"
)

// RemindHandler は 10分後のリマインド処理を行います
type RemindHandler struct {
	reminderService service.ReminderService
}

// NewRemindHandler はリマインドハンドラーを作成します
func NewRemindHandler(reminderService service.ReminderService) *RemindHandler {
	return &RemindHandler{
		reminderService: reminderService,
	}
}

// ServeHTTP は /check/remind エンドポイント
func (h *RemindHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// リクエスト本体を読み込む
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "リクエスト本体の読み込み失敗", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// JSON パース
	var payload service.TaskPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "JSON パース失敗", http.StatusBadRequest)
		return
	}

	// service.CheckRemind 実行
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.reminderService.CheckRemind(ctx, &payload); err != nil {
		fmt.Printf("リマインド処理エラー: %v\n", err)
		// Cloud Tasks 側へは 200 で応答（再試行回避）
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
