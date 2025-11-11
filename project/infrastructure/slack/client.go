package slack

import (
	"context"
	"fmt"

	"slack-bot/project/infrastructure/secret"

	"github.com/slack-go/slack"
)

// SlackClient は service.SlackPort の Slack SDK 実装です
type SlackClient struct {
	secretMgr  *secret.Manager
	tokenCache map[string]*slack.Client // teamID -> SlackClient
}

// NewSlackClient は Slack クライアントを初期化します
func NewSlackClient(secretMgr *secret.Manager) *SlackClient {
	return &SlackClient{
		secretMgr:  secretMgr,
		tokenCache: make(map[string]*slack.Client),
	}
}

// getSlackClient は teamID に対応する Slack API クライアントを取得します
// シークレット名から Slack Bot トークンを取得してクライアントを作成
func (sc *SlackClient) getSlackClient(ctx context.Context, teamID, secretTokenPrefix string) (*slack.Client, error) {
	// キャッシュを確認
	if cli, exists := sc.tokenCache[teamID]; exists {
		return cli, nil
	}

	// Secret Manager からトークンを取得
	secretName := fmt.Sprintf("%s%s", secretTokenPrefix, teamID)
	token, err := sc.secretMgr.GetSecret(ctx, secretName)
	if err != nil {
		return nil, fmt.Errorf("slack: トークン取得失敗 (teamID=%s): %w", teamID, err)
	}

	// Slack クライアント作成
	cli := slack.New(token)

	// キャッシュに保存
	sc.tokenCache[teamID] = cli

	return cli, nil
}

// HasUserReplied は指定ユーザーが返信しているかを判定します
func (sc *SlackClient) HasUserReplied(ctx context.Context, teamID, channelID, messageTS, userID, oldest string) (bool, error) {
	// Slack クライアント取得
	cli, err := sc.getSlackClient(ctx, teamID, "slack_token_")
	if err != nil {
		return false, err
	}

	// conversations.replies で messageTS 以降のメッセージを取得
	messages, _, _, err := cli.GetConversationReplies(
		&slack.GetConversationRepliesParameters{
			ChannelID: channelID,
			Timestamp: messageTS,
			Oldest:    oldest,
		},
	)
	if err != nil {
		return false, fmt.Errorf("slack: 返信確認失敗 (channel=%s, ts=%s): %w", channelID, messageTS, err)
	}

	// メッセージをループして対象ユーザーの投稿を検索
	for _, msg := range messages {
		// 親メッセージ自体は除外
		if msg.Timestamp == messageTS {
			continue
		}
		// 対象ユーザーの投稿があれば true
		if msg.User == userID {
			return true, nil
		}
	}

	// ページング処理（必要に応じて次ページを確認）
	// 簡略版: 最初のページで見つからなければ false

	return false, nil
}

// PostThreadMessage はスレッドにメッセージを投稿します
func (sc *SlackClient) PostThreadMessage(ctx context.Context, teamID, channelID, messageTS, text string) error {
	// Slack クライアント取得
	cli, err := sc.getSlackClient(ctx, teamID, "slack_token_")
	if err != nil {
		return err
	}

	// スレッドにメッセージ投稿
	_, _, err = cli.PostMessageContext(
		ctx,
		channelID,
		slack.MsgOptionText(text, false),
		slack.MsgOptionTS(messageTS),
	)
	if err != nil {
		return fmt.Errorf("slack: スレッドメッセージ投稿失敗 (channel=%s, ts=%s): %w", channelID, messageTS, err)
	}

	return nil
}

// PostDM はユーザーに DM を送信します
func (sc *SlackClient) PostDM(ctx context.Context, teamID, userID, text string) error {
	// Slack クライアント取得
	cli, err := sc.getSlackClient(ctx, teamID, "slack_token_")
	if err != nil {
		return err
	}

	// ユーザーとの DM チャンネルを開く
	// OpenConversation で DM チャンネルを開く
	dmCh, _, _, err := cli.OpenConversation(
		&slack.OpenConversationParameters{
			Users: []string{userID},
		},
	)
	if err != nil {
		return fmt.Errorf("slack: DM チャンネル作成失敗 (user=%s): %w", userID, err)
	}

	// DM を送信
	_, _, err = cli.PostMessageContext(
		ctx,
		dmCh.ID,
		slack.MsgOptionText(text, false),
	)
	if err != nil {
		return fmt.Errorf("slack: DM 送信失敗 (user=%s): %w", userID, err)
	}

	return nil
}

// ClearCache はトークンキャッシュをクリアします（テスト用）
func (sc *SlackClient) ClearCache() {
	sc.tokenCache = make(map[string]*slack.Client)
}
