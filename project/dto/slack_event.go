package dto

// SlackEventRequest は Slack Events API のリクエスト全体を表します
type SlackEventRequest struct {
	Token          string               `json:"token"`
	TeamID         string               `json:"team_id"`
	APIAppID       string               `json:"api_app_id"`
	Event          SlackEvent           `json:"event"`
	Type           string               `json:"type"` // "event_callback", "url_verification"
	EventID        string               `json:"event_id"`
	EventTime      int64                `json:"event_time"`
	Challenge      string               `json:"challenge,omitempty"` // URL検証時のみ
	Authorizations []SlackAuthorization `json:"authorizations,omitempty"`
}

// SlackEvent は様々なSlackイベントを表現する汎用構造体です
type SlackEvent struct {
	Type      string `json:"type"`                // "message", "app_mention" など
	User      string `json:"user"`                // イベント発生者（メッセージ送信者）
	Text      string `json:"text"`                // メッセージ本文
	Channel   string `json:"channel"`             // チャンネルID
	Timestamp string `json:"ts"`                  // メッセージTS（親メッセージのts）
	ThreadTs  string `json:"thread_ts,omitempty"` // スレッドTS（スレッド内の場合）
	BotID     string `json:"bot_id,omitempty"`    // Bot投稿の場合
	SubType   string `json:"subtype,omitempty"`   // "bot_message"など

	// app_mention イベント固有
	BotProfile *SlackBotProfile `json:"bot_profile,omitempty"`
}

// SlackBotProfile は Bot ユーザー情報を表します
type SlackBotProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SlackAuthorization は OAuth 認可情報を表します
type SlackAuthorization struct {
	EnterpriseID string `json:"enterprise_id,omitempty"`
	TeamID       string `json:"team_id"`
	UserID       string `json:"user_id"`
	IsBot        bool   `json:"is_bot"`
}

// SlackTokenRequest は Slack OAuth token 交換リクエストのペイロードです
type SlackTokenRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
}

// SlackTokenResponse は Slack OAuth token 交換レスポンスです
type SlackTokenResponse struct {
	OK          bool             `json:"ok"`
	Error       string           `json:"error,omitempty"`
	AccessToken string           `json:"access_token"`
	TokenType   string           `json:"token_type"`
	Scope       string           `json:"scope"`
	BotUserID   string           `json:"bot_user_id"`
	AppID       string           `json:"app_id"`
	Team        SlackTeam        `json:"team"`
	Enterprise  *SlackEnterprise `json:"enterprise,omitempty"`
}

// SlackTeam は Slack ワークスペース情報です
type SlackTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SlackEnterprise は Enterprise Grid の組織情報です
type SlackEnterprise struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
