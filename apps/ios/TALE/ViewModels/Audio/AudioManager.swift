import Foundation
import AVFoundation
import Combine

/// シンプルな BGM 管理用スタブ
final class AudioManager: ObservableObject {
    /// ObservableObject の更新通知（必要に応じて send() する）
    let objectWillChange = ObservableObjectPublisher()
    private var player: AVAudioPlayer?
    private var ttsPlayer: AVAudioPlayer?

    func configurePlaybackSession() {
        let session = AVAudioSession.sharedInstance()
        do {
            try session.setCategory(.playAndRecord,
                                    mode: .voiceChat,
                                    options: [.defaultToSpeaker, .allowBluetoothHFP, .allowBluetoothA2DP, .mixWithOthers])
            try session.setActive(true, options: .notifyOthersOnDeactivation)
        } catch {
            debugLog("AudioManager session error:", error)
        }
    }

    func startBGM() {
        // すでに再生中なら何もしない
        if let player, player.isPlaying { return }

        // バンドル内の BGM を探す（ファイル名は 456_BPM80.mp3 を想定）
        let candidates = [
            ("General_bgm", "mp3"),
            ("BGM", "mp3")
        ]

        var url: URL?
        for (name, ext) in candidates {
            if let found = Bundle.main.url(forResource: name, withExtension: ext) {
                url = found
                break
            }
        }

        guard let bgmURL = url else {
            debugLog("AudioManager: BGM file not found in bundle.")
            return
        }

        do {
            let newPlayer = try AVAudioPlayer(contentsOf: bgmURL)
            newPlayer.numberOfLoops = -1  // 無限ループ
            newPlayer.volume = 0.3
            newPlayer.prepareToPlay()
            newPlayer.play()
            player = newPlayer
        } catch {
            debugLog("AudioManager: failed to start BGM:", error)
        }
    }

    func stopBGM() {
        player?.stop()
    }

    func playBGM(named name: String, ext: String = "mp3", volume: Float = 0.3) {
        guard let url = Bundle.main.url(forResource: name, withExtension: ext) else {
            debugLog("AudioManager: BGM file not found in bundle:", "\(name).\(ext)")
            return
        }
        do {
            let newPlayer = try AVAudioPlayer(contentsOf: url)
            newPlayer.numberOfLoops = -1
            newPlayer.volume = volume
            newPlayer.prepareToPlay()
            newPlayer.play()
            player = newPlayer
        } catch {
            debugLog("AudioManager: failed to start BGM:", error)
        }
    }

    /// 任意の音声ファイルURLを再生（TTSなど）
    func playVoice(from url: URL) {
        do {
            let newPlayer = try AVAudioPlayer(contentsOf: url)
            newPlayer.volume = 1.0
            newPlayer.prepareToPlay()
            newPlayer.play()
            ttsPlayer = newPlayer
        } catch {
            debugLog("AudioManager: failed to play voice:", error)
        }
    }
}
