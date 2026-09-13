import SwiftUI

// MARK: - ContentView Layer Extensions

extension ContentView {
    @ViewBuilder
    func questMapLayer(geo: GeometryProxy) -> some View {
        if showQuestMap {
            ZStack {
                SolidBackgroundARView(controller: arSceneController)
                    .edgesIgnoringSafeArea(.all)

                VStack {
                    Spacer()
                    MovementPad(size: geo.size) { dx, dz in
                        arSceneController.move(dx: Float(dx), dz: Float(dz))
                    }
                    .frame(maxWidth: .infinity)
                    .frame(height: geo.size.height * 0.26)
                    .padding(.horizontal, 24)
                    .padding(.vertical, 16)
                    .cornerRadius(16)
                    .shadow(color: Color.black.opacity(0.2), radius: 10, x: 0, y: 6)
                    .padding(.bottom, geo.size.height * 0.03)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottom)
            }
            .transition(.opacity)
            .zIndex(22)

            if arSceneController.showSakuraPopup {
                ZStack {
                    Color.black.opacity(0.4)
                        .edgesIgnoringSafeArea(.all)
                        .onTapGesture {
                            withAnimation {
                                arSceneController.showSakuraPopup = false
                            }
                            showSakuraQuestion = false
                            arSceneController.clearAnswerTargets()
                        }

                    VStack(spacing: 24) {
                        VStack(spacing: 8) {
                            Text("⚜️")
                                .font(.system(size: 40))
                                .foregroundColor(Color(red: 1.0, green: 0.8, blue: 0.4))
                                .padding(.bottom, 4)

                            Text("SAKURA QUEST")
                                .font(.system(size: 34, weight: .heavy, design: .serif))
                                .foregroundColor(Color(red: 1.0, green: 0.95, blue: 0.8))
                                .shadow(color: Color(red: 0.4, green: 0.1, blue: 0.3).opacity(0.8), radius: 2, x: 1, y: 1)

                            Rectangle()
                                .frame(height: 2)
                                .foregroundColor(Color(red: 1.0, green: 0.8, blue: 0.4))
                                .padding(.horizontal, 40)
                        }
                        .padding(.top, 20)

                        Text("浅草寺：鐘の音の秘密を解け\n（豆知識）")
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(.white.opacity(0.9))
                            .multilineTextAlignment(.center)
                            .padding(.horizontal)

                        VStack(spacing: 16) {
                            Button(action: {
                                debugLog("Start button tapped")
                                withAnimation {
                                    arSceneController.showSakuraPopup = false
                                }
                                showQuestMap = false
                                showSakuraQuestion = true
                            }) {
                                PopupStyledButton(text: "Start")
                            }

                            Button(action: {
                                withAnimation {
                                    arSceneController.showSakuraPopup = false
                                }
                                showSakuraQuestion = false
                                arSceneController.clearAnswerTargets()
                            }) {
                                PopupStyledButton(text: "Back")
                            }
                        }
                        .padding(.horizontal, 30)
                        .padding(.bottom, 30)
                    }
                    .frame(maxWidth: 360)
                    .background(
                        ZStack {
                            Color(red: 0.7, green: 0.3, blue: 0.5).opacity(0.85)
                            RoundedRectangle(cornerRadius: 20)
                                .stroke(Color(red: 1.0, green: 0.8, blue: 0.4), lineWidth: 3)
                                .padding(4)
                            RoundedRectangle(cornerRadius: 24)
                                .stroke(Color(red: 0.5, green: 0.2, blue: 0.4), lineWidth: 2)
                        }
                    )
                    .cornerRadius(24)
                    .shadow(color: .black.opacity(0.5), radius: 15, x: 0, y: 10)
                    .padding(20)
                }
                .transition(.opacity)
                .zIndex(40)
            }

            if arSceneController.showCastlePopup {
                ZStack {
                    Color.black.opacity(0.4)
                        .edgesIgnoringSafeArea(.all)
                        .onTapGesture {
                            withAnimation {
                                arSceneController.showCastlePopup = false
                            }
                        }

                    VStack(spacing: 16) {
                        Text("CASTLE QUEST")
                            .font(.system(size: 28, weight: .heavy, design: .serif))
                            .foregroundColor(.white)
                        Text("仲見世：仲見世のしるしを探せ（観察ミッション)")
                            .font(.subheadline.weight(.medium))
                            .foregroundColor(.white.opacity(0.9))

                        Button(action: {
                            withAnimation {
                                arSceneController.showCastlePopup = false
                            }
                        }) {
                            PopupStyledButton(text: "Back")
                        }
                        .padding(.top, 8)
                    }
                    .padding(24)
                    .frame(maxWidth: 320)
                    .background(
                        RoundedRectangle(cornerRadius: 20)
                            .fill(Color.gray.opacity(0.9))
                    )
                    .shadow(color: .black.opacity(0.4), radius: 12, x: 0, y: 6)
                }
                .transition(.opacity)
                .zIndex(41)
            }
        }
    }

