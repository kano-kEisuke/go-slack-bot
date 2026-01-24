# 環境変数セットアップガイド（GCP Secret Manager 版）

## 概要

このアプリケーションは、セキュリティを確保するため、**GCP Secret Manager** を使用してセンシティブな情報を管理します。
Slack認証情報やOAuthシークレットは、ソースコードや `.env` ファイルには記載せず、Secret Manager から動的に読み込まれます。

## セキュリティのメリット

✅ **ソースコード管理から分離**: シークレットが誤ってGitにコミットされるリスクがゼロ  
✅ **アクセス制御**: IAMでシークレットへのアクセスを細かく制御可能  
✅ **監査ログ**: シークレットへのアクセス履歴を記録  
✅ **バージョン管理**: シークレットのローテーションと履歴管理が容易  
✅ **暗号化**: 保存時・転送時ともに自動暗号化

## 必須設定手順

### 1. GCP Secret Manager を有効化

```bash
# Secret Manager API を有効化
gcloud services enable secretmanager.googleapis.com

# プロジェクトIDを確認
gcloud config get-value project
```

### 2. Slack App の認証情報を取得

1. https://api.slack.com/apps にアクセス
2. あなたのアプリを選択
3. 以下の値を取得（メモ帳などに一時保存）：

#### SLACK_SIGNING_SECRET
- **場所**: Settings → Basic Information → App Credentials → Signing Secret
- **例**: `1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p`

#### SLACK_CLIENT_ID  
- **場所**: Settings → Basic Information → App Credentials → Client ID
- **例**: `1234567890.1234567890`

#### SLACK_CLIENT_SECRET
- **場所**: Settings → Basic Information → App Credentials → Client Secret
- **例**: `1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p`

### 3. OAuth State Secret を生成

セキュリティ用のランダム文字列を生成します：

```bash
openssl rand -base64 32
```

出力例: `Xy7ZqW3vR5tN9kL2mP8jH4fD6gS1aE0cV7bN5xQ3wR8=`

### 4. Secret Manager にシークレットを保存

以下のコマンドで、取得した値を Secret Manager に保存します：

```bash
# プロジェクトIDを設定（環境に合わせて変更）
PROJECT_ID="slack-reminder-bot-20251114"

# 1. Slack Signing Secret を保存
echo -n "取得したSIGNING_SECRETをここに貼り付け" | \
  gcloud secrets create slack-signing-secret \
  --data-file=- \
  --project=$PROJECT_ID

# 2. Slack Client ID を保存
echo -n "取得したCLIENT_IDをここに貼り付け" | \
  gcloud secrets create slack-client-id \
  --data-file=- \
  --project=$PROJECT_ID

# 3. Slack Client Secret を保存
echo -n "取得したCLIENT_SECRETをここに貼り付け" | \
  gcloud secrets create slack-client-secret \
  --data-file=- \
  --project=$PROJECT_ID

# 4. OAuth State Secret を保存
echo -n "生成したOAUTH_STATE_SECRETをここに貼り付け" | \
  gcloud secrets create oauth-state-secret \
  --data-file=- \
  --project=$PROJECT_ID
```

### 5. サービスアカウントに権限を付与

Cloud Run で実行するサービスアカウントに、Secret Manager へのアクセス権限を付与します：

```bash
# サービスアカウント名を設定
SERVICE_ACCOUNT="run-exec@${PROJECT_ID}.iam.gserviceaccount.com"

# Secret Manager へのアクセス権限を付与
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:${SERVICE_ACCOUNT}" \
  --role="roles/secretmanager.secretAccessor"
```

### 6. シークレットの確認

保存されたシークレットを確認します：

```bash
# シークレット一覧を表示
gcloud secrets list --project=$PROJECT_ID

# 特定のシークレットの値を確認（テスト用）
gcloud secrets versions access latest --secret="slack-signing-secret" --project=$PROJECT_ID
```

### 7. アプリケーションの起動

環境変数が正しく設定されていれば、アプリケーションが起動します：

```bash
# ローカル起動（GCPの認証情報が必要）
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
go run project/cmd/main.go

# Cloud Run へデプロイ
gcloud run deploy slack-reminder-bot \
  --source . \
  --region asia-northeast1 \
  --project $PROJECT_ID
```

## エラーメッセージ

環境変数が正しく設定されていない場合、以下のようなエラーが表示されます：

```
設定読み込み失敗: SLACK_SIGNING_SECRET 取得失敗: Secret Manager からの取得失敗 (name=slack-signing-secret): rpc error: code = NotFound desc = Secret [projects/.../secrets/slack-signing-secret] not found
```

このエラーが表示された場合は、該当するシークレットが Secret Manager に保存されているか確認してください。

## シークレットの更新（ローテーション）

セキュリティのため、定期的にシークレットを更新することを推奨します：

```bash
# 既存のシークレットに新しいバージョンを追加
echo -n "新しいSIGNING_SECRET" | \
  gcloud secrets versions add slack-signing-secret \
  --data-file=- \
  --project=$PROJECT_ID

# 古いバージョンを無効化（必要に応じて）
gcloud secrets versions disable VERSION_ID \
  --secret="slack-signing-secret" \
  --project=$PROJECT_ID
```

