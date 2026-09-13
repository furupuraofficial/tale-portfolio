import SwiftUI

struct JapaneseLessonOverlay: View {
    let geo: GeometryProxy
    @Binding var showLessonScene: Bool
    @Binding var showLessonCompletePopup: Bool
    let currentLessonId: String?
    let currentMessage: String
    let onClose: () -> Void
    let onStartLesson: (Int) -> Void
    let onStartPlayback: () -> Void
    let onNextStep: () -> Void

    @State private var showBeginnerLessons = false
    @State private var selectedLessonId: Int?

    var body: some View {
        ZStack(alignment: .center) {
            Image("Lesson_Intro")
                .resizable()
                .scaledToFill()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .ignoresSafeArea()

            if !showLessonScene {
                Button(action: onClose) {
                    Color.black.opacity(0.5)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                }
                .buttonStyle(.plain)
            }

            if showLessonScene {
                lessonSceneView
            } else {
                lessonMenuView
            }
        }
        .frame(width: geo.size.width, height: geo.size.height, alignment: .center)
        .padding(.top, 80)
    }

    private var lessonMenuView: some View {
        VStack(spacing: 16) {
            if showBeginnerLessons {
                Text("Please select your Lesson")
                    .frame(maxWidth: .infinity, alignment: .center)
                    .multilineTextAlignment(.center)
                    .foregroundColor(.white)
                    .font(.title2)
                    .bold()
                VStack(spacing: 12) {
                    ForEach(1...5, id: \.self) { num in
                        lessonButton(title: "Lesson\(num)") {
                            selectedLessonId = num
                            onStartLesson(num)
                        }
                    }
                }
            } else {
                Text("Please select your level")
                    .foregroundColor(.white)
                    .font(.title2)
                    .bold()
                VStack(spacing: 16) {
                    lessonButton(title: "Easy") {
                        withAnimation {
                            showBeginnerLessons = true
                        }
                    }
                    lessonButton(title: "Standard")
                    lessonButton(title: "Difficult")
                }
            }
        }
        .frame(
            maxWidth: geo.size.width * 0.9,
            maxHeight: .infinity,
            alignment: .center
        )
    }

    private var lessonSceneView: some View {
        ZStack {
            VStack(spacing: 1) {
                Spacer()
                ZashikiPreviewView()
                    .frame(
                        width: geo.size.width * 0.9,
                        height: geo.size.height * 0.45
                    )
                ConversationCard(
                    text: currentMessage,
                    geo: geo
                )

                LessonControlButtons(
                    onStart: { onStartPlayback() },
                    onNext: { onNextStep() }
                )
                Spacer(minLength: 20)
            }
            .padding(.horizontal, 16)
            .padding(.top, 80)

            if showLessonCompletePopup {
                Color.black.opacity(0.45)
                    .ignoresSafeArea()
                    .transition(.opacity)
                VStack(spacing: 16) {
                    Text("Lesson \(currentLessonId ?? "") クリア ✨")
                        .font(.title3.weight(.bold))
                        .foregroundColor(.white)
                    Text("おめでとう！！")
                        .font(.headline)
                        .foregroundColor(.white)
                    Text("タップして閉じる")
                        .font(.subheadline)
                        .foregroundColor(.white.opacity(0.8))
                        .padding(.top, 4)
                }
                .padding(.horizontal, 24)
                .padding(.vertical, 20)
                .background(.ultraThinMaterial)
                .cornerRadius(20)
                .shadow(color: .black.opacity(0.3), radius: 12, x: 0, y: 8)
                .padding(32)
                .onTapGesture {
                    withAnimation {
                        showLessonCompletePopup = false
                        showLessonScene = false
                    }
                }
                .transition(.scale.combined(with: .opacity))
                .zIndex(40)
            }
        }
    }

    private func lessonButton(title: String, action: @escaping () -> Void = {}) -> some View {
        Button(action: action) {
            Text(title)
                .frame(maxWidth: 300)
                .padding()
                .background(Color.white.opacity(0.85))
                .foregroundColor(.black)
                .font(.headline)
                .cornerRadius(14)
                .shadow(color: .black.opacity(0.25), radius: 8, x: 0, y: 4)
        }
    }
}
