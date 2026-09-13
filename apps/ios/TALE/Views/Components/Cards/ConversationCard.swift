import SwiftUI

struct ConversationCard: View {
    let text: String
    let geo: GeometryProxy

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(text)
                .foregroundColor(.white)
                .padding(.horizontal, 16)
                .padding(.vertical, 12)
        }
        .frame(width: geo.size.width * 0.8)
        .frame(
            minHeight: geo.size.height * 0.2,
            maxHeight: geo.size.height * 0.2
        )
        .background(
            Color.black.opacity(0.7)
                .clipShape(RoundedRectangle(cornerRadius: 16))
        )
        .shadow(color: .black.opacity(0.25), radius: 12, x: 10, y: 10)
    }
}
