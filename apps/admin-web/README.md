# TALE Admin Web

TALEの会話ルールと、それに対応するARアクションを管理するReact製のWeb UIです。

## 機能

- 会話ルールの一覧表示と検索
- ルールの作成・編集・削除
- 発話に合わせて実行するARアクションの編集
- APIエラーと保存状態のフィードバック

## セットアップ

```bash
cp .env.example .env
npm ci
npm start
```

標準では `http://localhost:8080` のBackend APIへ接続します。接続先は `REACT_APP_API_BASE` で変更できます。

## コマンド

```bash
npm test -- --watchAll=false
npm run build
```

## API

管理画面は `/rule/items` 以下のCRUD APIを利用します。APIの仕様と実装状況は[Backend README](../../services/api/README.md)を参照してください。
