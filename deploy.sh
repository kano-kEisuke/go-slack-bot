#!/bin/bash

# Cloud Run デプロイスクリプト
# 使用方法: ./deploy.sh [project-id] [region] [service-name]

set -e

# デフォルト値
PROJECT_ID="${1:-}"
REGION="${2:-asia-northeast1}"
SERVICE_NAME="${3:-slack-reminder-bot}"
IMAGE_TAG="latest"

# 入力値チェック
if [ -z "$PROJECT_ID" ]; then
  echo "❌ エラー: GCP プロジェクト ID が指定されていません"
  echo ""
  echo "使用方法: ./deploy.sh <project-id> [region] [service-name]"
  echo ""
  echo "例:"
  echo "  ./deploy.sh my-gcp-project asia-northeast1 slack-reminder-bot"
  exit 1
fi

echo "📦 Cloud Run デプロイスクリプト"
echo "========================================"
echo "プロジェクト ID: $PROJECT_ID"
echo "リージョン: $REGION"
echo "サービス名: $SERVICE_NAME"
echo "========================================"
echo ""

# GCP ログイン確認
echo "🔐 GCP 認証確認..."
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
  echo "❌ GCP にログインしていません。以下を実行してください:"
  echo "  gcloud auth login"
  exit 1
fi
echo "✅ GCP にログイン済み"
echo ""

# プロジェクト設定
echo "🔧 プロジェクト設定..."
gcloud config set project $PROJECT_ID
echo "✅ プロジェクトを設定しました: $PROJECT_ID"
echo ""

# Docker イメージビルド
echo "🐳 Docker イメージをビルド中..."
docker build -t slack-reminder-bot:$IMAGE_TAG .
echo "✅ Docker イメージをビルドしました"
echo ""

# Container Registry 認証
echo "🔑 Container Registry に認証中..."
gcloud auth configure-docker
echo "✅ Container Registry に認証しました"
echo ""

# イメージをタグ付け
echo "🏷️  イメージにタグ付けしています..."
docker tag slack-reminder-bot:$IMAGE_TAG gcr.io/$PROJECT_ID/$SERVICE_NAME:$IMAGE_TAG
echo "✅ イメージをタグ付けしました: gcr.io/$PROJECT_ID/$SERVICE_NAME:$IMAGE_TAG"
echo ""

# Container Registry にプッシュ
echo "📤 イメージを Container Registry にプッシュ中..."
docker push gcr.io/$PROJECT_ID/$SERVICE_NAME:$IMAGE_TAG
echo "✅ イメージをプッシュしました"
echo ""

# Cloud Run にデプロイ
echo "🚀 Cloud Run にデプロイ中..."
gcloud run deploy $SERVICE_NAME \
  --image gcr.io/$PROJECT_ID/$SERVICE_NAME:$IMAGE_TAG \
  --region $REGION \
  --platform managed \
  --allow-unauthenticated \
  --memory 512Mi \
  --cpu 1 \
  --timeout 3600s \
  --max-instances 100 \
  --set-env-vars="GCP_PROJECT=$PROJECT_ID,FIRESTORE_DB=slack-reminder"

echo ""
echo "✅ デプロイが完了しました！"
echo ""

# サービス URL 表示
SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --region $REGION --format='value(status.url)')
echo "📍 サービス URL: $SERVICE_URL"
echo ""

# ヘルスチェック
echo "💚 ヘルスチェック実行中..."
sleep 3
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" $SERVICE_URL/health)
if [ "$HEALTH_STATUS" = "200" ]; then
  echo "✅ ヘルスチェック: OK"
else
  echo "⚠️  ヘルスチェック: $HEALTH_STATUS (予期しない状態)"
fi
echo ""

echo "📋 ログを確認するコマンド:"
echo "  gcloud run services logs read $SERVICE_NAME --region $REGION --limit 50"
echo ""

echo "✨ デプロイ完了!"
