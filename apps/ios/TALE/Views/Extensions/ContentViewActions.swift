import SwiftUI
import AVFoundation
import AVKit

// MARK: - ContentView Actions

extension ContentView {
    func goBackToMap() {
        showQuestMap = true
    }

    // MARK: - Intro Video & Player

    func setupIntroPlayerIfNeeded() {
        guard introPlayer == nil else { return }
        let candidates: [(String, String)] = [
            ("Window2", "mov"),
            ("Window2", "MOV")
        ]
        var loadedURL: URL?
        for (name, ext) in candidates {
            if let url = Bundle.main.url(forResource: name, withExtension: ext) {
                loadedURL = url
                break
            }
        }

        if let url = loadedURL {
            let asset = AVAsset(url: url)
            let item = AVPlayerItem(asset: asset)
            let queuePlayer = AVQueuePlayer()
            introLooper = AVPlayerLooper(player: queuePlayer, templateItem: item)
            queuePlayer.actionAtItemEnd = .none
            introPlayer = queuePlayer
        } else {
            print("⚠️ Window2.mp4 がバンドル内に見つかりません")
        }
    }

    func startIntroVideo() {
        guard let player = introPlayer else { return }

        if hasPlayedIntroVideo { return }
        hasPlayedIntroVideo = true

        let delay: TimeInterval = 2
        let playDuration: TimeInterval = 3

        showIntroVideo = true
        player.seek(to: .zero)

        DispatchQueue.main.asyncAfter(deadline: .now() + delay) {
            guard let player = introPlayer, player.currentItem != nil else {
                showIntroVideo = false
                return
            }
            player.play()

            DispatchQueue.main.asyncAfter(deadline: .now() + playDuration) {
                guard let player = introPlayer else { return }
                player.pause()
                showIntroVideo = false
            }
        }
    }

    func startIntroThenConversation() {
        guard !hasStartedConversation else { return }
        DispatchQueue.main.asyncAfter(deadline: .now() + 15) {
            startConversationNow()
        }
    }

    func startConversationNow() {
        guard !hasStartedConversation else { return }
        hasStartedConversation = true
        isConversationPaused = false
        withAnimation {
            showIntroAnimation = false
        }
        backend.startConversation(languageCode: languageStore.code)
    }

    func pauseConversationUI() {
        guard hasStartedConversation else { return }
        backend.pauseConversation()
        isConversationPaused = true
    }

    func resumeConversationUI() {
        guard hasStartedConversation else { return }
        backend.resumeConversation()
        isConversationPaused = false
    }

    func startLessonPlayback() {
        guard let id = backend.currentLessonId else {
            print("⚠️ Lesson ID is missing; cannot start lesson.")
            return
        }
        showLessonCompletePopup = false
        hasLessonStarted = true
    }

    func fetchCurrentLessonIfNeeded() {
        guard let id = backend.currentLessonId else { return }
        backend.fetchCurrentLessonStep(lessonId: paddedLessonId(id))
    }

    func paddedLessonId(_ id: String) -> String {
        if let numeric = Int(id) {
            return String(format: "%03d", numeric)
        }
        return id
    }

    func selectLanguage(title: String) {
        selectedLanguage = title
        let code = languageCode(for: title)
        languageStore.setLanguage(code: code)
        backend.setLanguage(code: code)
        withAnimation {
            showLanguageSelection = false
            showLanguageGreeting = true
        }
    }

    func formattedTime(_ isoString: String?) -> String? {
        guard let isoString else { return nil }
        let formatter = ISO8601DateFormatter()
        if let date = formatter.date(from: isoString) {
            let out = DateFormatter()
            out.locale = .current
            out.dateFormat = "HH:mm:ss"
            return out.string(from: date)
        }
        return isoString
    }
}
