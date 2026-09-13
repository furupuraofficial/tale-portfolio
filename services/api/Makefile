.PHONY: build run run-gemini clean db-up db-down db-restart db-shell db-reset db-psql

# ビルド (直下に tale-backend バイナリを作成)
build:
	go build -o tale-backend ./cmd/voice-chat

# 実行 (OpenAI Realtime API)
run:
	go run ./cmd/voice-chat

# 実行 (Gemini Live API)
run-gemini:
	go run ./cmd/gemini-test

# クリーンアップ
clean:
	rm -f tale-backend voice-chat

# PostgreSQL起動
db-up:
	docker-compose up -d

# PostgreSQL停止
db-down:
	docker-compose down

# PostgreSQL再起動
db-restart:
	docker-compose down && docker-compose up -d

# PostgreSQLコンテナに入る
db-shell:
	docker exec -it tale_postgres bash

# PostgreSQL リセット（データ削除して再作成）
db-reset:
	docker-compose down -v && docker-compose up -d

# PostgreSQLに直接接続
db-psql:
	docker exec -it tale_postgres psql -U tale -d tale