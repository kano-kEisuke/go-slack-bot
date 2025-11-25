# ビルドステージ
FROM golang:1.25.0 AS builder

WORKDIR /workspace

# キャッシュを効率化するため go.mod と go.sum を先にコピー
COPY go.mod go.sum ./

# 依存関係をダウンロード
RUN go mod download

# ソースコードをコピー
COPY . .

# アプリケーションをビルド
# CGO_ENABLED=0 で静的リンクし、他のライブラリに依存しないバイナリを作成
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /workspace/slack-reminder-bot ./project/cmd/main.go

# 実行ステージ
FROM gcr.io/distroless/base-debian12

# タイムゾーン設定用のカレンダーをコピー
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# ビルドステージから実行ファイルをコピー
COPY --from=builder /workspace/slack-reminder-bot /slack-reminder-bot

# ヘルスチェック用ポート
EXPOSE 8080

# Cloud Run はデフォルトで $PORT 環境変数を使用 (デフォルト 8080)
ENV PORT=8080

# アプリケーション起動
ENTRYPOINT ["/slack-reminder-bot"]
