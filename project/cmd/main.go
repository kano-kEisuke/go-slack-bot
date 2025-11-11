package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"slack-bot/project/handler"
	"slack-bot/project/infrastructure/config"
	"slack-bot/project/infrastructure/secret"
	"slack-bot/project/infrastructure/slack"
	"slack-bot/project/infrastructure/store"
	"slack-bot/project/infrastructure/tasks"
	"slack-bot/project/service"
)

func main() {
	ctx := context.Background()

	// 1. 設定を読み込む
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("設定読み込み失敗: %v", err)
	}

	// 2. 依存関係を初期化
	// Secret Manager
	secretMgr, err := secret.NewManager(ctx, cfg.GcpProject)
	if err != nil {
		log.Fatalf("Secret Manager 初期化失敗: %v", err)
	}
	defer secretMgr.Close()

	// Firestore リポジトリ
	repo, err := store.NewFirestoreRepo(ctx, cfg)
	if err != nil {
		log.Fatalf("Firestore 初期化失敗: %v", err)
	}
	defer repo.Close()

	// Slack API ポート実装
	slackClient := slack.NewSlackClient(secretMgr)

	// Cloud Tasks ポート実装
	tasksClient, err := tasks.NewCloudTasksClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Cloud Tasks クライアント初期化失敗: %v", err)
	}
	defer tasksClient.Close()

	// 3. サービス層を初期化
	reminderService := service.NewReminderService(cfg, repo, repo, slackClient, tasksClient)

	// 4. HTTP ハンドラーを設定
	mux := http.NewServeMux()

	// Slack イベント受信
	mux.Handle("/slack/events", handler.NewEventsHandler(cfg.SlackSigningSecret, reminderService))

	// Slack スラッシュコマンド
	mux.Handle("/slack/commands", handler.NewCommandsHandler(cfg.SlackSigningSecret, repo, slackClient))

	// Cloud Tasks からのコールバック
	mux.Handle("/check/remind", handler.NewRemindHandler(reminderService))
	mux.Handle("/check/escalate", handler.NewEscalateHandler(reminderService))

	// OAuth コールバック
	mux.Handle("/slack/oauth_redirect", handler.NewOAuthHandler(cfg, repo, secretMgr))

	// ヘルスチェック
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// 5. サーバー起動
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("サーバー起動: %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Fatalf("サーバーエラー: %v", err)
	}
}
