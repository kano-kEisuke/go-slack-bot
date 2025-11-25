# Slack返信リマインドBot 仕様書（ドラフト）

## 01. 概要
- **目的**：メンションされた人の返信遅延を防止し、必要に応じて上長へ自動エスカレーションする。
- **実行契機**：Slack上で **Botへのメンションを含むメッセージ** が投稿されたとき。
- **対応対象**：Botが招待されている **DM / グループDM / チャンネル**（スレッド含む）。

---

## 02. 主要アクター
- **Bot**：本アプリ（Cloud Runでホスト）
- **送信者**：メンション付きメッセージを送るユーザー
- **対象者**：Bot以外でメンションされたユーザー（返信を求められる人）
- **上長**：返信がない場合にDM通知を受け取るユーザー（ワークスペースごとに設定）

---

## 03. トリガー（イベント）
- メッセージに **`@Bot`** が含まれている。
- 同じメッセージ内に、**`@対象者`（Bot以外）** が1人以上含まれている。
- 対象者ごとに「返信監視」を開始する。

---

## 04. 期待動作（基本フロー）
1. **t=0（トリガー時）**  
   - Botはメッセージから **全てのメンション（Bot以外）** を抽出。  
   - 対象者ごとに監視レコードを保存。  
   - **10分後**チェック＆**30分後**チェックのジョブを予約。

2. **t=+10分：初回リマインド**  
   - 対象者が **メンション付き返信をしていない場合**、**スレッドにリマインド投稿**（対象者をメンション）。
   - メンション付き返信済みなら何もしない。

3. **t=+30分：再送＆上長通知**  
   - 対象者が **メンション付き返信をしていない場合**、
     - **スレッドに再リマインド**（対象者をメンション）  
     - **上長にDM通知**（設定されている場合のみ）
   - メンション付き返信済みなら何もしない。

> 返信の定義：**対象者がトリガーメッセージ送信者へ @メンション をつけて返信**していること。  
> 例：  
> - ユーザーA: "進捗報告お願いします @bot @ユーザーB"  
> - ユーザーB: "８割です @ユーザーA" ← この場合のみ返信完了と判定

---

## 05. 通知メッセージ（文面案）
- **10分リマインド（スレッドに投稿）**  
  - `@対象者 さん、お手すきの際にご返信お願いします🙏（自動リマインド）`
  - ※ 送信条件：メンション送信元へのメンション返信がない場合
  
- **30分リマインド再送（スレッドに投稿）**  
  - `@対象者 さん、まだ未返信のようです。目安だけでもご共有ください🙏（自動リマインド）`
  - ※ 送信条件：メンション送信元へのメンション返信がない場合
  
- **上長DM（30分時）**  
  - `【エスカレーション】@対象者 さんが未返信です。対象スレッド: <スレッドURL>`
  - ※ 送信条件：メンション送信元へのメンション返信がない場合

※ 口調は柔らかく、圧をかけすぎない表現で統一。

---

## 06. スラッシュコマンド（管理用）
- `/_set_manager @上長`  
  - ワークスペースの上長を設定（上書き）。
- `/_unset_manager`  
  - 上長設定を削除（以後30分時も上長DMは送らない）。
- `/_get_manager`  
  - 現在の上長設定を表示。
- `/_policy`（任意）  
  - 現在のポリシー（10分/30分・夜間抑止の有無など）を表示。

> コマンド名は競合回避のため先頭に `_` を付与。必要に応じて変更可。

---

## 07. スコープ（最小権限）
- `chat:write`（メッセージ投稿）
- `app_mentions:read`（Botメンション受信）
- `channels:history` / `groups:history` / `im:history` / `mpim:history`（返信確認用）
- `commands`（スラッシュコマンド）

**イベント購読**：  
- `message.channels`, `message.groups`, `message.im`, `message.mpim`

---

## 08. データモデル（Firestore）
### Tenant（ワークスペース設定）
- `team_id` : string（主キー）
- `manager_user_id` : string（上長のSlackユーザーID）※KMS暗号化推奨
- `bot_token_secret_name` : string（Secret Managerのキー名）
- `created_at` : int64

### Mention（監視対象）
- `team_id` : string
- `channel_id` : string
- `message_ts` : string（親メッセージTS）
- `mentioned_user_id` : string（対象者）
- `created_at` : int64
- `reminded` : bool（10分通知済）
- `escalated` : bool（30分通知済）

> **保存しない**：メッセージ本文・表示名・メールアドレス（個人情報/機密）。  
> **IDのみ**を保持し、必要な表示はリアルタイムAPIで取得。

---

## 09. 時限ジョブ（Cloud Tasks）
- 予約ジョブ：
  - **10分後** → `/check/remind`  
  - **30分後** → `/check/escalate`