## ローカル開発環境の設定

ローカルで開発する場合は、GCPの認証情報が必要です：

### 方法1: gcloud CLI で認証（推奨）

```bash
# GCP にログイン
gcloud auth login

# アプリケーションデフォルト認証を設定
gcloud auth application-default login

# プロジェクトを設定
gcloud config set project $PROJECT_ID
```

### 方法2: サービスアカウントキーを使用

```bash
# サービスアカウントキーを作成
gcloud iam service-accounts keys create ~/key.json \
  --iam-account=$SERVICE_ACCOUNT

# 環境変数に設定
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/key.json"

# アプリケーションを起動
go run project/cmd/main.go
```

⚠️ **注意**: サービスアカウントキーは厳重に管理し、Gitにコミットしないでください。

## トラブルシューティング

### Q: Secret Manager からシークレットを取得できない

**A**: 以下を確認してください：

1. Secret Manager API が有効化されているか
   ```bash
   gcloud services list --enabled | grep secretmanager
   ```

2. サービスアカウントに適切な権限があるか
   ```bash
   gcloud projects get-iam-policy $PROJECT_ID \
     --flatten="bindings[].members" \
     --filter="bindings.members:serviceAccount:$SERVICE_ACCOUNT"
   ```

3. シークレットが存在するか
   ```bash
   gcloud secrets describe slack-signing-secret --project=$PROJECT_ID
   ```

### Q: ローカルで "permission denied" エラーが出る

**A**: 認証情報が正しく設定されていません：

```bash
# 認証状態を確認
gcloud auth list

# アプリケーションデフォルト認証を再設定
gcloud auth application-default login
```

### Q: 特定のシークレットだけ取得できない

**A**: シークレット名のスペルミスや、バージョンの問題の可能性があります：

```bash
# シークレットの最新バージョンを確認
gcloud secrets versions list slack-signing-secret --project=$PROJECT_ID

# 手動でアクセスしてテスト
gcloud secrets versions access latest \
  --secret="slack-signing-secret" \
  --project=$PROJECT_ID
```

## セキュリティのベストプラクティス

### ✅ すべき事

1. **Secret Manager を使用する**
   - すべてのセンシティブ情報は Secret Manager で管理
   - ソースコードや `.env` ファイルには記載しない

2. **IAM 権限を最小限に**
   - サービスアカウントには必要最小限の権限のみ付与
   - `roles/secretmanager.secretAccessor` のみで十分

3. **定期的にシークレットをローテーション**
   - 特に `slack-client-secret` と `oauth-state-secret` は3〜6ヶ月ごとに更新
   - Secret Manager のバージョン管理機能を活用

4. **監査ログを確認**
   ```bash
   # Secret Manager へのアクセスログを確認
   gcloud logging read "resource.type=secretmanager.googleapis.com/Secret" \
     --project=$PROJECT_ID \
     --limit=50
   ```

5. **環境ごとにシークレットを分離**
   - 開発環境と本番環境で異なる Secret Manager プロジェクトを使用
   - または、シークレット名にプレフィックスを付ける（例: `prod-slack-signing-secret`）

### ❌ してはいけない事

1. **シークレットをハードコーディングしない**
   ```go
   // ❌ ダメな例
   const SlackClientSecret = "abc123secret"
   
   // ✅ 良い例（Secret Manager から取得）
   secret, err := getSecretFromManager(ctx, client, projectID, "slack-client-secret")
   ```

2. **`.env` ファイルをGitにコミットしない**
   - `.gitignore` に `.env` が含まれていることを確認
   - テンプレートとして `.env.example` を使用

3. **サービスアカウントキーをソースコード管理に含めない**
   - `*.json` を `.gitignore` に追加
   - Cloud Run では Workload Identity を使用（キー不要）

4. **過剰な権限を付与しない**
   - `roles/owner` や `roles/editor` は使用しない
   - 必要最小限の権限のみ付与

## Secret Manager の便利なコマンド

```bash
# すべてのシークレットをリスト表示
gcloud secrets list --project=$PROJECT_ID

# 特定のシークレットの詳細を表示
gcloud secrets describe slack-signing-secret --project=$PROJECT_ID

# シークレットのバージョン履歴を表示
gcloud secrets versions list slack-signing-secret --project=$PROJECT_ID

# 古いバージョンを削除
gcloud secrets versions destroy VERSION_ID \
  --secret="slack-signing-secret" \
  --project=$PROJECT_ID

# シークレット全体を削除（注意！）
gcloud secrets delete slack-signing-secret --project=$PROJECT_ID
```

## 参考リンク

- [Slack API - Basic App Setup](https://api.slack.com/authentication/basics)
- [GCP Secret Manager](https://cloud.google.com/secret-manager/docs)
- [Cloud Run - Secret Manager Integration](https://cloud.google.com/run/docs/configuring/secrets)
