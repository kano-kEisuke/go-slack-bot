package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"slack-bot/project/domain"
	"slack-bot/project/infrastructure/config"
)

// ReminderService はメンション監視とリマインド通知を管理するサービスです
type ReminderService interface {
	// OnMention はメンション検知時に呼ばれ、監視レコードを保存し、定期チェックタスクをキューに登録します
	OnMention(ctx context.Context, ev *MentionEvent) error

	// CheckRemind は10分後の定期チェックで呼ばれ、返信がなければリマインドを送信します
	CheckRemind(ctx context.Context, p *TaskPayload) error

	// CheckEscalate は30分後の定期チェックで呼ばれ、返信がなければ再通知と上長DMを送信します
	CheckEscalate(ctx context.Context, p *TaskPayload) error
}

// reminderService は ReminderService の実装です
type reminderService struct {
	cfg *config.Config
	mr  domain.MentionRepository
	tr  domain.TenantRepository
	sp  SlackPort
	tp  TaskPort
}

// NewReminderService は ReminderService のインスタンスを作成します
func NewReminderService(
	cfg *config.Config,
	mr domain.MentionRepository,
	tr domain.TenantRepository,
	sp SlackPort,
	tp TaskPort,
) ReminderService {
	return &reminderService{
		cfg: cfg,
		mr:  mr,
		tr:  tr,
		sp:  sp,
		tp:  tp,
	}
}

// OnMention はメンション検知時に監視レコード保存とタスク予約を行います
func (rs *reminderService) OnMention(ctx context.Context, ev *MentionEvent) error {
	// メンション対象者を抽出
	mentionedUserIDs := parseMentionedUserIDs(ev.Text, ev.BotUserID)
	if len(mentionedUserIDs) == 0 {
		return nil // Bot以外にメンション対象がないためスキップ
	}

	// 各メンション対象者について監視レコード作成とタスク予約
	for _, userID := range mentionedUserIDs {
		// ドメインエンティティ作成
		m := &domain.Mention{
			TeamID:          ev.TeamID,
			ChannelID:       ev.ChannelID,
			MessageTS:       ev.MessageTS,
			MentionedUserID: userID,
			CreatedAt:       ev.NowUnix,
			Reminded:        false,
			Escalated:       false,
		}

		// バリデーション
		if err := m.Validate(); err != nil {
			return fmt.Errorf("OnMention: メンション検証失敗: %w", err)
		}

		// Firestore保存
		if err := rs.mr.Save(ctx, m); err != nil {
			if errors.Is(err, domain.ErrInvalid) {
				return fmt.Errorf("OnMention: メンション保存バリデーション失敗: %w", err)
			}
			return fmt.Errorf("OnMention: メンション保存失敗: %w", err)
		}

		// タスクペイロード
		payload := &TaskPayload{
			TeamID:    ev.TeamID,
			ChannelID: ev.ChannelID,
			MessageTS: ev.MessageTS,
			UserID:    userID,
		}

		// 実行時刻計算
		t0 := time.Unix(ev.NowUnix, 0)
		runAt10 := t0.Add(rs.cfg.RemindDuration)
		runAt30 := t0.Add(rs.cfg.EscalateDuration)

		// 10分後リマインドタスク登録
		if err := rs.tp.EnqueueRemind(ctx, runAt10.Unix(), payload); err != nil {
			return fmt.Errorf("OnMention: 10分後リマインドタスク登録失敗: %w", err)
		}

		// 30分後エスカレーションタスク登録
		if err := rs.tp.EnqueueEscalate(ctx, runAt30.Unix(), payload); err != nil {
			return fmt.Errorf("OnMention: 30分後エスカレーションタスク登録失敗: %w", err)
		}
	}

	return nil
}