- ペイロード：`team_id`, `channel_id`, `message_ts`, `mentioned_user_id`
- 認証：**OIDC or 共有シークレットヘッダ**でCloud Runの専用エンドポイントのみ許可
- 冪等性：同一キー（team+channel+ts+user）で重複実行が来ても**状態フラグ**で多重投稿を防止

---

## 10. 返信判定ロジック
- `conversations.history` or `conversations.replies` を **`oldest = message_ts`** で取得
- `user == mentioned_user_id` の発言が存在すれば「返信あり」
- **自己返信やBot投稿は無視**

---

## 11. エッジケース / 仕様補足
- **複数対象者**：1メッセージ内で複数ユーザーがメンションされていたら、**対象者ごと**に監視し通知も個別。
- **スレッド/非スレッド**：スレッドが無い場合は、親メッセージに紐づくスレッドとして投稿（`thread_ts = message_ts`）。  
- **夜間/休日の抑止（任意機能）**：JST 22:00–8:00 はリマインドを遅延して朝一送信、などポリシー化可。
- **Botが抜けた/権限不足**：投稿先が無い/権限エラーの場合はログに記録しフェイルセーフ（上長DMだけ送る等）を検討。
- **対象者がすでに退席**：`user_presence`は参照しない（通知だけ丁寧に）。  
- **再送設計**：30分時は「再リマインド + 上長DM」。以降は送らない（初期仕様）。将来、最大回数や間隔は設定化可能。

---

## 12. セキュリティ / プライバシー
- **本文は保存しない**（ID・時刻のみ）
- **トークンはDBに保存しない**：Secret Managerに格納（FirestoreにはSecret名だけ保持）
- **上長IDなど軽機密はKMSで暗号化**して保存
- ログにも**個人名や本文を出力しない**（必要ならIDのみ）
- Slack署名検証（`X-Slack-Signature`）は必須

---

## 13. 失敗時の挙動
- Slack API 429/5xx：指数バックオフで再試行（最大回数は小さめ）
- Firestore/Tasks失敗：リトライまたはデッドレターログ
- 30分時の上長未設定：**上長DMはスキップ**、再リマインドのみ

---

## 14. 非目標（初期リリースでやらない）
- メッセージ本文の解析・要約
- マルチ言語切替（日本語固定）
- 複雑な勤務時間カレンダー連携
- SLA保証（ベストエフォート）

---

## 15. 導入・操作（ユーザー向け要約）
1. 管理者が **Manifest** をSlack公式に貼り付けてアプリ作成 → **Install**
2. Botを使うチャンネルに **/invite @Bot**
3. `/_set_manager @上長` を一度実行
4. 会話で **`@Bot @対象者 〜`** と書く  
   → 返信がなければ **10分でリマインド**、**30分で再送＋上長DM**

---

## 16. 成功基準（受け入れ条件）
- 対象者が返信した場合、**以後のリマインド/上長DMは送られない**
- 10分/30分のタイミングで **正しい文面** が投稿される
- `/_set_manager`, `/_unset_manager`, `/_get_manager` が正しく動く
- 保存されるデータに**個人情報・本文が含まれていない**


Slack Reminder Bot（構成図）

project/
├── cmd/
│   └── main.go
│       └── 🌱 アプリの起動係（設定を読み込んでHTTPサーバーを起動）
│
├── domain/                               🎯 ビジネスルール（純粋な設計）
│   ├── entity.go        → Tenant, Mention の形（データの設計図）　✅
│   ├── repository.go    → Firestoreとの出入りの約束（interface）　✅
│   └── err.go
├── handler/                              🚪 HTTPリクエストの入口
│   ├── events_handler.go    → Slackのメンションイベントを受け取る
│   ├── commands_handler.go  → /_set_manager などスラッシュコマンド処理
│   ├── remind_handler.go    → Cloud Tasks からの10分後リマインド処理
│   ├── escalate_handler.go  → Cloud Tasks からの30分後上長通知処理
│   └── oauth_handler.go     → Slackインストール完了（OAuth）処理
│
├── service/                              🧠 ユースケースの中核ロジック
│   ├── port.go         → SlackPort / TaskPort / SecretPort の約束(interface)　✅
│   ├── model.go        → 内部処理用の軽いデータ型（MentionEventなど）　✅
│   └── reminder_service.go　✅
│       ├── OnMention     → メンション検知 → Firestore保存 → タスク予約　✅
│       ├── CheckRemind   → 10分後に返信がなければリマインド　✅
│       └── CheckEscalate → 30分後も返信なければ再通知 + 上長DM　✅
│
├── dto/                                  📦 外部とのデータ受け渡し箱
│   ├── slack_event.go    → Events API 用
│   ├── slack_command.go  → Slash Command 用
│   └── task_payload.go   → Cloud Tasks 用（team_id, channel_id, ts, user）
│
├── infrastructure/                       ⚙️ 技術の詳細（外部とのやり取り）
│   ├── slack/
│   │   └── client.go       → Slack API呼び出し実装（SlackPort実体）
│   ├── store/
│   │   └── firestore.go    → Firestore保存実装（Repository実体）
│   ├── tasks/
│   │   └── cloudtasks.go   → Cloud Tasksスケジュール実装（TaskPort実体）
│   ├── httpsec/
│   │   └── slack_verify.go → X-Slack-Signature検証（リクエスト改ざん防止）
│   ├── secret/
│   │   └── manager.go      → Secret Manager実装（金庫でトークン管理）
│   └── config/
│       └── env.go          → 🌍 環境変数読込（Config構造体）　✅
│
└── go.mod / go.sum

