import Foundation
import AVFoundation
import Combine

final class AudioStreamPlayer {
    private let engine = AVAudioEngine()
    private let player = AVAudioPlayerNode()

    /// PCMを詰めるフォーマット（24kHz / 2ch / Float32）
    private let playbackFormat: AVAudioFormat

    /// 「このセリフの再生が始まる直前」に1回だけ呼ばれる
    var onPlaybackStart: (() -> Void)?

    /// いま何かしらのセリフを再生中かどうか（自前フラグ）
    private var isSpeaking = false

    init?() {
        engine.attach(player)

        // ★ 再生は 24kHz / 2ch（L=R）で扱う
        guard let format = AVAudioFormat(
            commonFormat: .pcmFormatFloat32,
            sampleRate: 24_000,
            channels: 2,
            interleaved: false
        ) else {
            return nil
        }

        playbackFormat = format

        // ★ 2ch フォーマットで明示接続
        engine.connect(player, to: engine.mainMixerNode, format: playbackFormat)

        debugLog("🔊 AudioSession sampleRate:", AVAudioSession.sharedInstance().sampleRate)
        let mixerFormat = engine.mainMixerNode.outputFormat(forBus: 0)
        debugLog("🔊 mixerFormat:", mixerFormat.sampleRate, "Hz,", mixerFormat.channelCount, "ch")
        debugLog("🔊 playbackFormat:", playbackFormat.sampleRate, "Hz,", playbackFormat.channelCount, "ch")
    }

    private func startEngineIfNeeded() {
        if !engine.isRunning {
            do {
                try engine.start()
            } catch {
                debugLog("engine start error:", error)
            }
        }
        if !player.isPlaying {
            player.play()
        }
    }

    func stop() {
        // scheduleBuffer を止めてキューも破棄する
        player.stop()
        engine.stop()
        isSpeaking = false
    }

    /// Backend から来た base64 PCM (24kHz / 16bit / mono) を再生
    /// - Parameters:
    ///   - base64: PCMデータ（mono）
    ///   - isFinalChunk: セリフの最後のチャンクかどうか
    ///   - onPlaybackComplete: isFinalChunk の再生完了で呼ばれる
    func enqueuePCM(
        base64: String,
        isFinalChunk: Bool = false,
        onPlaybackComplete: (() -> Void)? = nil
    ) {
        guard let data = Data(base64Encoded: base64), !data.isEmpty else {
            return
        }
        enqueuePCMData(
            data,
            isFinalChunk: isFinalChunk,
            onPlaybackComplete: onPlaybackComplete
        )
    }

    func enqueuePCMData(
        _ data: Data,
        isFinalChunk: Bool = false,
        onPlaybackComplete: (() -> Void)? = nil
    ) {
        guard !data.isEmpty else { return }

        // 16bit mono (2byte / sample)
        let sampleCount = data.count / MemoryLayout<Int16>.size
        let frameCount = AVAudioFrameCount(sampleCount)

        guard let buffer = AVAudioPCMBuffer(
            pcmFormat: playbackFormat,
            frameCapacity: frameCount
        ) else {
            return
        }
        buffer.frameLength = frameCount

        data.withUnsafeBytes { raw in
            guard let src = raw.bindMemory(to: Int16.self).baseAddress else { return }
            guard
                let chans = buffer.floatChannelData,
                buffer.format.channelCount >= 2
            else { return }

            let left  = chans[0]
            let right = chans[1]
            for i in 0..<sampleCount {
                let v = Float(src[i]) / Float(Int16.max)
                left[i]  = v
                right[i] = v  // L=R に複製
            }
        }

        debugLog("🎙 enqueuePCMData bytes:", data.count)

        // 「このセリフの最初のチャンクかどうか」をここで判定
        let isFirstChunkOfUtterance = !isSpeaking
        if isFirstChunkOfUtterance {
            isSpeaking = true
        }

        startEngineIfNeeded()

        if isFinalChunk {
            player.scheduleBuffer(
                buffer,
                completionCallbackType: .dataPlayedBack
            ) { [weak self] _ in
                // 最後のチャンクの再生が終わったらフラグを下ろす
                self?.isSpeaking = false
                onPlaybackComplete?()
            }
        } else {
            player.scheduleBuffer(buffer, completionHandler: nil)
        }

        // buffer をキューに積み終わったタイミングで「再生開始」を通知
        if isFirstChunkOfUtterance {
            DispatchQueue.main.async { [weak self] in
                self?.onPlaybackStart?()
            }
        }
    }
}
