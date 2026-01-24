#!/bin/bash

set -e

# 色出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# .env から環境変数を読み込む
if [ -f .env ]; then
  export $(cat .env | grep -v '^#' | xargs)
fi

echo "📋 .env ファイルから設定を読み込み中..."
echo "📦 Cloud Run デプロイスクリプト"
echo "========================================"
echo "プロジェクト ID: $GCP_PROJECT"
echo "リージョン: $REGION"
echo "サービス名: slack-reminder-bot"
echo "========================================"
echo ""

# 環境変数の検証（Secret Manager使用のため、Slackシークレットは除外）
echo "🔍 環境変数を検証中..."
REQUIRED_VARS=(
  "GCP_PROJECT"
  "REGION"
  "FIRESTORE_PROJECT_ID"
  "FS_COLLECTION_TENANTS"
  "FS_COLLECTION_MENTIONS"
  "OAUTH_REDIRECT_URL"
  "TASKS_QUEUE_REMIND"
  "TASKS_QUEUE_ESCALATE"
  "TASKS_AUDIENCE"
  "TASKS_SERVICE_ACCOUNT"
  "REMIND_AFTER"
  "ESCALATE_AFTER"
  "APP_BASE_URL"
)

MISSING_VARS=()
for var in "${REQUIRED_VARS[@]}"; do
  if [ -z "${!var}" ]; then
    MISSING_VARS+=("$var")
  fi
done

if [ ${#MISSING_VARS[@]} -gt 0 ]; then
  echo -e "${RED}❌ 以下の環境変数が設定されていません:${NC}"
  for var in "${MISSING_VARS[@]}"; do
    echo "  - $var"
  done
  exit 1
fi
echo -e "${GREEN}✅ 全ての環境変数が設定されています${NC}"
echo -e "${YELLOW}ℹ️  Slack認証情報はSecret Managerから取得されます${NC}"
echo ""

# GCP 認証確認
echo "🔐 GCP 認証確認..."
if ! gcloud auth list | grep -q ACTIVE; then
  echo -e "${RED}❌ GCP にログインしていません${NC}"
  gcloud auth login
fi
echo -e "${GREEN}✅ GCP にログイン済み${NC}"
echo ""

# プロジェクト設定
echo "🔧 プロジェクト設定..."
gcloud config set project "$GCP_PROJECT"
echo -e "${GREEN}✅ プロジェクトを設定しました: $GCP_PROJECT${NC}"
echo ""

# イメージ名
IMAGE_NAME="gcr.io/$GCP_PROJECT/slack-reminder-bot:latest"

# Docker buildx が使用可能か確認
echo "🐳 Docker buildx の確認..."
if ! docker buildx ls | grep -q "default"; then
  echo "Docker buildx インスタンスを作成中..."
  docker buildx create --name default --use || docker buildx use default
fi
echo -e "${GREEN}✅ Docker buildx は利用可能です${NC}"
echo ""

# 古いイメージを削除（ローカルのキャッシュをクリア）
echo "🗑️  古いイメージをクリア中..."
docker rmi -f "$IMAGE_NAME" 2>/dev/null || true
docker buildx prune -f 2>/dev/null || true
echo -e "${GREEN}✅ クリア完了${NC}"
echo ""

# Container Registry に認証
echo "🔑 Container Registry に認証中..."
gcloud auth configure-docker gcr.io
echo -e "${GREEN}✅ 認証完了${NC}"
echo ""

# Docker イメージをビルド & プッシュ（linux/amd64 で固定）
echo "🐳 Docker イメージをビルド中..."
docker buildx build \
  --platform linux/amd64 \
  -t "$IMAGE_NAME" \
  --push \
  --load \
  .

if [ $? -ne 0 ]; then
  echo -e "${RED}❌ Docker イメージのビルドに失敗しました${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Docker イメージをビルド & プッシュしました${NC}"
echo ""

# Cloud Run にデプロイ
echo "🚀 Cloud Run にデプロイ中..."
gcloud run deploy slack-reminder-bot \
  --image="$IMAGE_NAME" \
  --region="$REGION" \
  --platform=managed \
  --allow-unauthenticated \
  --set-env-vars="\
GCP_PROJECT=$GCP_PROJECT,\
REGION=$REGION,\
FIRESTORE_PROJECT_ID=$FIRESTORE_PROJECT_ID,\
FS_COLLECTION_TENANTS=$FS_COLLECTION_TENANTS,\
FS_COLLECTION_MENTIONS=$FS_COLLECTION_MENTIONS,\
SLACK_SIGNING_SECRET=$SLACK_SIGNING_SECRET,\
SLACK_CLIENT_ID=$SLACK_CLIENT_ID,\
SLACK_CLIENT_SECRET=$SLACK_CLIENT_SECRET,\
OAUTH_REDIRECT_URL=$OAUTH_REDIRECT_URL,\
OAUTH_STATE_SECRET=$OAUTH_STATE_SECRET,\
SECRET_TOKEN_PREFIX=slack_token_,\
TASKS_QUEUE_REMIND=$TASKS_QUEUE_REMIND,\
TASKS_QUEUE_ESCALATE=$TASKS_QUEUE_ESCALATE,\
TASKS_AUDIENCE=$TASKS_AUDIENCE,\
TASKS_SERVICE_ACCOUNT=$TASKS_SERVICE_ACCOUNT,\
REMIND_AFTER=$REMIND_AFTER,\
ESCALATE_AFTER=$ESCALATE_AFTER,\
APP_BASE_URL=$APP_BASE_URL" \
  --service-account="run-exec@$GCP_PROJECT.iam.gserviceaccount.com" \
  --memory=512Mi \
  --cpu=1

if [ $? -ne 0 ]; then
  echo -e "${RED}❌ Cloud Run へのデプロイに失敗しました${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Cloud Run へのデプロイが完了しました${NC}"
echo ""

# サービス URL を取得
SERVICE_URL=$(gcloud run services describe slack-reminder-bot \
  --region="$REGION" \
  --format='value(status.url)')

echo "========================================"
echo -e "${GREEN}✅ デプロイが完了しました！${NC}"
echo "========================================"
echo ""
echo "📍 サービス URL: $SERVICE_URL"
echo ""
echo "💚 ヘルスチェック実行中..."
sleep 10
HEALTH_CHECK=$(curl -s -o /dev/null -w "%{http_code}" "$SERVICE_URL/health" 2>/dev/null || echo "000")
if [ "$HEALTH_CHECK" = "200" ]; then
  echo -e "${GREEN}✅ ヘルスチェック: OK${NC}"
else
  echo -e "${YELLOW}⚠️  ヘルスチェック: $HEALTH_CHECK${NC}"
fi
echo ""
echo "📝 次のステップ:"
echo "1. Slack App の設定を更新してください（https://api.slack.com/apps）:"
echo "   - Event Subscriptions → Request URL: $SERVICE_URL/slack/events"
echo "   - Interactivity & Shortcuts → Request URL: $SERVICE_URL/slack/commands"
echo "   - OAuth & Permissions → Redirect URL: $SERVICE_URL/slack/oauth_redirect"
echo ""
echo "2. アプリをワークスペースにインストール:"
echo "   Settings → Install App → Install to Workspace"
echo ""
echo "3. 動作確認:"
echo "   Slackで誰かをメンションして、10分後にリマインダーが届くか確認"
echo ""
