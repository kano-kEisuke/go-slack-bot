# 🤖 Slack App セットアップガイド

このガイドに従って、Slack App を作成し、必要な認証情報を取得してください。

---

## 📋 全体フロー

```
1. Slack App を作成
2. 認証情報を取得（Signing Secret, Client ID, Client Secret）
3. イベント購読を設定
4. スラッシュコマンドを設定
5. OAuth スコープを設定
6. Workspace にインストール
```

**所要時間**: 30分程度

---

## 1️⃣ Slack App を作成

### ステップ1: [Slack API Dashboard](https://api.slack.com/apps) にアクセス

1. ブラウザで https://api.slack.com/apps にアクセス
2. **Create New App** ボタンをクリック
3. **From an app manifest** を選択

### ステップ2: Manifest をセットアップ

以下の YAML を貼り付けます：

```yaml
display_information:
  name: Slack Reminder Bot
  description: メンション返信を監視し、返信がない場合にリマインドを送信します

features:
  bot_user:
    display_name: slack-reminder-bot
    always_online: true
  slash_commands:
    - command: /remind-config
      url: https://YOUR_SERVICE_URL/slack/commands
      description: リマインド設定を変更
      usage_hint: "[設定項目]"
  event_subscriptions:
    url: https://YOUR_SERVICE_URL/slack/events
    events:
      - app_mention
      - message

oauth_config:
  scopes:
    bot:
      - chat:write
      - chat:write.public
      - groups:read
      - users:read
      - users:read.email
  redirect_urls:
    - https://YOUR_SERVICE_URL/slack/oauth_redirect

settings:
  interactivity:
    is_enabled: true
    request_url: https://YOUR_SERVICE_URL/slack/events
  bot_tokens_expiration_enabled: false
```

**⚠️ 重要**: `YOUR_SERVICE_URL` を自分のサービス URL に置き換えます（初回は仮で OK）

例：`https://slack-reminder-bot-abc123.run.app`

### ステップ3: App を作成

1. 上記 YAML を貼り付け
2. **Create** ボタンをクリック
3. Workspace を選択
4. **Create App** をクリック

---

## 2️⃣ 認証情報を取得

### 取得が必要な情報

デプロイ時に以下の3つが必要です：

| 項目 | 取得元 | .env での設定 |
|------|--------|--------------|
| **Signing Secret** | Settings → Basic Information | `SLACK_SIGNING_SECRET` |
| **Client ID** | Settings → Basic Information | `SLACK_CLIENT_ID` |
| **Client Secret** | Settings → Basic Information | `SLACK_CLIENT_SECRET` |

### 具体的な手順

