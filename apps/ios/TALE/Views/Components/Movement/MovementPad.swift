import SwiftUI

struct MovementPad: View {
    let size: CGSize
    let onMove: (CGFloat, CGFloat) -> Void

    var body: some View {
        let padSize = min(size.width, size.height) * 0.42
        let offset = padSize * 0.35

        return ZStack {
            Circle()
                .fill(
                    AngularGradient(
                        gradient: Gradient(
                            colors: [
                                Color.purple.opacity(0.7),
                                Color.blue.opacity(0.9),
                                Color.cyan.opacity(0.8),
                                Color.purple.opacity(0.7)
                            ]
                        ),
                        center: .center
                    )
                )
                .frame(width: padSize, height: padSize)
                .blur(radius: 16)
                .opacity(0.45)

            Circle()
                .fill(.ultraThinMaterial)
                .frame(width: padSize, height: padSize)
                .overlay(
                    Circle()
                        .stroke(
                            LinearGradient(
                                gradient: Gradient(colors: [Color.cyan.opacity(0.8), Color.purple.opacity(0.6)]),
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            ),
                            lineWidth: 3
                        )
                )
                .shadow(color: Color.black.opacity(0.25), radius: 12, x: 0, y: 8)

            Circle()
                .fill(
                    RadialGradient(
                        gradient: Gradient(colors: [Color.white.opacity(0.2), Color.clear]),
                        center: .center,
                        startRadius: 0,
                        endRadius: padSize * 0.6
                    )
                )
                .frame(width: padSize * 0.8, height: padSize * 0.8)

            movementButton(
                systemName: "arrow.up",
                colors: [Color.cyan, Color.blue]
            ) {
                onMove(0, 1000)
            }
            .offset(y: -offset)

            movementButton(
                systemName: "arrow.down",
                colors: [Color.orange, Color.red]
            ) {
                onMove(0, -1000)
            }
            .offset(y: offset)

            movementButton(
                systemName: "arrow.left",
                colors: [Color.indigo, Color.purple]
            ) {
                onMove(1000, 0)
            }
            .offset(x: -offset)

            movementButton(
                systemName: "arrow.right",
                colors: [Color.green, Color.mint]
            ) {
                onMove(-1000, 0)
            }
            .offset(x: offset)

            Circle()
                .fill(Color.white.opacity(0.22))
                .frame(width: 14, height: 14)
                .shadow(color: Color.white.opacity(0.25), radius: 8, x: 0, y: 0)
        }
    }

    private func movementButton(
        systemName: String,
        colors: [Color],
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            ZStack {
                Circle()
                    .fill(
                        LinearGradient(
                            gradient: Gradient(colors: colors),
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        )
                    )
                    .frame(width: 60, height: 60)
                    .shadow(color: colors.last?.opacity(0.45) ?? Color.black.opacity(0.35), radius: 12, x: 0, y: 8)

                Circle()
                    .stroke(Color.white.opacity(0.5), lineWidth: 1.5)
                    .frame(width: 60, height: 60)

                Image(systemName: systemName)
                    .font(.system(size: 22, weight: .bold))
                    .foregroundColor(.white)
            }
        }
        .buttonStyle(.plain)
    }
}
