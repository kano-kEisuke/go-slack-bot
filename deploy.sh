#!/bin/bash

# Cloud Run デプロイスクリプト
# 使用方法: ./deploy.sh [project-id]
# 
# .env ファイルから環境変数を自動読み込みします
# セットアップ手順:
#   1. cp .env.example .env
#   2. nano .env  （各値を入力）
#   3. ./deploy.sh my-project-id

set -e

# 色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ========================================
# ステップ0: .env ファイルの確認と読み込み
# ========================================

if [ ! -f ".env" ]; then
  echo -e "${RED}❌ エラー: .env ファイルが見つかりません${NC}"
  echo ""
  echo "セットアップ手順:"
  echo "  1. ${BLUE}cp .env.example .env${NC}"
  echo "  2. ${BLUE}nano .env${NC}  （値を入力）"
  echo "  3. ${BLUE}./deploy.sh my-project-id${NC}"
  exit 1
fi

echo -e "${BLUE}📋 .env ファイルから設定を読み込み中...${NC}"
source .env

# 入力値チェック（コマンドライン引数でプロジェクト ID を指定）
PROJECT_ID="${1:-$GCP_PROJECT}"
if [ -z "$PROJECT_ID" ]; then
  echo -e "${RED}❌ エラー: GCP プロジェクト ID が指定されていません${NC}"
  echo ""
  echo "以下の方法で指定してください:"
  echo "  方法1: 引数として指定"
  echo "    ${BLUE}./deploy.sh my-project-id${NC}"
  echo ""
  echo "  方法2: .env ファイルに GCP_PROJECT を設定"
  echo "    ${BLUE}nano .env${NC}"
  echo "    ${BLUE}GCP_PROJECT=my-project-id${NC}"
  exit 1
fi

SERVICE_NAME="${SERVICE_NAME:-slack-reminder-bot}"
IMAGE_TAG="latest"

echo -e "${BLUE}📦 Cloud Run デプロイスクリプト${NC}"
echo "========================================"
echo "プロジェクト ID: $PROJECT_ID"
echo "リージョン: $REGION"
echo "サービス名: $SERVICE_NAME"
echo "========================================"
echo ""

# ========================================
# ステップ1: 環境変数の検証
# ========================================

echo -e "${BLUE}🔍 環境変数を検証中...${NC}"

# チェック対象の環境変数
REQUIRED_VARS=(
  "GCP_PROJECT"
  "REGION"
  "FIRESTORE_PROJECT_ID"
  "FS_COLLECTION_TENANTS"
  "FS_COLLECTION_MENTIONS"
  "SLACK_SIGNING_SECRET"
  "SLACK_CLIENT_ID"
  "SLACK_CLIENT_SECRET"
  "OAUTH_REDIRECT_URL"
  "OAUTH_STATE_SECRET"
  "SECRET_TOKEN_PREFIX"
  "TASKS_QUEUE_REMIND"
  "TASKS_QUEUE_ESCALATE"
  "TASKS_AUDIENCE"
  "TASKS_SERVICE_ACCOUNT"
  "REMIND_AFTER"
  "ESCALATE_AFTER"
)

MISSING_VARS=()

for var in "${REQUIRED_VARS[@]}"; do
  if [ -z "${!var}" ]; then
    MISSING_VARS+=("$var")
  fi
done

