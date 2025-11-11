package httpsec

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// VerifySlackSignature は Slack からのリクエストの署名を検証します
// リクエストの X-Slack-Signature ヘッダと X-Slack-Request-Timestamp ヘッダを確認し、
// 改ざんやリプレイ攻撃から保護します
func VerifySlackSignature(signingSecret, signature, timestamp, body string) error {
	// タイムスタンプの検証（5分以内）
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	now := time.Now().Unix()
	if abs(now-ts) > 300 { // 5分 = 300秒
		return fmt.Errorf("request timestamp too old: now=%d, ts=%d", now, ts)
	}

	// 署名の検証
	// Slack署名: "v0=<hash>"
	// hash = HMAC-SHA256("v0:<timestamp>:<body>", signingSecret)
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, body)
	expectedSignature := computeSignature(signingSecret, baseString)

	// 定時間比較（タイミング攻撃対策）
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// computeSignature は Slack 署名を計算します
func computeSignature(signingSecret, baseString string) string {
	h := hmac.New(sha256.New, []byte(signingSecret))
	h.Write([]byte(baseString))
	hash := h.Sum(nil)
	// 16進数文字列に変換して "v0=" プレフィックスを付与
	return fmt.Sprintf("v0=%x", hash)
}

// abs は絶対値を計算します
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// ExtractSignatureFromHeader は "v0=..." 形式の署名文字列から "v0=..." 部分を抽出します
// Slack から送られる X-Slack-Signature ヘッダは "v0=abc123..." の形式です
func ExtractSignatureFromHeader(headerValue string) string {
	// "v0=" で始まっていることを確認
	if strings.HasPrefix(headerValue, "v0=") {
		return headerValue
	}
	return ""
}
