import SwiftUI

struct LanguageSelectionOverlay: View {
    let geo: GeometryProxy
    @Binding var showLanguageSelection: Bool
    @Binding var showLanguageGreeting: Bool
    @Binding var selectedLanguage: String?

    let greetingText: (String) -> String
    let onSelectLanguage: (String) -> Void
    let onCloseGreeting: () -> Void

    private let languageOptions = [
        "日本語", "English", "中文", "한국어", "Français", "Español"
    ]

    var body: some View {
        ZStack(alignment: .center) {
            Image("Language_Select")
                .resizable()
                .scaledToFill()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .ignoresSafeArea()

            PageCurlTransition(trigger: !showLanguageSelection) {
                Group {
                    if showLanguageSelection {
                        VStack(spacing: 20) {
                            Text("Please select your favolite language")
                                .foregroundColor(.white)
                                .font(.headline)
                                .frame(maxWidth: .infinity, alignment: .center)
                                .multilineTextAlignment(.center)
                            Text("お好きな言葉をお選びください。")
                                .foregroundColor(.white)
                                .font(.subheadline)
                                .frame(maxWidth: .infinity, alignment: .center)
                                .multilineTextAlignment(.center)

                            VStack(spacing: 10) {
                                ForEach(languageOptions, id: \.self) { option in
                                    Button {
                                        onSelectLanguage(option)
                                    } label: {
                                        Text(option)
                                            .frame(maxWidth: .infinity)
                                            .padding()
                                            .background(Color.white.opacity(0.2))
                                            .foregroundColor(.white)
                                            .cornerRadius(8)
                                    }
                                }
                            }
                            .padding()
                        }
                        .frame(
                            maxWidth: geo.size.width * 0.9,
                            maxHeight: .infinity,
                            alignment: .center
                        )
                        .padding()
                    } else if let lang = selectedLanguage {
                        VStack {
                            VStack(spacing: 80) {
                                Text(greetingText(lang))
                                    .font(.system(.body, design: .serif))
                                    .foregroundColor(.black)
                                    .fontWeight(.regular)
                                    .multilineTextAlignment(.center)
                                    .frame(maxWidth: .infinity, alignment: .center)
                                    .padding(80)
                                    .background(
                                        Image("Intro_Message")
                                            .resizable()
                                            .scaledToFill()
                                            .clipped()
                                            .clipShape(RoundedRectangle(cornerRadius: 12))
                                    )
                            }
                            .frame(maxWidth: .infinity, alignment: .center)
                            .padding(.horizontal, 12)
                        }
                        .frame(
                            maxWidth: geo.size.width,
                            maxHeight: .infinity,
                            alignment: .center
                        )
                        .contentShape(Rectangle())
                        .onTapGesture {
                            withAnimation {
                                showLanguageGreeting = false
                            }
                            onCloseGreeting()
                        }
                    }
                }
            }
        }
        .frame(width: geo.size.width, height: geo.size.height, alignment: .top)
        .padding(.top, 80)
    }
}
