// BackendClient+SSE.swift
import Combine
import Foundation

extension BackendClient {

    // MARK: - /conversation/transcript/stream (SSE)

    func startHistoryStream() {
        transcriptSSEStartCount += 1
        debugLog("📡 startHistoryStream called (\(transcriptSSEStartCount))")
        // 既に接続済みなら何もしない（多重接続防止）
        if historyTask != nil {
            debugLog("📡 startHistoryStream skipped (already connected)")
            return
        }

        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)
        components?.path = "/conversation/transcript/stream"
        guard let streamURL = components?.url else { return }

        var request = URLRequest(url: streamURL)
        request.timeoutInterval = .infinity

        historyBuffer.removeAll()
        debugLog("🔌 connect transcript SSE:", streamURL.absoluteString)
        historyTask = session.dataTask(with: request)
        historyTask?.resume()
    }

    func handleHistoryEvent(_ data: Data) {
            guard let content = String(data: data, encoding: .utf8) else { return }

            guard let line = content
                .split(separator: "\n")
                .first(where: { $0.hasPrefix("data:") })
            else { return }

            let jsonString = line.dropFirst("data:".count)
                .trimmingCharacters(in: .whitespaces)
            guard let jsonData = jsonString.data(using: .utf8) else { return }

            struct TranscriptPayload: Decodable {
                let text: String?
                let step: String?
                let done: Bool?
                let arActions: [ARAction]?
            }

            do {
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                let payload = try decoder.decode(TranscriptPayload.self, from: jsonData)

                debugLog("📡 SSE transcript event - step: \(payload.step ?? "nil"), arActions: \(payload.arActions?.count ?? 0)")
                if let arActions = payload.arActions, !arActions.isEmpty {
                    debugLog("📡 SSE transcript received ARActions: \(arActions)")
                }

                DispatchQueue.main.async {
                    if let text = payload.text, !text.isEmpty {
                        self.currentMessage = text
                        // 履歴も最新1件を保持
                        self.messages = [
                            HistoryItem(
                                role: "transcript",
                                content: text,
                                sessionId: nil,
                                step: payload.step,
                                timestamp: nil
                            )
                        ]
                    }
                    if let stepString = payload.step,
                       let step = ConversationStep(rawValue: stepString) {
                        self.currentStep = step
                    }
                }

                // 履歴SSEでもARActionが来る場合は処理
                self.handleARActions(payload.arActions)
            } catch {
                debugLog("transcript SSE decode error:", error)
            }
        }


    private struct TranscriptPayload: Decodable {
        let text: String?
        let step: String?
        let done: Bool?
    }

    // MARK: - /conversation/audio/stream (SSE)

    func startAudioStream() {
        audioSSEStartCount += 1
        debugLog("📡 startAudioStream called (\(audioSSEStartCount))")
        // 既に接続済みなら何もしない（多重接続防止）
        if audioTask != nil {
            debugLog("📡 startAudioStream skipped (already connected)")
            return
        }

        var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false)
        components?.path = "/conversation/audio/stream"
        guard let streamURL = components?.url else { return }

        var request = URLRequest(url: streamURL)
        request.timeoutInterval = .infinity

        audioBuffer.removeAll()
        debugLog("🔌 connect audio SSE:", streamURL.absoluteString)
        audioTask = session.dataTask(with: request)
        audioTask?.resume()
    }

    func handleAudioEvent(_ data: Data) {
            guard let content = String(data: data, encoding: .utf8) else { return }
            guard let line = content
                .split(separator: "\n")
                .first(where: { $0.hasPrefix("data:") })
            else { return }

            let jsonString = line.dropFirst("data:".count)
                .trimmingCharacters(in: .whitespaces)
            guard let jsonData = jsonString.data(using: .utf8) else { return }

            struct AudioPayload: Decodable {
                let type: String?
                let text: String?
                let step: String?
                let arActions: [ARAction]?
                let userName: String?
                let audioChunk: String?
                let audioDone: Bool?
                let sampleRate: Int?
                let channels: Int?
                let bytesPerSample: Int?
            }

            do {
                let decoder = JSONDecoder()
                decoder.keyDecodingStrategy = .convertFromSnakeCase
                let payload = try decoder.decode(AudioPayload.self, from: jsonData)

                let isFinal = payload.audioDone == true
                let payloadStep = payload.step
                debugLog("🔊 SSE audio event step=\(payloadStep ?? "nil") final=\(isFinal) arActions=\(payload.arActions?.count ?? 0)")

                if let arActions = payload.arActions, !arActions.isEmpty {
                    debugLog("🔊 SSE received ARActions: \(arActions)")
                }

                // SSEから受信したARActionを処理（音声方式に関係なく適用）
                self.handleARActions(payload.arActions)

                // スクリプトTTSのみ SSE 音声を再生
                guard shouldUseSSEAudio(step: payloadStep) else { return }

                if let syncedText = payload.text, !syncedText.isEmpty {
                    DispatchQueue.main.async {
                        self.currentMessage = syncedText
                    }
                }

                // 🔴 累積せず「来たチャンクをそのままキューへ」
                if let base64 = payload.audioChunk, !base64.isEmpty {
                    debugLog("🔊 SSE enqueue audio step=\(payloadStep ?? "nil") final=\(isFinal)")
                    audioStreamPlayer?.enqueuePCM(
                        base64: base64,
                        isFinalChunk: isFinal,
                        onPlaybackComplete: { [weak self] in
                            self?.handleAudioPlaybackFinished(
                                step: payloadStep
                            )
                        }
                    )
                } else if isFinal {
                    debugLog("🔊 Received audioDone=true (no audio data)")
                    // 音声なしで end だけ来た場合もステップ進行をトリガー
                    handleAudioPlaybackFinished(
                        step: payloadStep
                    )
                }

            } catch {
                debugLog("audio SSE decode error:", error)
            }
        }

    private struct AudioPayload: Decodable {
        let type: String?
        let text: String?
        let step: String?
        let arActions: [ARAction]?
        let userName: String?
        let audioChunk: String?
        let audioDone: Bool?
        let sampleRate: Int?
        let channels: Int?
        let bytesPerSample: Int?
    }
}