// CheckRemind は10分後のチェックで返信がなければリマインドを送信します
func (rs *reminderService) CheckRemind(ctx context.Context, p *TaskPayload) error {
	// 監視レコード取得
	m, err := rs.mr.Find(ctx, p.TeamID, p.ChannelID, p.MessageTS, p.UserID)
	if err != nil {
		if err == domain.ErrMentionNotFound {
			// 古いタスクなのでスキップ
			return nil
		}
		return fmt.Errorf("CheckRemind: メンション取得失敗: %w", err)
	}

	// すでにリマインド済みなら冪等性保証
	if m.Reminded {
		return nil
	}

	// 返信確認
	replied, err := rs.sp.HasUserReplied(ctx, p.TeamID, p.ChannelID, p.MessageTS, p.UserID, p.MessageTS)
	if err != nil {
		return fmt.Errorf("CheckRemind: 返信判定失敗: %w", err)
	}
	if replied {
		// すでに返信済み
		return nil
	}

	// リマインドメッセージ投稿
	text := fmt.Sprintf("<@%s> さん、お手すきの際にご返信お願いします🙏（自動リマインド）", p.UserID)
	if err := rs.sp.PostThreadMessage(ctx, p.TeamID, p.ChannelID, p.MessageTS, text); err != nil {
		return fmt.Errorf("CheckRemind: リマインドメッセージ投稿失敗: %w", err)
	}

	// リマインド完了フラグ更新
	if err := rs.mr.MarkReminded(ctx, p.TeamID, p.ChannelID, p.MessageTS, p.UserID); err != nil {
		if err == domain.ErrMentionNotFound {
			// 既に削除されているため無視
			return nil
		}
		return fmt.Errorf("CheckRemind: リマインドフラグ更新失敗: %w", err)
	}

	return nil
}

// CheckEscalate は30分後のチェックで返信がなければ再通知と上長DMを送信します
func (rs *reminderService) CheckEscalate(ctx context.Context, p *TaskPayload) error {
	// 監視レコード取得
	m, err := rs.mr.Find(ctx, p.TeamID, p.ChannelID, p.MessageTS, p.UserID)
	if err != nil {
		if err == domain.ErrMentionNotFound {
			// 古いタスクなのでスキップ
			return nil
		}
		return fmt.Errorf("CheckEscalate: メンション取得失敗: %w", err)
	}

	// すでにエスカレート済みなら冪等性保証
	if m.Escalated {
		return nil
	}

	// 返信確認
	replied, err := rs.sp.HasUserReplied(ctx, p.TeamID, p.ChannelID, p.MessageTS, p.UserID, p.MessageTS)
	if err != nil {
		return fmt.Errorf("CheckEscalate: 返信判定失敗: %w", err)
	}
	if replied {
		// すでに返信済み
		return nil
	}

	// 30分再通知（スレッド投稿）
	text30 := fmt.Sprintf("<@%s> さん、まだ未返信のようです。目安だけでもご共有ください🙏（自動リマインド）", p.UserID)
	if err := rs.sp.PostThreadMessage(ctx, p.TeamID, p.ChannelID, p.MessageTS, text30); err != nil {
		return fmt.Errorf("CheckEscalate: 30分再通知投稿失敗: %w", err)
	}

	// 上長取得と上長DM送信
	tenant, err := rs.tr.Get(ctx, p.TeamID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// テナント未設定なため上長DMはスキップ（エラーにしない）
		} else {
			return fmt.Errorf("CheckEscalate: テナント取得失敗: %w", err)
		}
	} else if tenant.ManagerUserID != nil {
		// 上長DM送信
		dmText := fmt.Sprintf(
			"【エスカレーション】<@%s> さんが未返信です。対象スレッド: https://app.slack.com/client/%s/%s/thread/%s",
			p.UserID,
			p.TeamID,
			p.ChannelID,
			p.MessageTS,
		)
		if err := rs.sp.PostDM(ctx, p.TeamID, *tenant.ManagerUserID, dmText); err != nil {
			return fmt.Errorf("CheckEscalate: 上長DM送信失敗: %w", err)
		}
	}

	// エスカレート完了フラグ更新
	if err := rs.mr.MarkEscalated(ctx, p.TeamID, p.ChannelID, p.MessageTS, p.UserID); err != nil {
		if err == domain.ErrMentionNotFound {
			// 既に削除されているため無視
			return nil
		}
		return fmt.Errorf("CheckEscalate: エスカレートフラグ更新失敗: %w", err)
	}

	return nil
}

// parseMentionedUserIDs はテキストからSlackメンション（<@USERID>形式）を抽出し、
// BotUserIDを除外したユーザーID一覧を返します
func parseMentionedUserIDs(text, botUserID string) []string {
	// <@USERID> 形式のメンションを抽出
	re := regexp.MustCompile(`<@([A-Z0-9]+)>`)
	matches := re.FindAllStringSubmatch(text, -1)

	// 重複除去とBot除外用のmap
	seen := make(map[string]bool)
	var result []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		userID := match[1]

		// Bot除外
		if userID == botUserID {
			continue
		}

		// 重複除去（最初に出現した順を保持）
		if !seen[userID] {
			seen[userID] = true
			result = append(result, userID)
		}
	}

	return result
}
