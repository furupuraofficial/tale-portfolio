// BackendClient.swift

import Foundation
import Combine

protocol ARActionManaging: AnyObject {
    var isZashikiReady: Bool { get }
    func playAnimation(named name: String)
    func showSpeechBubble()
    func hideSpeechBubble()
    func setIdle()
}

final class BackendClient: NSObject, ObservableObject {
    
    // MARK: - Published State
    @Published var currentStep: ConversationStep = .intro
    
    @Published var currentMessage: String = BackendClient.defaultFindMeMessage()
    @Published var messages: [HistoryItem] = []
    @Published var currentSessionId: String?
    @Published var currentMode: Int = 0
    @Published var selectedLanguageCode: String = "ja"
    
    // Lesson進行状態（Audio判定・次ステップ進行に使う）
    @Published var currentLessonStepId: String?
    @Published var isLessonActive: Bool = false
    
    // MARK: - Shared properties
    var session: URLSession!
    var historyTask: URLSessionDataTask?
    var historyBuffer = Data()
    var audioTask: URLSessionDataTask?
    var audioBuffer = Data()
    var audioSSEStartCount: Int = 0
    var transcriptSSEStartCount: Int = 0
    var startRetryCount = 0
    var webSocketTask: URLSessionWebSocketTask?
    var shouldReconnectWebSocket = true
    weak var arManager: ARActionManaging? {
        didSet {
            print("🎬 arManager didSet - new value: \(arManager != nil ? "not nil" : "nil")")
            flushPendingARActionsIfPossible()
        }
    }
    
    /// WebSocket経由の音声再生時にテキストを仮保持するためのバッファ
    var pendingTranscript: (text: String, step: String?)?
    
    var audioStreamPlayer: AudioStreamPlayer? = AudioStreamPlayer()
    let baseURL = BackendClient.configuredBaseURL()
    var currentLessonId: String?

    private enum PendingARAction {
        case playAnimation(String)
        case showSpeechBubble
        case hideSpeechBubble
        case idle
    }

    private var pendingARActions: [PendingARAction] = []
    
    private static func defaultFindMeMessage() -> String {
        let lang = Locale.preferredLanguages.first?.lowercased() ?? Locale.current.identifier.lowercased()
        let code = String(lang.prefix(2))

        switch code {
        case "ja":
            return "スマホを動かして私を見つけてね"
        case "zh":
            return "移动手机来找到我哦"
        case "ko":
            return "휴대폰을 움직여서 나를 찾아줘"
        case "fr":
            return "Bouge ton téléphone pour me trouver"
        case "es":
            return "Mueve tu teléfono para encontrarme"
        default:
            return "Move your phone around to find me"
        }
    }

    private static func configuredBaseURL() -> URL {
        let candidates = [
            ProcessInfo.processInfo.environment["TALE_BACKEND_URL"],
            Bundle.main.object(forInfoDictionaryKey: "TALEBackendURL") as? String,
            "http://127.0.0.1:8080"
        ]

        for case let value? in candidates {
            let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
            if let url = URL(string: trimmed), !trimmed.isEmpty {
                return url
            }
        }

        preconditionFailure("A valid TALE backend URL is required")
    }
    
    override init() {
        super.init()
        
        let config = URLSessionConfiguration.default
        session = URLSession(configuration: config, delegate: self, delegateQueue: nil)
        
        startHistoryStream()
        startAudioStream()
        startWebSocket()
        startStatePolling()
    }

    func stopAudioPlayback() {
        audioStreamPlayer?.stop()
    }

