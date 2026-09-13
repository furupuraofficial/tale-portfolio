import SwiftUI
import RealityKit
import ARKit
import Combine
import AVFoundation
import AVKit

// MARK: - メインの SwiftUI View

struct ContentView: View {
    @StateObject var backend = BackendClient()
    @StateObject var audioManager = AudioManager()
    @StateObject var languageStore = LanguageStore()
    @StateObject var arSceneController = ARSceneController()

    @State var showIntroAnimation = true
    @State var hasStartedConversation = false

    @State var introPlayer: AVQueuePlayer?
    @State var introLooper: AVPlayerLooper?
    @State var showIntroVideo = false
    @State var hasPlayedIntroVideo = false

    @State var hasLessonStarted = false

    @State var showRestaurantList = false
    @State var showSupportForm = false
    @State var isConversationPaused = false
    @State var showDessertBubbleTop = false
    @State var showQuestMenu = false
    @State var showQuestMap = false
    @State var showJapaneseLessonMenu = false
    @State var showLessonScene = false
    @State var showLessonCompletePopup = false
    @State var showSakuraQuestion = false
    @State var showLaunchOverlay = true

    @State var showLanguageSelection = true
    @State var selectedLanguage: String?
    @State var showLanguageGreeting = false

    var body: some View {
        GeometryReader { geo in
            ZStack {
                ARViewContainer(controller: arSceneController)
                    .edgesIgnoringSafeArea(.all)
                    .zIndex(0)

                questMapLayer(geo: geo)

                sakuraQuestionLayer

                standardUILayer(geo: geo)

                correctOverlayLayer

                if showLaunchOverlay {
                    LaunchOverlay()
                        .transition(.opacity)
                        .zIndex(200)
                }
            }
            .edgesIgnoringSafeArea(.all)
            .sheet(isPresented: $showRestaurantList) {
                RestaurantListOverlay()
                    .environmentObject(backend)
            }
            .sheet(isPresented: $showSupportForm) {
                SupportFormOverlay(isPresented: $showSupportForm)
            }
        }
        .onAppear {
            audioManager.configurePlaybackSession()
            audioManager.startBGM()
            setupIntroPlayerIfNeeded()
            debugLog("🎬 ContentView.onAppear: Setting arManager to arSceneController")
            backend.arManager = arSceneController
            arSceneController.onZashikiReady = {
                backend.flushPendingARActions()
            }
            debugLog("🎬 ContentView.onAppear: arManager set, backend should flush pending actions")
            DispatchQueue.main.asyncAfter(deadline: .now() + 2.0) {
                withAnimation(.easeOut(duration: 0.4)) {
                    showLaunchOverlay = false
                }
            }
        }
        .onChange(of: arSceneController.showCorrectOverlay) { isShown in
            if isShown {
                showSakuraQuestion = false
                showQuestMap = false
                arSceneController.clearAnswerTargets()
            }
        }
        .onDisappear {
            audioManager.stopBGM()
        }
        .onChange(of: showSakuraQuestion) { isShown in
            if isShown {
                audioManager.playBGM(named: "Quest_bgm")
            } else {
                audioManager.startBGM()
            }
        }
        .onChange(of: showQuestMap) { isShown in
            if isShown {
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                    arSceneController.animateCameraIntro()
                }
            }
        }
        .onChange(of: arSceneController.showCorrectOverlay) { isShown in
            if isShown {
                audioManager.playBGM(named: "MissionClear_bgm")
            } else if showSakuraQuestion {
                audioManager.playBGM(named: "Quest_bgm")
            } else {
                audioManager.startBGM()
            }
        }
        .onChange(of: backend.currentStep) { newStep in
            if newStep != .freeQuestion {
                withAnimation {
                    showDessertBubbleTop = false
                }
            }
        }
        .onChange(of: backend.currentLessonStepId) { stepId in
            debugLog("📍 stepId changed to:", stepId ?? "nil")
            let trimmed = stepId?.trimmingCharacters(in: .whitespacesAndNewlines).uppercased() ?? ""
            debugLog("📍 trimmed value:", trimmed)
            if trimmed == "COMPLETE" || trimmed.hasSuffix("_COMPLETE") {
                debugLog("✅ completion detected:", trimmed)
                withAnimation {
                    showLessonScene = true
                    showLessonCompletePopup = true
                }
            }
        }
    }
}

#Preview {
    ContentView()
}
