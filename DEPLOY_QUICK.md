# Cloud Run デプロイ - クイックスタート

このドキュメントは、Slack Reminder Bot を Google Cloud Run にデプロイするための簡潔なガイドです。

## 📋 前提条件

```bash
# 必要なツール
- Google Cloud SDK (gcloud)
- Docker
- Go 1.25.0 (ローカルテスト時)
```

## 🚀 クイックデプロイ（推奨）

提供されているデプロイスクリプトで、自動的にビルド・プッシュ・デプロイを行えます。

```bash
# 基本的な使用方法
./deploy.sh <project-id>

# 例
./deploy.sh my-gcp-project

# リージョン指定する場合
./deploy.sh my-gcp-project asia-northeast1

# サービス名をカスタマイズ
./deploy.sh my-gcp-project asia-northeast1 my-slack-bot
```

**スクリプトが実行する内容：**
1. ✅ GCP 認証確認
2. ✅ プロジェクト設定
3. ✅ Docker イメージビルド
4. ✅ Container Registry にプッシュ
5. ✅ Cloud Run にデプロイ
6. ✅ ヘルスチェック実行

## 🔧 手動デプロイ手順

### 1. GCP プロジェクト設定

```bash
export PROJECT_ID="your-project-id"
export REGION="asia-northeast1"
export SERVICE_NAME="slack-reminder-bot"

gcloud auth login
gcloud config set project $PROJECT_ID
```

### 2. Docker イメージ準備

```bash
# ビルド
docker build -t slack-reminder-bot:latest .

# ローカルテスト（オプション）
docker run -p 8080:8080 \
  -e GCP_PROJECT=$PROJECT_ID \
  slack-reminder-bot:latest

# 別ターミナルでテスト
curl http://localhost:8080/health
```

### 3. Container Registry へプッシュ

```bash
# 認証
gcloud auth configure-docker

# タグ付け
docker tag slack-reminder-bot:latest gcr.io/$PROJECT_ID/$SERVICE_NAME:latest

# プッシュ
docker push gcr.io/$PROJECT_ID/$SERVICE_NAME:latest
```

### 4. Cloud Run へデプロイ

```bash
gcloud run deploy $SERVICE_NAME \
  --image gcr.io/$PROJECT_ID/$SERVICE_NAME:latest \
  --region $REGION \
  --platform managed \
  --allow-unauthenticated \
  --memory 512Mi \
  --cpu 1 \
  --timeout 3600s \
  --max-instances 100
```

## 🔐 Secret Manager 設定（初回のみ）

Slack の認証情報を登録します：

```bash
# Slack Bot Token
echo -n "xoxb-..." | gcloud secrets create slack-bot-token --data-file=-

# Slack Signing Secret
echo -n "your-signing-secret" | gcloud secrets create slack-signing-secret --data-file=-
```

## ✅ デプロイ確認

```bash
# サービス状態確認
gcloud run services describe $SERVICE_NAME --region $REGION

# サービス URL 取得
gcloud run services describe $SERVICE_NAME --region $REGION --format='value(status.url)'

# ヘルスチェック
SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --region $REGION --format='value(status.url)')
curl $SERVICE_URL/health

# ログ確認
gcloud run services logs read $SERVICE_NAME --region $REGION --limit 50

# リアルタイムログ
gcloud alpha run services logs read $SERVICE_NAME --region $REGION --limit 50 --follow
```

## 📊 リソース設定

| 設定項目 | 値 | 説明 |
|--------|-----|------|
| メモリ | 512Mi | 標準的なワークロード用 |
| CPU | 1 | 1 vCPU |
| タイムアウト | 3600s | 1時間（Cloud Tasks 用） |
| 最大インスタンス | 100 | スケーリング上限 |
| 同時実行 | デフォルト | 接続ごとに新しいリクエストを処理 |

## 🔄 更新手順

新しいバージョンをデプロイする場合：

```bash
# コードを修正後、再度スクリプトを実行
./deploy.sh $PROJECT_ID

# または手動でデプロイ
docker build -t slack-reminder-bot:v2 .
docker tag slack-reminder-bot:v2 gcr.io/$PROJECT_ID/$SERVICE_NAME:v2
docker push gcr.io/$PROJECT_ID/$SERVICE_NAME:v2

gcloud run deploy $SERVICE_NAME \
  --image gcr.io/$PROJECT_ID/$SERVICE_NAME:v2 \
  --region $REGION
```

## 📍 エンドポイント

デプロイ後、以下のエンドポイントが利用可能です：

- **ヘルスチェック**: `{SERVICE_URL}/health`
- **Slack イベント**: `{SERVICE_URL}/slack/events`
- **Slack コマンド**: `{SERVICE_URL}/slack/commands`
- **OAuth コールバック**: `{SERVICE_URL}/slack/oauth_redirect`
- **10分リマインド**: `{SERVICE_URL}/check/remind`
- **30分エスカレーション**: `{SERVICE_URL}/check/escalate`

## 🐛 トラブルシューティング

### ビルドエラー

```bash
# キャッシュをクリア
docker system prune -a
docker build --no-cache -t slack-reminder-bot:latest .
```

### デプロイ権限エラー

```bash
# 現在のユーザーの権限確認
gcloud projects get-iam-policy $PROJECT_ID --flatten="bindings[].members" --format="table(bindings.role)" --filter="bindings.members:$(gcloud config get-value account)"

# 必要な権限: roles/run.admin, roles/compute.admin
```

### 接続エラー

```bash
# Cloud Tasks API が有効か確認
gcloud services list --enabled | grep cloudtasks

# 有効にする
gcloud services enable cloudtasks.googleapis.com

# Firestore API 有効化
gcloud services enable firestore.googleapis.com
```

### ログに "permission denied" エラーが出ている場合

```bash
# Cloud Run サービスアカウントに権限を付与
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:$PROJECT_ID@appspot.gserviceaccount.com" \
  --role="roles/cloudtasks.taskRunner"
```

## 📚 詳細ドキュメント

より詳しい情報は `DEPLOY.md` を参照してください。

## 🗑️ クリーンアップ

テスト後にリソースを削除する場合：

```bash
# Cloud Run サービス削除
gcloud run services delete $SERVICE_NAME --region $REGION

# Container Registry のイメージ削除
gcloud container images delete gcr.io/$PROJECT_ID/$SERVICE_NAME --quiet

# Secret Manager のシークレット削除
gcloud secrets delete slack-bot-token slack-signing-secret
```

---

**注意**: 本番環境での削除は慎重に行ってください。