    func handleARActions(_ actions: [ARAction]?) {
        print("🎬 handleARActions called, actions count:", actions?.count ?? 0)
        guard let actions, !actions.isEmpty else {
            print("🎬 handleARActions: no actions to process")
            return
        }
        print("🎬 handleARActions: processing \(actions.count) actions")
        DispatchQueue.main.async { [weak self] in
            guard let self else { return }
            for action in actions {
                print("🎬 Processing action - type: \(action.type ?? "nil"), target: \(action.target ?? "nil"), params: \(action.params ?? [:])")
                guard let type = action.type else {
                    print("🎬 Skipping action with nil type")
                    continue
                }
                switch type {
                case "PLAY_ANIMATION":
                    if let name = action.params?["name"] {
                        print("🎬 Enqueuing PLAY_ANIMATION: \(name)")
                        self.enqueueOrPerform(.playAnimation(name))
                    } else {
                        print("🎬 PLAY_ANIMATION has no 'name' param")
                    }
                case "SHOW_SPEECH_BUBBLE":
                    print("🎬 Enqueuing SHOW_SPEECH_BUBBLE")
                    self.enqueueOrPerform(.showSpeechBubble)
                case "HIDE_SPEECH_BUBBLE":
                    print("🎬 Enqueuing HIDE_SPEECH_BUBBLE")
                    self.enqueueOrPerform(.hideSpeechBubble)
                case "IDLE":
                    print("🎬 Enqueuing IDLE")
                    self.enqueueOrPerform(.idle)
                default:
                    print("🎬 Unknown AR action type:", type)
                }
            }
        }
    }

    private func enqueueOrPerform(_ action: PendingARAction) {
        if let manager = arManager {
            print("🎬 arManager available, performing action immediately")
            perform(action, with: manager)
        } else {
            print("🎬 arManager is nil, adding to pendingARActions (count: \(pendingARActions.count + 1))")
            pendingARActions.append(action)
        }
    }
    
    func flushPendingARActions() {
            flushPendingARActionsIfPossible()
        }

    private func flushPendingARActionsIfPossible() {
        guard let manager = arManager, !pendingARActions.isEmpty else {
            if arManager == nil {
                print("🎬 flushPending: arManager is nil")
            }
            if pendingARActions.isEmpty {
                print("🎬 flushPending: no pending actions")
            }
            return
        }
        print("🎬 Flushing \(pendingARActions.count) pending AR actions")
        let actions = pendingARActions
        pendingARActions.removeAll()
        for action in actions {
            perform(action, with: manager)
        }
    }

    private func perform(_ action: PendingARAction, with manager: ARActionManaging) {
        print("🎬 perform() called with action")
        switch action {
        case .playAnimation(let name):
            print("🎬 Calling manager.playAnimation(\(name))")
            manager.playAnimation(named: name)
        case .showSpeechBubble:
            print("🎬 Calling manager.showSpeechBubble()")
            manager.showSpeechBubble()
        case .hideSpeechBubble:
            print("🎬 Calling manager.hideSpeechBubble()")
            manager.hideSpeechBubble()
        case .idle:
            print("🎬 Calling manager.setIdle()")
            manager.setIdle()
        }
    }
    
    private func isLessonContext(step: String?) -> Bool {
        // lessonActive が一番信頼できる
        if isLessonActive { return true }

        // 念のため stepId っぽい形式でも拾う（例: "L1_INTRO"）
        guard let s = step, !s.isEmpty else { return false }
        return s.hasPrefix("L") && s.contains("_")   // L1_INTRO / L2_STEP3 などを想定
    }

    // ここは共通ロジックだけ残す
    // スクリプトTTS（SSE）か、Realtime/WS 音声かをステップで切り替える
    func shouldUseSSEAudio(step: String?) -> Bool {

        // ✅ Lesson中は常にTTS/SSE（ここを最優先）
        if isLessonContext(step: step) {
            return true
        }

        let resolvedStep: ConversationStep?
        if let raw = step, let mapped = ConversationStep(rawValue: raw) {
            resolvedStep = mapped
        } else {
            resolvedStep = currentStep
        }

        guard let step = resolvedStep else { return true }
        switch step {
        case .intro, .askName, .greetByName, .startGuide:
            return true   // 台本TTS → SSE
        default:
            return false  // Realtime/WS
        }
    }

    
    private func isLessonStep(_ step: String?) -> Bool {
        guard let step else { return false }
        return step.uppercased().hasPrefix("LESSON")
    }
    
    func handleAudioPlaybackFinished(step: String?) {
        DispatchQueue.main.async {
            // 状態更新（既存のままでOK）
            self.fetchState()
            
            // ✅ Lesson中なら次へ
            //if self.isLessonContext(step: step) {
              //  guard let lessonId = self.currentLessonId else {
                //    print("lesson id not found. currentLessonId=nil, step:", step ?? "nil")
                  //  return
                //}
               // self.nextLessonStep(lessonId: lessonId)
            //}
            //
        }
    }

    
    func resetForNewSession(sessionId: String?) {
        currentSessionId = sessionId
        messages.removeAll()
        currentMessage = "Loading..."
    }
    