1. [Slack API Dashboard](https://api.slack.com/apps) で自分のアプリを選択
2. 左メニューから **Settings** → **Basic Information** をクリック
3. **App Credentials** セクションを見つけます
4. 以下の値をコピー：
   - **Signing Secret**
   - **Client ID**
   - **Client Secret**

5. 別のテキストファイルに一時保存（`.env` 設定時に使用）

### 注意

- **Client Secret** は絶対に GitHub などに公開しないでください
- `.env` ファイルも同様に機密情報なので、`.gitignore` で除外してください

---

## 3️⃣ イベント購読を設定

### 有効化するイベント

1. 左メニューから **Features** → **Event Subscriptions** をクリック
2. **Events** セクションで以下を有効化：
   - `app_mention` - Bot がメンションされたとき
   - `message` - メッセージが投稿されたとき

### リクエスト URL

**Event Subscriptions** の **Request URL** に以下を入力：

```
https://YOUR_SERVICE_URL/slack/events
```

**検証**:
- URL が正しければ `Verified` と表示されます
- エラーが出た場合は、デプロイがまだ完了していない可能性があります

---

## 4️⃣ スラッシュコマンドを設定

### コマンド作成

1. 左メニューから **Features** → **Slash Commands** をクリック
2. **Create New Command** をクリック

#### コマンド1: /remind-config

```
Command: /remind-config
Request URL: https://YOUR_SERVICE_URL/slack/commands
Short Description: リマインド設定を変更
Usage hint: [設定項目]
```

**Save** をクリック

---

## 5️⃣ OAuth スコープを設定

### Bot Token スコープ

1. 左メニューから **Features** → **OAuth & Permissions** をクリック
2. **Scopes** → **Bot Token Scopes** で以下を有効化：

| スコープ | 説明 |
|---------|------|
| `chat:write` | メッセージ投稿 |
| `chat:write.public` | 公開チャンネルへのメッセージ投稿 |
| `groups:read` | DM・グループ情報取得 |
| `users:read` | ユーザー情報取得 |
| `users:read.email` | ユーザーメールアドレス取得 |

### リダイレクト URL

**Redirect URLs** に以下を追加：

```
https://YOUR_SERVICE_URL/slack/oauth_redirect
```

**Save URLs** をクリック

---

## 6️⃣ Workspace にインストール

### インストール手順

1. 左メニューから **Settings** → **Install App** をクリック
2. **Install to Workspace** をクリック
3. 権限の確認画面が表示されます
4. **許可** をクリック

### Bot ユーザーの確認

インストール完了後、以下を確認：

1. Slack Workspace にログイン
2. 左サイドバーで `@slack-reminder-bot` が表示されているか確認
3. DM でテスト: `@slack-reminder-bot hello`

応答があれば、セットアップは成功です！

---

## 📝 よくある設定ミス

### ❌ エラー: "URL verification failed"

**原因**: リクエスト URL が正しくない、またはデプロイがまだ完了していない

**対処**:
1. デプロイが完了しているか確認
2. URL が正しくコピーされているか確認
3. `https://` で始まっているか確認（`http://` ではない）

### ❌ エラー: "Invalid redirect URL"

**原因**: OAuth Redirect URL が `https://` で始まっていない

**対処**:
```
❌ http://slack-reminder-bot-abc.run.app/slack/oauth_redirect
✅ https://slack-reminder-bot-abc.run.app/slack/oauth_redirect
```

### ❌ エラー: "Signing Secret が無効"

**原因**: Signing Secret がコピーミスされている

**対処**:
1. [Slack API Dashboard](https://api.slack.com/apps) で App を選択
2. Settings → Basic Information
3. **Signing Secret** を再度確認してコピー
4. `.env` に貼り付けて上書き

---

## ✅ セットアップ完了チェックリスト

以下を確認したら、セットアップは完了です：

- [ ] Slack App が作成されている
- [ ] Signing Secret をコピーした
- [ ] Client ID をコピーした
- [ ] Client Secret をコピーした
- [ ] Event Subscriptions で `app_mention` と `message` を有効化
- [ ] Request URL が Verified と表示されている
- [ ] OAuth Scopes を設定した
- [ ] Redirect URL を設定した
- [ ] Workspace にインストール済み
- [ ] DM でテストして応答確認

---

## 🚀 次のステップ

1. `.env.example` をコピーして `.env` を作成
2. Slack App から取得した認証情報を入力：
   ```env
   SLACK_SIGNING_SECRET=<コピーした値>
   SLACK_CLIENT_ID=<コピーした値>
   SLACK_CLIENT_SECRET=<コピーした値>
   ```

3. [`SETUP_GUIDE.md`](SETUP_GUIDE.md) に戻ってデプロイを進める

---

## 📞 デバッグ

### ログで Slack イベント受信を確認

```bash
gcloud run services logs read slack-reminder-bot --region asia-northeast1 --limit 50
```

`app_mention` イベントを投稿してから確認すると、ログにイベント受信が記録されます。

### Slack API のテスト

```bash
# スレッドにメッセージ投稿（テスト用）
curl -X POST https://slack.com/api/chat.postMessage \
  -H 'Content-type: application/json' \
  --data '{"channel":"C123456","thread_ts":"1234567890.000001","text":"テスト"}'
```
