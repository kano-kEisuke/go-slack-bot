#!/bin/bash

# ========================================
# GCP Secret Manager 初期設定スクリプト
# ========================================
# このスクリプトは、Slack Bot の認証情報を GCP Secret Manager に保存します。
#
# 使用方法:
#   ./scripts/setup_secrets.sh
#
# 前提条件:
#   - gcloud CLI がインストールされている
#   - GCP プロジェクトが作成されている
#   - 適切な権限（Secret Manager Admin）を持っている

set -e  # エラー時に停止

# カラー出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# プロジェクトIDを取得
PROJECT_ID=$(gcloud config get-value project 2>/dev/null)

if [ -z "$PROJECT_ID" ]; then
  echo -e "${RED}エラー: GCP プロジェクトが設定されていません${NC}"
  echo "以下のコマンドでプロジェクトを設定してください:"
  echo "  gcloud config set project YOUR_PROJECT_ID"
  exit 1
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}GCP Secret Manager 初期設定${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "プロジェクトID: ${GREEN}${PROJECT_ID}${NC}"
echo ""

# Secret Manager API を有効化
echo -e "${YELLOW}[1/6] Secret Manager API を有効化しています...${NC}"
gcloud services enable secretmanager.googleapis.com --project=$PROJECT_ID
echo -e "${GREEN}✓ 完了${NC}"
echo ""

# Slack App の認証情報を入力
echo -e "${YELLOW}[2/6] Slack App の認証情報を入力してください${NC}"
echo -e "https://api.slack.com/apps から取得できます"
echo ""

# SLACK_SIGNING_SECRET
echo -e "${BLUE}Slack Signing Secret を入力してください:${NC}"
echo "場所: Settings → Basic Information → App Credentials → Signing Secret"
read -s SLACK_SIGNING_SECRET
if [ -z "$SLACK_SIGNING_SECRET" ]; then
  echo -e "${RED}エラー: 値が入力されていません${NC}"
  exit 1
fi
echo -e "${GREEN}✓ 入力完了${NC}"
echo ""

# SLACK_CLIENT_ID
echo -e "${BLUE}Slack Client ID を入力してください:${NC}"
echo "場所: Settings → Basic Information → App Credentials → Client ID"
read SLACK_CLIENT_ID
if [ -z "$SLACK_CLIENT_ID" ]; then
  echo -e "${RED}エラー: 値が入力されていません${NC}"
  exit 1
fi
echo -e "${GREEN}✓ 入力完了${NC}"
echo ""

# SLACK_CLIENT_SECRET
echo -e "${BLUE}Slack Client Secret を入力してください:${NC}"
echo "場所: Settings → Basic Information → App Credentials → Client Secret"
read -s SLACK_CLIENT_SECRET
if [ -z "$SLACK_CLIENT_SECRET" ]; then
  echo -e "${RED}エラー: 値が入力されていません${NC}"
  exit 1
fi
echo -e "${GREEN}✓ 入力完了${NC}"
echo ""

# OAUTH_STATE_SECRET を生成
echo -e "${YELLOW}[3/6] OAuth State Secret を自動生成しています...${NC}"
OAUTH_STATE_SECRET=$(openssl rand -base64 32)
echo -e "${GREEN}✓ 生成完了: ${OAUTH_STATE_SECRET}${NC}"
echo ""

# Secret Manager にシークレットを保存
echo -e "${YELLOW}[4/6] Secret Manager にシークレットを保存しています...${NC}"

# slack-signing-secret
echo -n "  - slack-signing-secret... "
if gcloud secrets describe slack-signing-secret --project=$PROJECT_ID &>/dev/null; then
  echo -n "(既存) "
  echo -n "$SLACK_SIGNING_SECRET" | gcloud secrets versions add slack-signing-secret --data-file=- --project=$PROJECT_ID
else
  echo -n "$SLACK_SIGNING_SECRET" | gcloud secrets create slack-signing-secret --data-file=- --project=$PROJECT_ID
fi
echo -e "${GREEN}✓${NC}"

# slack-client-id
echo -n "  - slack-client-id... "
if gcloud secrets describe slack-client-id --project=$PROJECT_ID &>/dev/null; then
  echo -n "(既存) "
  echo -n "$SLACK_CLIENT_ID" | gcloud secrets versions add slack-client-id --data-file=- --project=$PROJECT_ID
else
  echo -n "$SLACK_CLIENT_ID" | gcloud secrets create slack-client-id --data-file=- --project=$PROJECT_ID
fi
echo -e "${GREEN}✓${NC}"

# slack-client-secret
echo -n "  - slack-client-secret... "
if gcloud secrets describe slack-client-secret --project=$PROJECT_ID &>/dev/null; then
  echo -n "(既存) "
  echo -n "$SLACK_CLIENT_SECRET" | gcloud secrets versions add slack-client-secret --data-file=- --project=$PROJECT_ID
else
  echo -n "$SLACK_CLIENT_SECRET" | gcloud secrets create slack-client-secret --data-file=- --project=$PROJECT_ID
fi
echo -e "${GREEN}✓${NC}"

# oauth-state-secret
echo -n "  - oauth-state-secret... "
if gcloud secrets describe oauth-state-secret --project=$PROJECT_ID &>/dev/null; then
  echo -n "(既存) "
  echo -n "$OAUTH_STATE_SECRET" | gcloud secrets versions add oauth-state-secret --data-file=- --project=$PROJECT_ID
else
  echo -n "$OAUTH_STATE_SECRET" | gcloud secrets create oauth-state-secret --data-file=- --project=$PROJECT_ID
fi
echo -e "${GREEN}✓${NC}"
echo ""

# サービスアカウントに権限を付与
echo -e "${YELLOW}[5/6] サービスアカウントに権限を付与しています...${NC}"
SERVICE_ACCOUNT="run-exec@${PROJECT_ID}.iam.gserviceaccount.com"

# サービスアカウントが存在するか確認
if gcloud iam service-accounts describe $SERVICE_ACCOUNT --project=$PROJECT_ID &>/dev/null; then
  gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/secretmanager.secretAccessor" \
    --condition=None \
    --quiet
  echo -e "${GREEN}✓ 完了${NC}"
else
  echo -e "${YELLOW}⚠ サービスアカウント ${SERVICE_ACCOUNT} が見つかりません${NC}"
  echo -e "${YELLOW}  Cloud Run デプロイ後に権限を付与してください${NC}"
fi
echo ""

# 検証
echo -e "${YELLOW}[6/6] 設定を検証しています...${NC}"
echo "保存されたシークレット:"
gcloud secrets list --project=$PROJECT_ID | grep -E "slack-|oauth-" || true
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ すべての設定が完了しました！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "次のステップ:"
echo "  1. アプリケーションをローカルでテスト"
echo "     gcloud auth application-default login"
echo "     go run project/cmd/main.go"
echo ""
echo "  2. Cloud Run にデプロイ"
echo "     gcloud run deploy slack-reminder-bot --source . --region asia-northeast1"
echo ""
echo -e "${BLUE}詳細は ENV_SETUP.md を参照してください${NC}"
