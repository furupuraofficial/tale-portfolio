# Contributing

TALEへの改善提案を歓迎します。変更は目的を1つに絞った小さなブランチで作成してください。

## 開発フロー

1. `main` から作業ブランチを作成します。
2. 関連するテストとドキュメントを更新します。
3. 下記の検証を実行し、Pull Requestに結果を記載します。

```bash
cd services/api && go test ./...
cd apps/admin-web && npm ci && npm run lint && npm test && npm run build
```

iOSを変更した場合は、[iOS README](apps/ios/TALE/README.md)の手順で端末向けビルドと単体テストも実行してください。

## メディア資産

3Dモデル、画像、動画、音声を追加するPull Requestには、作者、配布元URL、ライセンス、変更の有無を記載してください。再配布条件が確認できない素材は追加しないでください。詳細は[ASSET_NOTICES.md](ASSET_NOTICES.md)を参照してください。

## 秘密情報

APIキー、認証情報、実際の会話履歴、個人情報をコミットしないでください。ローカル設定には各ディレクトリの `.env.example` をコピーした `.env` を使用します。