    @ViewBuilder
    var sakuraQuestionLayer: some View {
        if showSakuraQuestion {
            VStack(spacing: 0) {
                Text(.init("""
問題文

浅草寺の鐘は、1日に2回鳴ります。

Q. その時間帯は？
① 朝と夕方
② 昼と夜
③ 深夜のみ
"""))
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundColor(.white)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(16)
                    .background(Color.black.opacity(0.55))
                    .cornerRadius(12)
                    .padding(.horizontal, 16)
                    .padding(.top, 24)
                Spacer()
                Button(action: {
                    arSceneController.fire()
                }) {
                    Text("FIRE")
                        .font(.system(size: 18, weight: .black))
                        .foregroundColor(.white)
                        .padding(.horizontal, 32)
                        .padding(.vertical, 14)
                        .background(
                            LinearGradient(
                                colors: [
                                    Color(red: 0.95, green: 0.35, blue: 0.2),
                                    Color(red: 0.9, green: 0.15, blue: 0.05)
                                ],
                                startPoint: .top,
                                endPoint: .bottom
                            )
                        )
                        .cornerRadius(18)
                        .shadow(color: Color.black.opacity(0.4), radius: 8, x: 0, y: 6)
                }
                .padding(.bottom, 28)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .zIndex(60)
            .onAppear {
                arSceneController.showAnswerTargets()
            }
            .onDisappear {
                arSceneController.clearAnswerTargets()
            }
        }
    }

    @ViewBuilder
    func standardUILayer(geo: GeometryProxy) -> some View {
        if !showSakuraQuestion && !arSceneController.showCorrectOverlay {
            VStack(alignment: .trailing, spacing: 0) {
                headerBar
                menuButtons
                    .padding(.top, 100)
                    .padding(.trailing, 16)
                Spacer()
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
            .zIndex(5)

            VStack {
                Spacer()
                VStack(spacing: 12) {
                    ConversationCard(
                        text: backend.currentMessage,
                        geo: geo
                    )

                    ControlButtons(
                        hasStartedConversation: hasStartedConversation,
                        isPaused: isConversationPaused,
                        onStart: { startConversationNow() },
                        onPause: { pauseConversationUI() },
                        onResume: { resumeConversationUI() },
                        onPrev: { debugLog("◀️ left tapped") },
                        onNext: { debugLog("▶️ right tapped") }
                    )
                    .padding(.bottom, 40)
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 16)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottom)
            .zIndex(9)

            if showQuestMenu {
                ZStack {
                    Color.black.opacity(0.6)
                        .ignoresSafeArea()
                    Image("Quest_Menu")
                        .resizable()
                        .scaledToFit()
                        .frame(width: geo.size.width * 1.0)
                    VStack(spacing: 10) {
                        Text("MENU")
                            .foregroundColor(.black)
                            .frame(maxWidth: .infinity, alignment: .center)
                            .multilineTextAlignment(.center)
                            .font(.system(.largeTitle, design: .serif))
                            .fontWeight(.black)
                        questButton(title: "Culture Quest") {
                            showQuestMap = true
                            withAnimation {
                                showQuestMenu = false
                            }
                        }
                        questButton(title: "Japanese Lesson") {
                            withAnimation {
                                showJapaneseLessonMenu = true
                                showQuestMenu = false
                            }
                        }
                        questButton(title: "Help & Support")
                    }
                    .frame(width: 220)
                }
                .transition(.opacity)
                .zIndex(25)
                .onTapGesture {
                    withAnimation {
                        showQuestMenu = false
                    }
                }
            }

            if showJapaneseLessonMenu || showLessonScene {
                JapaneseLessonOverlay(
                    geo: geo,
                    showLessonScene: $showLessonScene,
                    showLessonCompletePopup: $showLessonCompletePopup,
                    currentLessonId: backend.currentLessonId,
                    currentMessage: backend.currentMessage,
                    onClose: {
                        withAnimation {
                            showJapaneseLessonMenu = false
                            showLessonScene = false
                        }
                    },
                    onStartLesson: { lessonId in
                        let paddedId = String(format: "%03d", lessonId)
                        backend.stopAudioPlayback()
                        showLessonCompletePopup = false
                        backend.startLesson(lessonId: paddedId)
                        withAnimation {
                            showJapaneseLessonMenu = false
                            showLessonScene = true
                        }
                    },
                    onStartPlayback: { startLessonPlayback() },
                    onNextStep: {
                        guard let id = backend.currentLessonId?.trimmingCharacters(in: .whitespacesAndNewlines),
                              !id.isEmpty else {
                            debugLog("⚠️ Lesson ID is missing; cannot go to next step.")
                            return
                        }
                        backend.nextLessonStep(lessonId: paddedLessonId(id))
                    }
                )
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
                .transition(.opacity)
                .zIndex(32)
            }

            if showQuestMap && !arSceneController.showCorrectOverlay {
                VStack {
                    HStack {
                        Spacer()
                        Button {
                            showQuestMap = false
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .font(.headline)
                                .fontWeight(.heavy)
                                .foregroundColor(.white)
                                .padding(12)
                                .background(Color.black.opacity(0.4))
                                .clipShape(Circle())
                        }
                    }
                    Spacer()
                }
                .padding()
                .transition(.opacity)
                .zIndex(30)
            }

            if (showLanguageSelection || showLanguageGreeting) && !arSceneController.showCorrectOverlay {
                LanguageSelectionOverlay(
                    geo: geo,
                    showLanguageSelection: $showLanguageSelection,
                    showLanguageGreeting: $showLanguageGreeting,
                    selectedLanguage: $selectedLanguage,
                    greetingText: greetingText(for:),
                    onSelectLanguage: { title in
                        selectLanguage(title: title)
                    },
                    onCloseGreeting: {
                        withAnimation {
                            showLanguageGreeting = false
                        }
                    }
                )
                .zIndex(60)
            }
        }
    }

    @ViewBuilder
    var correctOverlayLayer: some View {
        if arSceneController.showCorrectOverlay {
            QuestCompleteOverlay {
                arSceneController.showCorrectOverlay = false
                goBackToMap()
            }
            .zIndex(90)
        }
    }
}
