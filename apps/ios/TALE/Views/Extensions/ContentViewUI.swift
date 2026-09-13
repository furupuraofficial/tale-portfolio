import SwiftUI

// MARK: - ContentView UI Components

extension ContentView {
    var menuButtons: some View {
        VStack(spacing: 12) {
            menuButton(
                systemName: "fork.knife",
                action: { showRestaurantList = true }
            )
            menuButton(
                systemName: "questionmark.circle",
                action: { showSupportForm = true }
            )
            menuButton(
                systemName: "gearshape",
                action: { debugLog("⚙️ settings tapped") }
            )
            menuButton(
                systemName: UIImage(systemName: "crossed.swords") != nil
                    ? "crossed.swords"
                    : "cross.vial.fill",
                action: { withAnimation { showQuestMenu = true } }
            )
        }
    }

    func menuButton(systemName: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            ZStack {
                LinearGradient(
                    gradient: Gradient(colors: [Color.red, Color.orange]),
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
                .frame(width: 48, height: 48)
                .clipShape(Circle())

                Image(systemName: systemName)
                    .font(.system(size: 20, weight: .bold))
                    .foregroundColor(.white)
            }
        }
        .buttonStyle(.plain)
        .shadow(color: Color.black.opacity(0.25), radius: 6, x: 0, y: 3)
    }

    var headerBar: some View {
        ZStack {
            Color(red: 0.82, green: 0.72, blue: 0.60)

            HStack {
                Text("305")
                    .font(.system(size: 30, weight: .bold))
                    .foregroundColor(.black)

                Button(action: { debugLog("Language tapped") }) {
                    Text("Language")
                        .font(.subheadline)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .background(Color.white.opacity(0.2))
                        .cornerRadius(8)
                        .foregroundColor(.black)
                }
            }
            .padding(.horizontal, 12)
            .padding(.top, 8)
        }
        .frame(height: 108)
        .edgesIgnoringSafeArea(.top)
    }

    func questButton(title: String, action: @escaping () -> Void = {}) -> some View {
        Button(action: action) {
            HStack(spacing: 10) {
                Image(systemName: "chevron.right.circle.fill")
                    .font(.title3.weight(.semibold))
                    .foregroundColor(.black)
                Text(title)
                    .foregroundColor(.black)
                    .font(.system(size: 17, weight: .semibold, design: .serif))
                Spacer()
            }
            .padding(.vertical, 10)
            .padding(.horizontal, 8)
            .contentShape(Rectangle())
        }
    }
}
