# TALE Backend - AI Voice Chat System

Go言語とAI APIを使用した音声会話システムです。マイクから音声入力を受け取り、AIが音声で応答します。

## 機能

- **リアルタイム音声会話**: OpenAI Realtime API または Gemini 1.5 + TTS を使用
- **音声認識**: OpenAI Whisper APIを使用して音声をテキストに変換
- **AI会話**: GPT-4 / Gemini 1.5 を使用して自然な会話を生成
- **音声合成**: OpenAI TTS APIを使用してテキストを音声に変換
- **レッスンモード**: PostgreSQLに保存されたレッスンコンテンツの再生
- **ARアクション**: iOS ARアプリへのアニメーション指示
- **会話履歴**: 文脈を保持した連続的な会話が可能

## ポートフォリオ向け設計方針

レッスン進行のルールは `internal/lesson` に置き、HTTPやPostgreSQLなどの外部I/Oから分離しています。
`lesson.Session` はレッスンの開始、現在ステップ、次ステップへの進行を担当し、外部サービスなしで単体テストできます。
APIプロバイダーや永続化方式が変わっても、レッスンの進行ルールを独立して検証できる構成を目指しています。

このリポジトリには、APIキー、実際の会話履歴、IDE固有の設定、ビルド生成物を含めません。
ローカル実行時の会話履歴は `data/conversation_history.json` に保存されますが、Git管理対象外です。

## アーキテクチャ

### 2つの実行モード

| モード | コマンド | 特徴 |
|--------|----------|------|
| **OpenAI Realtime** | `make run` | 低レイテンシー、WebSocket双方向通信 |
| **Gemini + TTS** | `make run-gemini` | 低コスト、VADによる音声検出 |

### 処理フロー

**OpenAI Realtime API モード:**
```
マイク → OpenAI Realtime API（音声認識 + 応答生成 + 音声合成）→ スピーカー/iOS
```

**Gemini + TTS モード:**
```
マイク → VAD → Whisper（音声認識）→ Gemini 1.5（応答生成）→ TTS（音声合成）→ スピーカー/iOS
```

## 必要要件

### システム要件

- Go 1.25以上（`go.mod`に準拠）
- PortAudio ライブラリ
- PostgreSQL (レッスン機能を使う場合)
- マイクとスピーカー

### macOS

```bash
brew install portaudio
```

### Linux (Ubuntu/Debian)

```bash
sudo apt-get install portaudio19-dev
```

## セットアップ

### 1. 依存パッケージのインストール

```bash
go mod download
```

### 2. 環境変数の設定

テンプレートから `.env` ファイルを作成し、利用するモードのAPIキーを設定します:

```bash
cp .env.example .env
```

### 3. PostgreSQL起動 (オプション)

```bash
make db-up
```

## 使い方

### Makeコマンド

```bash
# OpenAI Realtime API モード
make run

# Gemini + TTS モード
make run-gemini

# ビルド
make build

# クリーンアップ
make clean

# PostgreSQL操作
make db-up        # 起動
make db-down      # 停止
make db-restart   # 再起動
make db-reset     # データリセット
make db-psql      # SQL接続
```

### APIエンドポイント

サーバー起動後、以下のエンドポイントが利用可能:

| エンドポイント | メソッド | 説明 |
|---------------|---------|------|
| `/ws` | WebSocket | リアルタイム音声/テキスト通信 |
| `/health` | GET | ヘルスチェック |
| `/conversation/start` | POST | 会話セッション開始 |
| `/conversation/state` | GET | 現在の状態取得 |
| `/conversation/history` | GET | 会話履歴取得 |
| `/conversation/transcript/stream` | GET (SSE) | テキストストリーミング |
| `/conversation/audio/stream` | GET (SSE) | 音声ストリーミング |
| `/lessons` | GET | レッスン一覧 |
| `/lessons/:id` | GET | レッスン詳細 |
| `/lessons/:id/start` | POST | レッスン開始 |
| `/lessons/:id/next` | POST | 次のステップへ |
| `/rule/items` | GET/POST | 会話ルールの一覧取得・作成 |
| `/rule/items/:id` | PUT/DELETE | 会話ルールの更新・削除 |

