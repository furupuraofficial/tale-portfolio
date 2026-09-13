import SwiftUI

struct QuestCompleteOverlay: View {
    let onBackToMap: () -> Void

    var body: some View {
        ZStack {
            Color.black.opacity(0.4)
                .ignoresSafeArea()
            FireworkBackground()
                .ignoresSafeArea()

            VStack(spacing: 24) {
                Text("Mission Complete!")
                    .font(.system(size: 42, weight: .heavy, design: .serif))
                    .fontWeight(.bold)
                    .foregroundColor(.white)

                Button(action: onBackToMap) {
                    Text("Back To Map")
                        .font(.headline)
                        .padding(.horizontal, 32)
                        .padding(.vertical, 14)
                        .background(Color.white)
                        .foregroundColor(.black)
                        .cornerRadius(12)
                }
            }
        }
    }
}
