package dto

// SlackCommandRequest は Slack スラッシュコマンドのリクエストを表します
type SlackCommandRequest struct {
	Token        string `form:"token"`
	TeamID       string `form:"team_id"`
	TeamDomain   string `form:"team_domain"`
	ChannelID    string `form:"channel_id"`
	ChannelName  string `form:"channel_name"`
	UserID       string `form:"user_id"`
	UserName     string `form:"user_name"`
	Command      string `form:"command"`      // コマンド名 (/_set_manager など)
	Text         string `form:"text"`         // コマンド引数
	ResponseURL  string `form:"response_url"` // レスポンス URL（遅延応答用）
	TriggerID    string `form:"trigger_id"`
	APIAppID     string `form:"api_app_id"`
	EnterpriseID string `form:"enterprise_id,omitempty"`
}

// SlackSlashResponse はスラッシュコマンドのレスポンスです
type SlackSlashResponse struct {
	ResponseType string        `json:"response_type"` // "in_channel" or "ephemeral"
	Text         string        `json:"text"`
	Blocks       []interface{} `json:"blocks,omitempty"` // Block Kit 形式
}
