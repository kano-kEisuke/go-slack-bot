# 🚀 Slack Reminder Bot - 完全セットアップガイド

このガイドに従って、段階的にセットアップを進めてください。

---

## 📋 セットアップの全体フロー

```
【フェーズ1】GCP プロジェクト作成 （初回のみ・40分）
    ↓
【フェーズ2】Slack App 作成 （初回のみ・20分）
    ↓
【フェーズ3】環境変数設定 （初回のみ・10分）
    ↓
【フェーズ4】デプロイ実行 （毎回・10分）
```

**所要時間**: 初回 = 1時間20分程度 / 更新時 = 10分

---

## 🎯 フェーズ1: GCP プロジェクト作成（初回のみ）

[`GCP_SETUP.md`](GCP_SETUP.md) を参照してください。

以下を実行：
1. GCP プロジェクト作成
2. 必要な API を有効化（6つ）
3. Firestore データベース作成
4. Cloud Tasks キュー作成（2つ）
5. サービスアカウント作成・権限設定（5つ）
6. **Secret Manager に OAuth State Secret を登録**

**⚠️ 重要**: Slack 認証情報（Signing Secret, Client ID, Client Secret）は、フェーズ2の後に Secret Manager に登録します。

**✅ フェーズ1 完了！**

---

## 🤖 フェーズ2: Slack App 作成（初回のみ）

[`SLACK_SETUP.md`](SLACK_SETUP.md) を参照してください。

以下を取得して、Secret Manager に登録します：
- Signing Secret
- Client ID
- Client Secret

### Secret Manager への登録（フェーズ2の後に実行）

```bash
# Slack Signing Secret を登録
echo -n "your-signing-secret-here" | \
  gcloud secrets create slack-signing-secret --data-file=-

# Slack Client ID を登録
echo -n "your-client-id-here" | \
  gcloud secrets create slack-client-id --data-file=-

# Slack Client Secret を登録
echo -n "your-client-secret-here" | \
  gcloud secrets create slack-client-secret --data-file=-
```

**✅ フェーズ2 完了！**

---

## 📝 フェーズ3: 環境変数設定（初回のみ）

### 3-1. .env ファイルを作成

```bash
cp .env.example .env
```

### 3-2. .env を編集

```bash
nano .env
```

または、テキストエディタで `.env` を開いて、以下の値を入力します：

#### GCP 設定部分

```env
GCP_PROJECT=my-slack-bot-project    # ← GCP プロジェクト ID に変更
REGION=asia-northeast1              # 東京推奨
FIRESTORE_PROJECT_ID=my-slack-bot-project  # ← 同じく GCP プロジェクト ID

FS_COLLECTION_TENANTS=tenants       # そのまま
FS_COLLECTION_MENTIONS=mentions     # そのまま
```

確認方法：
```bash
# GCP プロジェクト ID を確認
gcloud config get-value project
```

#### Slack 設定部分

⚠️ **重要**: Slack認証情報は環境変数ではなく、Secret Managerに保存されます。

`.env`ファイルには以下のダミー値を設定してください（デプロイスクリプトの検証用）:

```env
SLACK_SIGNING_SECRET=from-secret-manager
SLACK_CLIENT_ID=from-secret-manager
SLACK_CLIENT_SECRET=from-secret-manager
OAUTH_STATE_SECRET=from-secret-manager
```

実際の値はSecret Managerから自動で読み込まれます。

#### OAuth Redirect URL

```env
OAUTH_REDIRECT_URL=https://slack-reminder-bot-xxxxx.run.app/slack/oauth_redirect
# ↑ 初回は仮で OK。デプロイ後に実際の URL で上書き
```

#### Cloud Tasks 設定部分

```bash
# 以下のコマンドで各値を確認
gcloud tasks queues list --location=asia-northeast1

# 出力例：
# NAME                   LOCATION            RESPONSE_HANDLER
# remind-queue           asia-northeast1
# escalate-queue         asia-northeast1
```

`.env` に入力：
```env
TASKS_QUEUE_REMIND=projects/my-slack-bot-project/locations/asia-northeast1/queues/remind-queue
TASKS_QUEUE_ESCALATE=projects/my-slack-bot-project/locations/asia-northeast1/queues/escalate-queue
TASKS_AUDIENCE=https://slack-reminder-bot-xxxxx.run.app
# ↑ こちらも初回は仮で OK（デプロイ後に更新）

TASKS_SERVICE_ACCOUNT=slack-bot-service@my-slack-bot-project.iam.gserviceaccount.com
```

確認方法：
```bash
gcloud iam service-accounts list
```

#### タイミング設定

```env
REMIND_AFTER=10m      # 10分後にリマインド
ESCALATE_AFTER=30m    # 30分後にエスカレーション
```

