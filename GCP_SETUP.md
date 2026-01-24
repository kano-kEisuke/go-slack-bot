# ☁️ GCP セットアップガイド

このガイドに従って、Google Cloud Platform のセットアップを行ってください。

---

## 📋 GCP セットアップのフロー

```
1. GCP アカウント確認・ログイン
2. GCP プロジェクト作成
3. 必要な API を有効化
4. Firestore データベース作成
5. Cloud Tasks キューを作成
6. サービスアカウント作成・権限設定
7. Secret Manager にシークレット登録
```

**所要時間**: 40分

---

## 1️⃣ GCP アカウント確認・ログイン

### ステップ1: Google アカウント確認

```bash
# 現在のログイン状態確認
gcloud auth list
```

出力例：
```
     ACTIVE  ACCOUNT
*           user@example.com
```

ACTIVE が付いていればログイン済みです。

### ステップ2: ログイン（必要な場合）

```bash
gcloud auth login
```

ブラウザが開き、Google アカウントでのログインが要求されます。

---

## 2️⃣ GCP プロジェクト作成

### ターミナルで作成（推奨）

```bash
# プロジェクト作成
gcloud projects create my-slack-bot-project \
  --name="Slack Reminder Bot"

# プロジェクト ID を確認
gcloud config get-value project
# 出力: my-slack-bot-project

# プロジェクトを設定
gcloud config set project my-slack-bot-project
```

### または、コンソールで作成

