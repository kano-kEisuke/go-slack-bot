package slack

import (
	"context"
	"fmt"
	"strings"

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

// HasUserReplied は指定ユーザーが返信元ユーザーへメンション付きで返信しているかを判定します
// 返信完了の条件: 対象ユーザー(userID)が送信元ユーザー(mentionerUserID)へ @メンション をつけて返信している
func (sc *SlackClient) HasUserReplied(ctx context.Context, teamID, channelID, messageTS, userID, oldest string) (bool, error) {
	// この新しいメソッドは ParentUserID が必要になるため、内部実装は以下の通り
	// 呼び出し側が parentUserID を持っていない場合は以下の実装のままにする
	return sc.HasUserRepliedWithMention(ctx, teamID, channelID, messageTS, userID, "", oldest)
}

// HasUserRepliedWithMention は対象ユーザーが送信元ユーザーへメンション付きで返信しているか判定します
// userID: チェック対象のユーザー（メンションされた人）
// parentUserID: トリガーメッセージ送信者のユーザーID（メンションした人）
func (sc *SlackClient) HasUserRepliedWithMention(ctx context.Context, teamID, channelID, messageTS, userID, parentUserID, oldest string) (bool, error) {
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

	// メッセージをループして対象ユーザーのメンション返信を検索
	for _, msg := range messages {
		// 親メッセージ自体は除外
		if msg.Timestamp == messageTS {
			continue
		}

		// 対象ユーザーが投稿している場合のみチェック
		if msg.User == userID {
			// parentUserID が指定されている場合は、メンション返信を確認
			if parentUserID != "" {
				// userID が parentUserID へメンション (@ユーザーA) をつけているか確認
				if hasMentionToUser(msg.Text, parentUserID) {
					return true, nil // メンション返信を発見
				}
				// parentUserID へのメンションがない場合は、返信と判定しない
				continue
			} else {
				// parentUserID が指定されていない場合は、単純な投稿で判定
				return true, nil
			}
		}
	}

	// ページング処理（必要に応じて次ページを確認）
	// 簡略版: 最初のページで見つからなければ false

	return false, nil
}

// hasMentionToUser は text 内に特定ユーザーへの @メンション があるか判定します
func hasMentionToUser(text string, userID string) bool {
	if text == "" || userID == "" {
		return false
	}

	// Slack メンション形式: <@USERID>
	mentionPattern := fmt.Sprintf("<@%s>", userID)
	return strings.Contains(text, mentionPattern)
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

// GetUserID はユーザー名またはメールアドレスからユーザー ID を取得します
func (sc *SlackClient) GetUserID(ctx context.Context, teamID, userNameOrEmail string) (string, error) {
	// Slack クライアント取得
	cli, err := sc.getSlackClient(ctx, teamID, "slack_token_")
	if err != nil {
		return "", fmt.Errorf("slack: クライアント取得失敗: %w", err)
	}

	// ユーザー名で検索（@ を除去）
	userName := userNameOrEmail
	if len(userName) > 0 && userName[0] == '@' {
		userName = userName[1:]
	}

	// users.list を使ってユーザー名から ID を取得
	users, err := cli.GetUsersContext(ctx)
	if err != nil {
		return "", fmt.Errorf("slack: ユーザー一覧取得失敗: %w", err)
	}

	for _, u := range users {
		if u.Name == userName || u.RealName == userName || u.Profile.Email == userNameOrEmail {
			return u.ID, nil
		}
	}

	return "", fmt.Errorf("slack: ユーザーが見つかりません: %s", userNameOrEmail)
}