### 3-3. 環境変数を検証

.env を保存した後、スクリプトを実行して環境変数をチェック：

```bash
./deploy.sh
```

すべての環境変数が正しく設定されていれば、デプロイに進みます。

**✅ フェーズ3 完了！**

---

## 🚀 フェーズ4: デプロイ実行（毎回）

### 4-1. デプロイスクリプトを実行

```bash
# GCP プロジェクト ID を指定してデプロイ
./deploy.sh my-slack-bot-project
```

### 4-2. 出力を確認

デプロイが成功すると、以下のように表示されます：

```
✅ デプロイが完了しました！

📍 サービス URL: https://slack-reminder-bot-abc123.run.app

💚 ヘルスチェック: OK

🎯 次のステップ:
  1. Slack App Dashboard で OAUTH_REDIRECT_URL を更新
     URL: https://slack-reminder-bot-abc123.run.app/slack/oauth_redirect
  2. https://api.slack.com/apps にアクセス
  3. OAuth & Permissions → Redirect URLs を更新
  4. .env ファイルの OAUTH_REDIRECT_URL も更新
```

### 4-3. Slack App 設定を更新

1. [Slack App Dashboard](https://api.slack.com/apps) にアクセス
2. 自分のアプリを選択 → **OAuth & Permissions**
3. **Redirect URLs** を編集
4. 新しい URL を追加：`https://slack-reminder-bot-abc123.run.app/slack/oauth_redirect`
5. **変更を保存**

### 4-4. .env ファイルを更新

```bash
# .env を開く
nano .env

# OAUTH_REDIRECT_URL と TASKS_AUDIENCE を実際の URL に変更
OAUTH_REDIRECT_URL=https://slack-reminder-bot-abc123.run.app/slack/oauth_redirect
TASKS_AUDIENCE=https://slack-reminder-bot-abc123.run.app
```

### 4-5. 再度デプロイ（重要！）

```bash
./deploy.sh my-slack-bot-project
```

**✅ デプロイ完了！**

---

## 🔍 デプロイ後の確認

### ヘルスチェック

```bash
SERVICE_URL=$(gcloud run services describe slack-reminder-bot --region asia-northeast1 --format='value(status.url)')
curl $SERVICE_URL/health
# 出力: ok
```

### ログ確認

```bash
gcloud run services logs read slack-reminder-bot --region asia-northeast1 --limit 50
```

### リアルタイムログ

```bash
gcloud alpha run services logs read slack-reminder-bot --region asia-northeast1 --limit 50 --follow
```

---

## 🐛 トラブルシューティング

### エラー: `.env ファイルが見つかりません`

```bash
cp .env.example .env
nano .env  # 値を入力
./deploy.sh
```

### エラー: `環境変数が設定されていません`

`.env` ファイルを確認して、不足している値を入力してください。

```bash
nano .env
```

### エラー: `GCP にログインしていません`

```bash
gcloud auth login
```

### エラー: `docker: command not found`

[Docker Desktop をインストール](https://www.docker.com/products/docker-desktop)

### デプロイは完了したが、ログにエラーが出ている

```bash
# ログを確認
gcloud run services logs read slack-reminder-bot --region asia-northeast1 --limit 50

# よくあるエラー:
# - Secret Manager 権限なし
#   → サービスアカウントに roles/secretmanager.secretAccessor を付与
# - Firestore 接続失敗
#   → GCP_PROJECT と FIRESTORE_PROJECT_ID が同じか確認
# - Cloud Tasks キュー不正
#   → TASKS_QUEUE_REMIND, TASKS_QUEUE_ESCALATE が正しいか確認
```

---

## 📚 次のステップ

1. **Slack App を Workspace にインストール**
   - [`SLACK_SETUP.md`](SLACK_SETUP.md) 参照

2. **Slack でテスト**
   - チャンネルで `@bot @user-name` とメンション
   - 10分後にリマインドが送信されるか確認

3. **本番環境へ**
   - 上長 DM 通知の設定
   - 監視対象チャンネルの確認

---

## 📞 サポート

問題が発生した場合：

1. **ログを確認**
   ```bash
   gcloud run services logs read slack-reminder-bot --region asia-northeast1
   ```

2. **環境変数を確認**
   ```bash
   gcloud run services describe slack-reminder-bot --region asia-northeast1
   ```

3. **詳細は各ガイドを参照**
   - GCP 設定: [`GCP_SETUP.md`](GCP_SETUP.md)
   - Slack 設定: [`SLACK_SETUP.md`](SLACK_SETUP.md)
   - デプロイ詳細: [`DEPLOY.md`](DEPLOY.md)
