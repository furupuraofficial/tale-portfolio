import AVFoundation

/// 共通のオーディオセッション初期化ヘルパー
enum AppAudioSession {
    /// スピーカーからAI音声を再生できるよう、録音も許可した再生設定にする
    static func configureForAIVoicePlayback() {
        let session = AVAudioSession.sharedInstance()
        do {
            try session.setCategory(.playAndRecord,
                                    mode: .voiceChat,
                                    options: [.defaultToSpeaker, .allowBluetooth, .allowBluetoothA2DP, .mixWithOthers])
            try session.setActive(true, options: .notifyOthersOnDeactivation)
        } catch {
            print("🔊 AVAudioSession setup failed:", error)
        }
    }
}
