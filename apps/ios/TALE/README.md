# TALE

TALEは拡張現実（AR）を活用した日本語学習アプリケーションです。インタラクティブなレッスン、クエストマップ、居酒屋体験などを通じて、没入型の言語学習体験を提供します。

## 主な機能

- **AR学習体験**: ARKit/RealityKitを使用した3D空間での日本語学習
- **インタラクティブレッスン**: ステップバイステップの日本語レッスン
- **クエストシステム**: ゲーム感覚で学べるクエストマップ
- **居酒屋AR体験**: 実践的な会話練習ができるレストランシミュレーション
- **音声ストリーミング**: リアルタイムの音声フィードバック
- **多言語対応**: 複数言語でのUI表示に対応
- **リアルタイム通信**: WebSocketとSSEによるバックエンドとの連携

## 技術スタック

- **言語**: Swift
- **UI フレームワーク**: SwiftUI
- **AR**: ARKit, RealityKit
- **ネットワーク**: WebSocket, Server-Sent Events (SSE)
- **音声**: AVFoundation, AVKit
- **アニメーション**: Lottie (WebView経由)
- **3Dモデル**: USDZ, RealityComposer Pro

## プロジェクト構造

```
TALE/
├── AppDelegate.swift                     # アプリケーションのエントリーポイント
├── Models/                               # データモデル
│   ├── APIModels.swift                   # API通信用のモデル
│   └── ConversationStep.swift            # 会話ステップの定義
├── ViewModels/                           # ビューモデル層
│   ├── AR/                               # AR関連のロジック
│   │   ├── ARSceneController.swift       # ARシーンコントローラー
│   │   └── Animation/                    # アニメーション用RealityKitパッケージ
│   ├── Audio/                            # 音声管理
│   │   ├── AudioManager.swift            # BGM・効果音管理
│   │   ├── AudioStreamPlayer.swift       # ストリーミング再生
│   │   └── AVAudioSessionManager.swift   # オーディオセッション管理
│   ├── Backend/                          # バックエンド通信
│   │   ├── BackendClient.swift           # APIクライアント
│   │   ├── ConversationController.swift  # 会話制御
│   │   ├── SSEManager.swift              # Server-Sent Events管理
│   │   ├── WebSocketManager.swift        # WebSocket管理
│   │   └── StatePollingManager.swift     # 状態ポーリング
│   └── Language/                         # 言語設定
│       └── LanguageStore.swift           # 言語ストア
├── Views/                                # ビュー層
│   ├── MainView/                         # メインビュー
│   │   └── ContentView.swift             # メインコンテンツビュー
│   ├── AR/                               # ARビュー
│   │   ├── ARViewContainer.swift         # ARビューコンテナ
│   │   ├── LessonARView.swift            # レッスン用ARビュー
│   │   ├── IzakayaARView.swift           # 居酒屋ARビュー
│   │   └── QuestMapARView.swift          # クエストマップARビュー
│   ├── Components/                       # 再利用可能なコンポーネント
│   │   ├── Buttons/                      # ボタンコンポーネント
│   │   ├── Cards/                        # カードコンポーネント
│   │   ├── Controls/                     # コントロール類
│   │   ├── Movement/                     # 移動パッド
│   │   └── Web/                          # WebView (Lottie, Sketchfab)
│   ├── Overlays/                         # オーバーレイ画面
│   │   ├── Opening/                      # オープニング関連
│   │   ├── Lesson/                       # レッスン関連
│   │   ├── Quest/                        # クエスト関連
│   │   ├── Restaurant/                   # レストラン関連
│   │   └── Help/                         # ヘルプ・サポート
│   └── Extensions/                       # Viewの拡張
└── Media/                                # メディアリソース
    └── USDZ/                             # 3Dモデル
        └── Firework/                     # 花火エフェクト
```

## 要件

- iOS 16.0以降
- Xcode 15.0以降
- ARKit対応デバイス（iPhone XS以降推奨）

## セットアップ

1. リポジトリをクローン:
```bash
git clone <repository-url>
cd tale-frontend
```

2. Xcodeでプロジェクトを開く:
```bash
open TALE.xcodeproj
```

3. 必要に応じてTeamとBundle Identifierを設定

4. ビルドして実行

## ビルド方法

### 開発ビルド
1. Xcodeでプロジェクトを開く
2. シミュレーターまたは実機を選択
3. `Command + R` でビルド＆実行

### リリースビルド
1. Product > Archive を選択
2. Archiveが完了したら、Distributeを選択
3. 配布方法を選択（App Store、Ad Hoc、Enterpriseなど）

## アーキテクチャ

このプロジェクトはMVVMアーキテクチャパターンを採用しています：

- **Models**: データ構造とビジネスロジック
- **Views**: SwiftUIによるUI実装
- **ViewModels**: ViewとModelの橋渡し、状態管理
- **Extensions**: Viewの機能拡張（UI、アクション、言語対応など）

## 主要な依存関係

- ARKit: AR機能の実装
- RealityKit: 3Dコンテンツのレンダリングとアニメーション
- AVFoundation: 音声・動画の再生
- Combine: リアクティブプログラミング
- SwiftUI: 宣言的UIフレームワーク

## ライセンス

[ライセンス情報を追加してください]

## 開発者

[開発者情報を追加してください]
