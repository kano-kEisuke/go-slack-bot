# Cloud Run デプロイ手順

## 前提条件

- Google Cloud CLI (`gcloud`) がインストール済み
- Docker がインストール済み
- GCP プロジェクトに対する適切な権限がある

## 環境変数設定

```bash
# GCP プロジェクト設定
export PROJECT_ID="your-gcp-project-id"
export REGION="asia-northeast1"
export SERVICE_NAME="slack-reminder-bot"

# 認証
gcloud auth login
gcloud config set project $PROJECT_ID
```

## デプロイ手順

### 1. ローカルでビルドとテスト（オプション）

```bash
# ビルド
docker build -t slack-reminder-bot:latest .

# ローカルテスト
docker run -p 8080:8080 \
  -e GOOGLE_APPLICATION_CREDENTIALS=/tmp/credentials.json \
  -v ~/.config/gcloud/application_default_credentials.json:/tmp/credentials.json:ro \
  slack-reminder-bot:latest

# ヘルスチェック
curl http://localhost:8080/health
```

### 2. Container Registry に イメージ をプッシュ

```bash
# イメージのタグ付け
docker tag slack-reminder-bot:latest gcr.io/$PROJECT_ID/$SERVICE_NAME:latest

# 認証
gcloud auth configure-docker

# プッシュ
docker push gcr.io/$PROJECT_ID/$SERVICE_NAME:latest
```

### 3. Cloud Run にデプロイ

```bash
gcloud run deploy $SERVICE_NAME \
  --image gcr.io/$PROJECT_ID/$SERVICE_NAME:latest \
  --region $REGION \
  --platform managed \
  --allow-unauthenticated \
  --memory 512Mi \
  --cpu 1 \
  --timeout 3600s \
  --max-instances 100 \
  --set-env-vars="GCP_PROJECT=$PROJECT_ID,FIRESTORE_DB=slack-reminder"
```

### 環境変数の説明

| 変数名 | 説明 | 例 |
|-------|------|-----|
| `GCP_PROJECT` | GCP プロジェクト ID | `my-project-123` |
| `FIRESTORE_DB` | Firestore データベース ID | `slack-reminder` |
| `SLACK_SIGNING_SECRET` | Slack Signing Secret | (Secret Manager から自動取得) |
| `SLACK_BOT_TOKEN` | Slack Bot Token | (Secret Manager から自動取得) |

**注:** Slack の認証情報は Secret Manager で管理するため、デプロイ時に明示的に設定する必要はありません。

### 4. Secret Manager の設定（初回のみ）

Slack 認証情報を GCP Secret Manager に登録：

```bash
# Slack Bot Token を登録
echo -n "xoxb-your-bot-token" | gcloud secrets create slack-bot-token --data-file=-

# Slack Signing Secret を登録
echo -n "your-signing-secret" | gcloud secrets create slack-signing-secret --data-file=-

# Cloud Run サービスに Secret Manager アクセス権を付与
gcloud run services update $SERVICE_NAME \
  --region $REGION \
  --set-env-vars="SLACK_BOT_TOKEN_SECRET_NAME=slack-bot-token,SLACK_SIGNING_SECRET_NAME=slack-signing-secret"
```

### 5. デプロイ状態確認

```bash
# デプロイ状態確認
gcloud run services describe $SERVICE_NAME --region $REGION

# ログ確認
gcloud run services logs read $SERVICE_NAME --region $REGION --limit 50

# リアルタイムログ
gcloud alpha run services logs read $SERVICE_NAME --region $REGION --limit 50 --follow
```

### 6. ヘルスチェック

```bash
# デプロイ後の URL を取得
SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --region $REGION --format='value(status.url)')

# ヘルスチェック
curl $SERVICE_URL/health
```

## トラブルシューティング

### ビルドエラー

```bash
# ビルドキャッシュをクリア
docker system prune -a

# 再度ビルド
docker build --no-cache -t slack-reminder-bot:latest .
```

### デプロイ権限エラー

```bash
# Cloud Run への権限を確認
gcloud projects get-iam-policy $PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.role:roles/run.invoker"

# 必要に応じて権限を付与
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member=serviceAccount:your-service-account@$PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/run.invoker
```

### Firestore 接続エラー

```bash
# Firestore API が有効になっているか確認
gcloud services list --enabled | grep firestore

# 有効にする
gcloud services enable firestore.googleapis.com
```

## ローリングアップデート（本番環境）

既存のサービスを新しいバージョンに更新する場合：

```bash
# イメージをプッシュ
docker tag slack-reminder-bot:latest gcr.io/$PROJECT_ID/$SERVICE_NAME:v2
docker push gcr.io/$PROJECT_ID/$SERVICE_NAME:v2

# 新バージョンでデプロイ（トラフィック移行ステップも設定可能）
gcloud run deploy $SERVICE_NAME \
  --image gcr.io/$PROJECT_ID/$SERVICE_NAME:v2 \
  --region $REGION \
  --no-traffic  # 新しいリビジョンにトラフィックを移行しない

# トラフィック確認後、移行
gcloud run services update-traffic $SERVICE_NAME \
  --to-revisions LATEST=100 \
  --region $REGION
```

## スケーリング設定

```bash
# 同時実行の最大数を調整
gcloud run services update $SERVICE_NAME \
  --region $REGION \
  --concurrency 80

# インスタンス数を調整
gcloud run services update $SERVICE_NAME \
  --region $REGION \
  --min-instances 1 \
  --max-instances 100
```

## 削除（本番環境の場合は慎重に）

```bash
# サービスを削除
gcloud run services delete $SERVICE_NAME --region $REGION

# イメージを削除
gcloud container images delete gcr.io/$PROJECT_ID/$SERVICE_NAME --quiet
```
