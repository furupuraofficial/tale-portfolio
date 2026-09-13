import SwiftUI

struct PopupStyledButton: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.headline.weight(.bold))
            .foregroundColor(Color(red: 1.0, green: 0.95, blue: 0.8))
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(
                ZStack {
                    LinearGradient(
                        gradient: Gradient(colors: [
                            Color(red: 0.4, green: 0.1, blue: 0.3),
                            Color(red: 0.6, green: 0.2, blue: 0.4)
                        ]),
                        startPoint: .top,
                        endPoint: .bottom
                    )
                    RoundedRectangle(cornerRadius: 12)
                        .strokeBorder(
                            LinearGradient(
                                gradient: Gradient(colors: [
                                    Color(red: 1.0, green: 0.9, blue: 0.5),
                                    Color(red: 0.8, green: 0.6, blue: 0.2)
                                ]),
                                startPoint: .top,
                                endPoint: .bottom
                            ),
                            lineWidth: 3
                        )
                }
            )
            .cornerRadius(12)
            .shadow(color: .black.opacity(0.3), radius: 4, x: 0, y: 4)
    }
}