---

## 15. 返信判定の詳細

### 返信完了の条件

メンション返信（メンション送信元へのメンション付き返信）が必須です。

**例1: リマインド対象**
```
ユーザーA: "@bot 進捗報告お願いします @ユーザーB"
ユーザーB: "８割完了しました"  ← メンション送信元(A)へのメンション @A がない
```
→ **リマインド送信** 📢（CheckRemind/CheckEscalateで通知が行われます）

**例2: リマインド対象外（返信完了判定）**
```
ユーザーA: "@bot 進捗報告お願いします @ユーザーB"
ユーザーB: "８割完了しました @ユーザーA"  ← メンション送信元(A)へのメンション @A がある
```
→ **リマインド送信なし** ✅（CheckRemind/CheckEscalateでリマインドがスキップされます）

### 判定方法

1. **HasUserRepliedWithMention()** メソッドがメンション返信を検査
2. スレッド内で対象ユーザーが送信元ユーザーへメンション（`<@送信元ユーザーID>`）をつけた投稿を検索
3. メンション返信があれば **返信完了**、なければ **未返信**と判定

### データモデル

`TaskPayload` に以下が含まれています：
- `TeamID`: チーム ID
- `ChannelID`: チャンネル ID  
- `MessageTS`: メンションメッセージの ID（スレッド親）
- `UserID`: メンション対象ユーザー ID
- `ParentUserID`: メンション送信元ユーザー ID ← **新規追加**（メンション返信判定に使用）

---
    └── 📜 Goの依存管理ファイル（外部パッケージやバージョン情報）

────────────────────────────
🔁 全体の流れ
Slack → handler → service → infrastructure → Firestore / Cloud Tasks / Slack API

---

## 16. Cloud Run へのデプロイ

### 📦 前提条件
- Google Cloud SDK (`gcloud`) のインストール
- Docker のインストール
- GCP プロジェクトへのアクセス権限

### 🚀 クイックスタート（推奨）

```bash
# デプロイスクリプトを実行（自動ビルド・プッシュ・デプロイ）
./deploy.sh <GCP-PROJECT-ID> [region] [service-name]

# 例
./deploy.sh my-gcp-project asia-northeast1 slack-reminder-bot
```

スクリプトが以下の処理を自動実行します：
1. GCP 認証確認
2. Docker イメージのビルド
3. Container Registry へのプッシュ
4. Cloud Run へのデプロイ
5. ヘルスチェック実行

### 📖 詳細手順

- **クイックスタート**: `DEPLOY_QUICK.md` を参照
- **詳細設定**: `DEPLOY.md` を参照

### 🔐 Secret Manager 設定（初回のみ）

```bash
# Slack 認証情報を GCP Secret Manager に登録
echo -n "xoxb-your-token" | gcloud secrets create slack-bot-token --data-file=-
echo -n "your-signing-secret" | gcloud secrets create slack-signing-secret --data-file=-
```

### ✅ デプロイ確認

```bash
# サービス URL を取得
gcloud run services describe slack-reminder-bot --region asia-northeast1 --format='value(status.url)'

# ヘルスチェック実行
curl {SERVICE_URL}/health
```

### 📊 リソース設定（デフォルト）
- **メモリ**: 512Mi
- **CPU**: 1
- **タイムアウト**: 3600s
- **最大インスタンス**: 100

### 🗑️ クリーンアップ

```bash
# サービス削除
gcloud run services delete slack-reminder-bot --region asia-northeast1

# Container Registry のイメージ削除
gcloud container images delete gcr.io/PROJECT_ID/slack-reminder-bot --quiet
```

---

## 📦 ファイル一覧

| ファイル | 説明 |
|---------|------|
| `Dockerfile` | Cloud Run デプロイ用イメージ定義 |
| `.dockerignore` | Docker ビルド時に除外するファイル |
| `.gcloudignore` | gcloud デプロイ時に除外するファイル |
| `deploy.sh` | ワンコマンド デプロイスクリプト |
| `DEPLOY.md` | デプロイ詳細ドキュメント |
| `DEPLOY_QUICK.md` | デプロイクイックガイド |