// BackendClient+WebSocket.swift
import Combine
import Foundation

extension BackendClient {
    // MARK: - /ws (WebSocket)

    func startWebSocket() {
        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)
        components?.scheme = (baseURL.scheme == "https") ? "wss" : "ws"
        components?.path = "/ws"
        guard let wsURL = components?.url else { return }

        print("🔌 connect websocket:", wsURL.absoluteString)
        let task = session.webSocketTask(with: wsURL)
        webSocketTask = task
        shouldReconnectWebSocket = true
        task.resume()
        receiveWebSocketMessage()
    }

    private func receiveWebSocketMessage() {
        webSocketTask?.receive { [weak self] result in
            guard let self else { return }

            switch result {
            case .failure(let error):
                print("❌ websocket receive error:", error)
                self.reconnectWebSocket()
            case .success(let message):
                self.handleWebSocketMessage(message)
                // 続けて次のメッセージも待ち受け
                self.receiveWebSocketMessage()
            }
        }
    }

    private func handleWebSocketMessage(_ message: URLSessionWebSocketTask.Message) {
        let text: String?
        switch message {
        case .string(let str):
            text = str
        case .data(let data):
            text = String(data: data, encoding: .utf8)
        @unknown default:
            text = nil
        }

        guard let raw = text else {
            print("⚠️ websocket message not utf8")
            return
        }

        print("🧭 ws raw:", raw)
        guard let jsonData = raw.data(using: .utf8) else { return }

        do {
            let decoder = JSONDecoder()
            decoder.keyDecodingStrategy = .convertFromSnakeCase
            let payload = try decoder.decode(WSPayload.self, from: jsonData)
            let type = payload.type ?? ""

            // WSでもARActionが来る場合は処理
            print("🔌 WS payload received - type: \(type), arActions: \(payload.arActions?.count ?? 0)")
            self.handleARActions(payload.arActions)

            // 台本TTS/SSEで扱うステップでは、WS側のtext/audioは無視する（言語混在・二重再生防止）
            if shouldUseSSEAudio(step: payload.step) {
                print("🔇 ignore WS payload during SSE step type=\(type) step=\(payload.step ?? "nil")")
                return
            }

            switch type {
            case "audio":
                handleAudioPayload(payload)

            case "text", "transcript", "message":
                handleTextPayload(payload)

            default:
                // type が無い/未知でも text があれば表示に反映する
                handleFallbackPayload(payload)
            }
        } catch {
            print("websocket decode error:", error)
        }
    }

    // MARK: - Payload handlers

    /// 音声ペイロードの処理
    private func handleAudioPayload(_ payload: WSPayload) {
        let payloadStep = payload.step

        // ✅ Lesson中は常にSSE/TTSで再生する想定なので、WS音声は無視する
        if isLessonActive {
            print("🔇 ignore WS audio during lesson (step=\(payloadStep ?? "nil"))")
            return
        }

        // 🔴 スクリプトTTSはSSEで再生するため、WSでは再生しない
        guard !shouldUseSSEAudio(step: payloadStep) else {
            // TTS/SSE 用ステップの場合：
            // WS 側の audio は無視し、テキストだけ（もしあれば）即時反映
            if let t = payload.text, !t.isEmpty {
                DispatchQueue.main.async {
                    self.currentMessage = t
                    self.messages = [
                        HistoryItem(
                            role: "transcript",
                            content: t,
                            sessionId: nil,
                            step: payload.step,
                            timestamp: nil
                        )
                    ]
                }
            }
            return
        }

        // 🎯 Realtime/WS 音声の場合：
        // - "text" メッセージで pendingTranscript に貯めた内容があれば、それを再生開始タイミングで表示
        // - もし "audio" 側にも text が入ってきたら、それを pendingTranscript として使ってもOK
        if let t = payload.text, !t.isEmpty, self.pendingTranscript == nil {
            self.pendingTranscript = (text: t, step: payload.step)
        }

        // audioChunk がなく、final だけ来た場合のフォールバック
        let base64 = payload.audioChunk
        let isFinal = payload.audioDone == true

        guard let base64, !base64.isEmpty else {
            if isFinal {
                handleAudioPlaybackFinished(step: payloadStep)
            }
            return
        }

        // 🔔 一旦ここで pendingTranscript を画面に反映しておく
        // （AudioStreamPlayer.onPlaybackStart を使う場合は、そちらから呼んでもOK）
        flushPendingTranscript(forStep: payloadStep)

        print("🔊 WS enqueue audio step=\(payloadStep ?? "nil") final=\(isFinal)")
        audioStreamPlayer?.enqueuePCM(
            base64: base64,
            isFinalChunk: isFinal,
            onPlaybackComplete: { [weak self] in
                self?.handleAudioPlaybackFinished(step: payloadStep)
            }
        )
    }

    /// テキストペイロードの処理
    private func handleTextPayload(_ payload: WSPayload) {
        guard let t = payload.text, !t.isEmpty else { return }

        let useSSE = shouldUseSSEAudio(step: payload.step)

        if useSSE {
            // 📖 台本TTSなど、SSE側でしゃべるステップ：
            //    これまで通りテキストは即表示でOK
            DispatchQueue.main.async {
                self.currentMessage = t
                self.messages = [
                    HistoryItem(
                        role: "transcript",
                        content: t,
                        sessionId: nil,
                        step: payload.step,
                        timestamp: nil
                    )
                ]
            }
        } else {
            // 🎧 Realtime/WS 音声ステップ：
            //    ここでは currentMessage を更新せず、pendingTranscript に貯めておく
            self.pendingTranscript = (text: t, step: payload.step)

            // ログとしては残したい場合は messages にだけ積むのもアリ
            // （UI のメインテキストはまだ切り替えない）
            self.messages = [
                HistoryItem(
                    role: "transcript",
                    content: t,
                    sessionId: nil,
                    step: payload.step,
                    timestamp: nil
                )
            ]
        }
    }

    /// type 不明 or 空の場合のフォールバック処理
    private func handleFallbackPayload(_ payload: WSPayload) {
        guard let t = payload.text, !t.isEmpty else {
            print("ℹ️ ws unknown type:", payload.type ?? "(nil)")
            return
        }

        let useSSE = shouldUseSSEAudio(step: payload.step)

        if useSSE {
            DispatchQueue.main.async {
                self.currentMessage = t
                self.messages = [
                    HistoryItem(
                        role: "transcript",
                        content: t,
                        sessionId: nil,
                        step: payload.step,
                        timestamp: nil
                    )
                ]
            }
        } else {
            // Realtime/WS 音声かもしれないので、ひとまず pending に入れておく
            self.pendingTranscript = (text: t, step: payload.step)
            self.messages = [
                HistoryItem(
                    role: "transcript",
                    content: t,
                    sessionId: nil,
                    step: payload.step,
                    timestamp: nil
                )
            ]
        }
    }

    /// pendingTranscript があれば、それを currentMessage に反映する
    private func flushPendingTranscript(forStep step: String?) {
        guard let pending = pendingTranscript else { return }

        // step が一致しているか軽く確認（一致しないなら無視してもいい）
        if let s = step, let ps = pending.step, s != ps {
            // 必要ならここで return してもOK
            // 今回は「とりあえず表示しちゃう」方針でそのまま続行
        }

        DispatchQueue.main.async {
            self.currentMessage = pending.text
            self.messages = [
                HistoryItem(
                    role: "transcript",
                    content: pending.text,
                    sessionId: nil,
                    step: pending.step ?? step,
                    timestamp: nil
                )
            ]
        }
        pendingTranscript = nil
    }

    // MARK: - Reconnect

    private func reconnectWebSocket() {
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        guard shouldReconnectWebSocket else { return }
        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
            self.startWebSocket()
        }
    }

    // MARK: - WebSocket Payload Model

    // WebSocket 受信 JSON 用の struct
    private struct WSPayload: Decodable {
        let type: String?               // "text" / "audio" / "transcript" / "message" など
        let text: String?
        let step: String?
        let arActions: [ARAction]?
        let userName: String?
        let audioChunk: String?         // base64-encoded PCM16
        let audioDone: Bool?
        let sampleRate: Int?
        let channels: Int?
        let bytesPerSample: Int?
    }
}