if [ ${#MISSING_VARS[@]} -gt 0 ]; then
  echo -e "${RED}❌ エラー: 以下の環境変数が設定されていません:${NC}"
  echo ""
  for var in "${MISSING_VARS[@]}"; do
    echo -e "  ${RED}• $var${NC}"
  done
  echo ""
  echo "対応方法:"
  echo "  1. .env ファイルを開く:"
  echo -e "     ${BLUE}nano .env${NC}"
  echo "  2. 不足している値を入力する"
  echo "  3. ファイルを保存（Ctrl+O, Enter, Ctrl+X）"
  echo "  4. スクリプトを再実行:"
  echo -e "     ${BLUE}./deploy.sh $PROJECT_ID${NC}"
  echo ""
  echo "詳細は SETUP_GUIDE.md を参照:"
  echo -e "  ${BLUE}less SETUP_GUIDE.md${NC}"
  exit 1
fi

echo -e "${GREEN}✅ 全ての環境変数が設定されています${NC}"
echo ""

# ========================================
# ステップ2: GCP ログイン確認
# ========================================

echo -e "${BLUE}🔐 GCP 認証確認...${NC}"
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
  echo -e "${RED}❌ GCP にログインしていません${NC}"
  echo "以下を実行してください:"
  echo -e "  ${BLUE}gcloud auth login${NC}"
  exit 1
fi
echo -e "${GREEN}✅ GCP にログイン済み${NC}"
echo ""

# プロジェクト設定
echo -e "${BLUE}🔧 プロジェクト設定...${NC}"
gcloud config set project $PROJECT_ID
echo -e "${GREEN}✅ プロジェクトを設定しました: $PROJECT_ID${NC}"
echo ""

# Docker イメージビルド
echo -e "${BLUE}🐳 Docker イメージをビルド中...${NC}"
docker build -t slack-reminder-bot:$IMAGE_TAG .
echo -e "${GREEN}✅ Docker イメージをビルドしました${NC}"
echo ""

# Container Registry 認証
echo -e "${BLUE}🔑 Container Registry に認証中...${NC}"
gcloud auth configure-docker
echo -e "${GREEN}✅ Container Registry に認証しました${NC}"
echo ""

# イメージをタグ付け
echo -e "${BLUE}🏷️  イメージにタグ付けしています...${NC}"
docker tag slack-reminder-bot:$IMAGE_TAG gcr.io/$PROJECT_ID/$SERVICE_NAME:$IMAGE_TAG
echo -e "${GREEN}✅ イメージをタグ付けしました: gcr.io/$PROJECT_ID/$SERVICE_NAME:$IMAGE_TAG${NC}"
echo ""

# Container Registry にプッシュ
echo -e "${BLUE}📤 イメージを Container Registry にプッシュ中...${NC}"
docker push gcr.io/$PROJECT_ID/$SERVICE_NAME:$IMAGE_TAG
echo -e "${GREEN}✅ イメージをプッシュしました${NC}"
echo ""

# Cloud Run にデプロイ
echo -e "${BLUE}🚀 Cloud Run にデプロイ中...${NC}"
gcloud run deploy $SERVICE_NAME \
  --image gcr.io/$PROJECT_ID/$SERVICE_NAME:$IMAGE_TAG \
  --region $REGION \
  --platform managed \
  --allow-unauthenticated \
  --memory 512Mi \
  --cpu 1 \
  --timeout 3600s \
  --max-instances 100 \
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
SECRET_TOKEN_PREFIX=$SECRET_TOKEN_PREFIX,\
TASKS_QUEUE_REMIND=$TASKS_QUEUE_REMIND,\
TASKS_QUEUE_ESCALATE=$TASKS_QUEUE_ESCALATE,\
TASKS_AUDIENCE=$TASKS_AUDIENCE,\
TASKS_SERVICE_ACCOUNT=$TASKS_SERVICE_ACCOUNT,\
REMIND_AFTER=$REMIND_AFTER,\
ESCALATE_AFTER=$ESCALATE_AFTER"

echo ""
echo -e "${GREEN}✅ デプロイが完了しました！${NC}"
echo ""

# サービス URL 表示
SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --region $REGION --format='value(status.url)')
echo -e "${BLUE}📍 サービス URL: ${GREEN}$SERVICE_URL${NC}"
echo ""

# ヘルスチェック
echo -e "${BLUE}💚 ヘルスチェック実行中...${NC}"
sleep 3
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" $SERVICE_URL/health)
if [ "$HEALTH_STATUS" = "200" ]; then
  echo -e "${GREEN}✅ ヘルスチェック: OK${NC}"
else
  echo -e "${YELLOW}⚠️  ヘルスチェック: $HEALTH_STATUS (予期しない状態)${NC}"
fi
echo ""

echo -e "${BLUE}📋 ログを確認するコマンド:${NC}"
echo -e "  ${BLUE}gcloud run services logs read $SERVICE_NAME --region $REGION --limit 50${NC}"
echo ""

echo -e "${BLUE}🎯 次のステップ:${NC}"
echo "  1. Slack App Dashboard で OAUTH_REDIRECT_URL を更新"
echo -e "     URL: ${GREEN}$SERVICE_URL/slack/oauth_redirect${NC}"
echo "  2. https://api.slack.com/apps にアクセス"
echo "  3. OAuth & Permissions → Redirect URLs を更新"
echo "  4. .env ファイルの OAUTH_REDIRECT_URL も更新"
echo ""

echo -e "${GREEN}✨ デプロイ完了!${NC}"