## プロジェクト構造

```
services/api/
├── cmd/
│   ├── voice-chat/
│   │   └── main.go              # OpenAI Realtime API モード
│   └── gemini-test/
│       └── main.go              # Gemini + TTS モード
├── internal/
│   ├── ar/
│   │   └── server.go            # ARServer (WebSocketサーバー)
│   ├── audio/
│   │   ├── player.go            # 音声再生
│   │   └── recorder.go          # 音声録音
│   ├── realtime/
│   │   ├── client.go            # OpenAI Realtime API クライアント
│   │   ├── gemini_client.go     # Gemini 1.5 + TTS クライアント
│   │   └── state.go             # ステートマシン & スクリプト
│   ├── lesson/
│   │   ├── model.go             # レッスンのデータモデル
│   │   ├── repository.go        # レッスンDBリポジトリ
│   │   └── session.go           # レッスン進行のドメインロジック
│   └── api/
│       ├── swift.go             # iOS用REST API
│       └── lesson.go            # レッスンAPI
├── configs/
│   └── rules.json               # ルール定義
├── data/                         # ローカル実行時データ（Git管理対象外）
├── docs/
│   └── STATE_MACHINE_README.md  # ステートマシンの詳細
├── README.md                    # このファイル
├── Makefile
├── docker-compose.yml
├── go.mod
└── go.sum
```

## API使用について

### OpenAI API

| API | モデル | 用途 |
|-----|--------|------|
| Realtime API | gpt-realtime | リアルタイム音声会話（環境変数で変更可能） |
| Whisper API | whisper-1 | 音声認識 |
| TTS API | tts-1 | 音声合成 |
| Chat API | gpt-4o-mini | レッスン翻訳 |

### Gemini API

| API | モデル | 用途 |
|-----|--------|------|
| Generative AI | gemini-1.5-flash | テキスト会話 |

モデルの提供状況や料金は変更されるため、実行前に[OpenAI公式のモデル一覧](https://developers.openai.com/api/docs/models)と各プロバイダーの公式情報を確認してください。

## トラブルシューティング

### PortAudio のエラー

```
Failed to initialize portaudio
```

PortAudioがインストールされていない可能性があります。上記の「システム要件」を参照してインストールしてください。

### マイクが認識されない

システムのマイク権限を確認してください。macOSの場合:
システム環境設定 > セキュリティとプライバシー > マイク で許可が必要です。

### OpenAI API エラー

- API キーが正しく設定されているか確認
- API キーに十分なクレジットがあるか確認
- インターネット接続を確認

### Gemini API エラー

- GEMINI_API_KEY が設定されているか確認
- Google AI Studioでクォータを確認: https://aistudio.google.com/apikey

### broken pipe エラー

WebSocket接続がタイムアウトで切断されました。長時間アイドル状態が続くと発生します。

## ライセンス

[MIT License](../../LICENSE)。メディア資産には別条件が適用されます。

## 開発者向け情報

### 音声形式

| 項目 | OpenAI Realtime | Gemini + TTS |
|------|-----------------|--------------|
| 入力サンプルレート | 24kHz | 24kHz |
| 出力サンプルレート | 24kHz | 24kHz |
| フォーマット | PCM16 | PCM16 |
| チャンネル | モノラル | モノラル |

### カスタマイズ

- **システムプロンプト**: `getRealtimeSystemPrompt()` / `getSystemPrompt()`
- **音声**: TTS の voice パラメータ (`coral`, `alloy`, `echo`, `fable`, `onyx`, `nova`, `shimmer`)
- **VAD設定**: `gemini_client.go` の `SilenceThreshold`, `SilenceDurationMs`