1. [Google Cloud Console](https://console.cloud.google.com/) にアクセス
2. ページ上部の **プロジェクト選択** をクリック
3. **新しいプロジェクト** をクリック
4. **プロジェクト名**: `Slack Reminder Bot`
5. **作成** をクリック

---

## 3️⃣ 必要な API を有効化

```bash
# Firestore API
gcloud services enable firestore.googleapis.com

# Cloud Run API
gcloud services enable run.googleapis.com

# Cloud Tasks API
gcloud services enable cloudtasks.googleapis.com

# Secret Manager API
gcloud services enable secretmanager.googleapis.com

# Cloud Logging API
gcloud services enable logging.googleapis.com

# Container Registry API
gcloud services enable containerregistry.googleapis.com
```

### 確認

```bash
gcloud services list --enabled | grep -E "firestore|run|cloudtasks|secretmanager"
```

有効化されていれば表示されます。

---

## 4️⃣ Firestore データベース作成

### ステップ1: Firestore の初期化

```bash
gcloud firestore databases create \
  --region=asia-northeast1
```

### ステップ2: 確認

```bash
gcloud firestore databases list
```

出力例：
```
NAME          TYPE             LOCATION         DELETE_TIME
(default)     FIRESTORE_NATIVE asia-northeast1  
```

### または、コンソールで確認

1. [Google Cloud Console](https://console.cloud.google.com/)
2. 左メニュー → **Firestore**
3. データベースが表示されているか確認

---

## 5️⃣ Cloud Tasks キューを作成

### ステップ1: キュー作成

```bash
# 10分後のリマインド用キュー
gcloud tasks queues create remind-queue \
  --location=asia-northeast1

# 30分後のエスカレーション用キュー
gcloud tasks queues create escalate-queue \
  --location=asia-northeast1
```

### ステップ2: 確認

```bash
gcloud tasks queues list --location=asia-northeast1
```

出力例：
```
NAME              LOCATION            RESPONSE_HANDLER
remind-queue      asia-northeast1     
escalate-queue    asia-northeast1     
```

### ステップ3: 完全リソース名を取得

```bash
# プロジェクト ID を確認
export PROJECT_ID=$(gcloud config get-value project)
echo $PROJECT_ID

# 完全リソース名を表示
echo "Remind Queue: projects/$PROJECT_ID/locations/asia-northeast1/queues/remind-queue"
echo "Escalate Queue: projects/$PROJECT_ID/locations/asia-northeast1/queues/escalate-queue"
```

これらを `.env` ファイルに設定します：
```env
TASKS_QUEUE_REMIND=projects/my-slack-bot-project/locations/asia-northeast1/queues/remind-queue
TASKS_QUEUE_ESCALATE=projects/my-slack-bot-project/locations/asia-northeast1/queues/escalate-queue
```

---

## 6️⃣ サービスアカウント作成・権限設定

### ステップ1: サービスアカウント作成

```bash
gcloud iam service-accounts create slack-bot-service \
  --display-name="Slack Reminder Bot Service Account"
```

### ステップ2: 必要な権限を付与

```bash
export PROJECT_ID=$(gcloud config get-value project)

# 1. Cloud Run Invoker（Cloud Run の起動）
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:slack-bot-service@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/run.invoker"

# 2. Firestore User（Firestore の読み書き）
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:slack-bot-service@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/datastore.user"

# 3. Cloud Tasks Enqueuer（タスクの作成）
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:slack-bot-service@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/cloudtasks.enqueuer"

# 4. Secret Manager Secret Accessor（シークレットの読み取り）
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:slack-bot-service@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# 5. Secret Manager Admin（シークレットの作成・更新、OAuth時に必要）
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:slack-bot-service@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.admin"
```

### ステップ3: 権限の確認

```bash
gcloud projects get-iam-policy $PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:slack-bot-service@$PROJECT_ID.iam.gserviceaccount.com" \
  --format="table(bindings.role)"
```

出力例：
```
ROLE
roles/run.invoker
roles/datastore.user
roles/cloudtasks.enqueuer
roles/secretmanager.secretAccessor
roles/secretmanager.admin
```
DISPLAY NAME                          EMAIL
Slack Reminder Bot Service Account    slack-bot-service@my-slack-bot-project.iam.gserviceaccount.com
```

---

## 7️⃣ Secret Manager にシークレット登録

このアプリは、Slack認証情報をSecret Managerで管理します。環境変数ではなく、GCPのSecret Managerに保存することでセキュリティを強化しています。

### ステップ1: OAuth State Secret を生成・登録

```bash
# ランダムな文字列を生成
openssl rand -base64 32
# 出力例: abc123xyz789...（これをコピー）

# Secret Manager に登録
echo -n "abc123xyz789..." | \
  gcloud secrets create oauth-state-secret --data-file=-
```

### ステップ2: Slack認証情報を登録（Slack App作成後）

⚠️ **重要**: 以下のコマンドは、[SLACK_SETUP.md](SLACK_SETUP.md)でSlack Appを作成した後に実行してください。

```bash
# Slack Signing Secret を登録
# Slack App Dashboard → Settings → Basic Information → App Credentials → Signing Secret
echo -n "your-signing-secret-here" | \
  gcloud secrets create slack-signing-secret --data-file=-

# Slack Client ID を登録
# Slack App Dashboard → Settings → Basic Information → App Credentials → Client ID
echo -n "your-client-id-here" | \
  gcloud secrets create slack-client-id --data-file=-

# Slack Client Secret を登録
# Slack App Dashboard → Settings → Basic Information → App Credentials → Client Secret
echo -n "your-client-secret-here" | \
  gcloud secrets create slack-client-secret --data-file=-
```

### ステップ3: Secret一覧確認

```bash
gcloud secrets list
```

出力例：
```
NAME                      CREATED              REPLICATION_POLICY  LOCATIONS
oauth-state-secret        2024-01-15T10:00:00  automatic           -
slack-signing-secret      2024-01-15T10:01:00  automatic           -
slack-client-id           2024-01-15T10:02:00  automatic           -
slack-client-secret       2024-01-15T10:03:00  automatic           -
```

### 📝 Secret Managerの仕組み

- **環境変数に直接書かない**: `.env`ファイルには`from-secret-manager`というダミー値のみ
- **実行時に取得**: アプリ起動時にSecret Managerから実際の値を取得
- **OAuth時に自動作成**: Slack AppをインストールするとBot Tokenが自動で`slack_token_{TEAM_ID}`として保存される
- **権限管理**: サービスアカウントに`secretmanager.admin`権限を付与済み（ステップ6で実施）

---

## ✅ GCP セットアップ完了チェックリスト

以下を確認したら、GCP セットアップは完了です：

- [ ] `gcloud auth list` でログイン確認
- [ ] GCP プロジェクト作成済み
- [ ] 必要な API が有効化済み（6つ）
- [ ] Firestore データベース作成済み
- [ ] Cloud Tasks キュー 2 つ作成済み
- [ ] サービスアカウント作成済み
- [ ] サービスアカウントに 5 つの権限付与済み
- [ ] Secret Manager に OAuth State Secret 登録済み
- [ ] Secret Manager に Slack 認証情報登録済み（Slack App 作成後）

---

## 🔍 トラブルシューティング

### エラー: `gcloud: command not found`

Google Cloud SDK がインストールされていません。

**インストール手順**: [Google Cloud SDK をインストール](https://cloud.google.com/sdk/docs/install)

### エラー: `You do not currently have an active account`

GCP にログインしていません。

```bash
gcloud auth login
```

### エラー: `Firestore API is disabled`

API が有効化されていません。

```bash
gcloud services enable firestore.googleapis.com
```

### エラー: `Queue already exists`

キューが既に存在しています（問題なし）。

```bash
# キュー一覧確認
gcloud tasks queues list --location=asia-northeast1

# 削除したい場合
gcloud tasks queues delete remind-queue --location=asia-northeast1
```

---

## 📊 確認用コマンド集

```bash
# プロジェクト ID
gcloud config get-value project

# API 有効化確認
gcloud services list --enabled | grep -i firestore

# Firestore データベース確認
gcloud firestore databases list

# Cloud Tasks キュー確認
gcloud tasks queues list --location=asia-northeast1

# サービスアカウント確認
gcloud iam service-accounts list

# サービスアカウントの権限確認
gcloud projects get-iam-policy $(gcloud config get-value project) \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:slack-bot-service*" \
  --format="table(bindings.role)"

# Secret 確認
gcloud secrets list
```

---

## 🚀 次のステップ

1. **Slack App セットアップ**: [`SLACK_SETUP.md`](SLACK_SETUP.md) へ進む
2. **環境変数設定**: `.env.example` をコピーして `.env` を作成
3. **デプロイ**: [`SETUP_GUIDE.md`](SETUP_GUIDE.md) のフェーズ4 に進む

---

## 📞 料金に関して

**無料枠の確認**:

```bash
gcloud billing accounts list
```

GCP では以下のサービスに無料枠があります（2025年現在）：

- **Cloud Run**: 毎月 200 万リクエスト無料
- **Firestore**: 毎日 50,000 読み取り / 20,000 書き込み無料
- **Cloud Tasks**: 毎月 100 万 API 呼び出し無料
- **Secret Manager**: 毎月アクティブシークレット 6 個まで無料

**詳細**: [GCP 価格計算ツール](https://cloud.google.com/products/calculator)
