import SwiftUI

struct LaunchOverlay: View {
    var body: some View {
        ZStack {
            Color(red: 0.78, green: 0.62, blue: 0.08)
                .ignoresSafeArea()

            VStack(spacing: 20) {
                Text("TALE")
                    .font(.system(size: 42, weight: .heavy))
                    .foregroundColor(.white)
                    .shadow(color: .black.opacity(0.25), radius: 6, x: 0, y: 4)

                ProgressView()
                    .progressViewStyle(CircularProgressViewStyle(tint: .white))
                    .scaleEffect(1.1)
            }
        }
    }
}