    // MARK: - Lesson API
    
    // MARK: - Lesson API
    
    // 1) レスポンス型を変更
    struct LessonStep: Decodable {
        let stepId: String?
        let stepName: String?
        let content: String?
    }

    struct LessonStepResponse: Decodable {
        let status: String?
        let lessonId: String?
        let totalSteps: Int?
        let current: Int?
        let step: LessonStep?

        var lessonCompleted: Bool {
            return status == "completed"
        }
    }

    // 2) 既存のURLヘルパーはそのままでOK
    private func lessonURL(lessonId: String, action: String) -> URL {
        baseURL
            .appendingPathComponent("lessons")
            .appendingPathComponent(lessonId)   // "001"
            .appendingPathComponent(action)     // "start" / "next" / "current"
    }
    
    // 3) start / next / current は、そのまま handleLessonResponse に投げるだけでOK
    func startLesson(lessonId: String) {
        currentLessonId = lessonId
        currentLessonStepId = nil
        isLessonActive = true
        
        let url = lessonURL(lessonId: lessonId, action: "start")
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        
        resetForNewSession(sessionId: nil)
        
        session.dataTask(with: request) { data, response, error in
            if let error = error {
                print("lesson start error:", error)
                return
            }
            if let http = response as? HTTPURLResponse,
               !(200..<300).contains(http.statusCode) {
                print("lesson start failed status:", http.statusCode)
                if let data, let body = String(data: data, encoding: .utf8) {
                    print("lesson start error body:", body)
                }
                return
            }
            if let data {
                self.handleLessonResponse(data, context: "start \(lessonId)")
            }
        }.resume()
    }
    
    func nextLessonStep(lessonId: String) {
        let url = lessonURL(lessonId: lessonId, action: "next")
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        
        session.dataTask(with: request) { data, response, error in
            if let error = error {
                print("lesson next error:", error)
                return
            }
            if let http = response as? HTTPURLResponse,
               !(200..<300).contains(http.statusCode) {
                print("lesson next failed status:", http.statusCode)
                if let data, let body = String(data: data, encoding: .utf8) {
                    print("lesson next error body:", body)
                }
                return
            }
            if let data {
                self.handleLessonResponse(data, context: "next \(lessonId)")
            }
        }.resume()
    }
    
    func fetchCurrentLessonStep(lessonId: String) {
        let url = lessonURL(lessonId: lessonId, action: "current")
        
        session.dataTask(with: url) { data, response, error in
            if let error = error {
                print("lesson current error:", error)
                return
            }
            if let http = response as? HTTPURLResponse,
               !(200..<300).contains(http.statusCode) {
                print("lesson current failed status:", http.statusCode)
                if let data, let body = String(data: data, encoding: .utf8) {
                    print("lesson current error body:", body)
                }
                return
            }
            if let data {
                self.handleLessonResponse(data, context: "current \(lessonId)")
            }
        }.resume()
    }
    
    // 4) レスポンス処理を lessonId / stepId に対応させる
    private func handleLessonResponse(_ data: Data, context: String) {
        do {
            let decoder = JSONDecoder()
            decoder.keyDecodingStrategy = .convertFromSnakeCase

            let res = try decoder.decode(LessonStepResponse.self, from: data)

            DispatchQueue.main.async {
                if let lessonId = res.lessonId, !lessonId.isEmpty {
                    self.currentLessonId = lessonId
                    self.isLessonActive = true
                }

                // stepオブジェクトからcontentを取得
               // if let text = res.step?.content, !text.isEmpty {
                 //   self.currentMessage = text
                //} else {
                  //  self.currentMessage = ""
               // }

                // status == "completed" で完了判定
                if res.lessonCompleted {
                    self.isLessonActive = false
                    self.currentLessonStepId = "COMPLETE"
                } else {
                    self.currentLessonStepId = res.step?.stepId
                }

                print("📍 Lesson step updated: \(self.currentLessonStepId ?? "nil")")
            }
        } catch {
            if let raw = String(data: data, encoding: .utf8) {
                print("lesson response decode error:", error, "raw:", raw)
            } else {
                print("lesson response decode error:", error)
            }
        }
    }
}
