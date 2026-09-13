import SwiftUI

struct LessonControlButtons: View {
    let onStart: () -> Void
    let onNext: () -> Void

    var body: some View {
        HStack(spacing: 24) {
            circularButton(systemName: "play.fill", size: 72, action: onStart)
            circularButton(systemName: "arrow.forward", action: onNext)
        }
    }

    private func circularButton(
        systemName: String,
        size: CGFloat = 56,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            ZStack {
                LinearGradient(
                    gradient: Gradient(colors: [Color.red, Color.orange]),
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
                .frame(width: size, height: size)
                .clipShape(Circle())

                Image(systemName: systemName)
                    .font(.system(size: 20, weight: .bold))
                    .foregroundColor(.white)
            }
        }
        .buttonStyle(.plain)
        .shadow(color: Color.black.opacity(0.25), radius: 6, x: 0, y: 3)
    }
}
