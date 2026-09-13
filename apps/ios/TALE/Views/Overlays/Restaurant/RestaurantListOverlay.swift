import SwiftUI
import RealityKit
import ARKit
import Combine
import AVFoundation
import AVKit
import WebKit
import MessageUI
import UIKit

struct RestaurantListOverlay: View {
    @EnvironmentObject var backend: BackendClient

    struct Restaurant: Identifiable {
        let id: String
        let name: String
        let cuisine: String
        let distance: String
        let destination: Destination?
    }

    enum Destination {
        case sketchfab
        case japaneseRestaurant
    }

    private let restaurants: [Restaurant] = [
        .init(id: "sakura", name: "Sakura Bistro", cuisine: "和食・定食", distance: "徒歩3分", destination: .sketchfab),
        .init(id: "ocean", name: "Ocean Grill", cuisine: "シーフード", distance: "徒歩5分", destination: .japaneseRestaurant),
        .init(id: "pasta", name: "Pasta Verde", cuisine: "イタリアン", distance: "徒歩7分", destination: nil),
        .init(id: "spice", name: "Spice Route", cuisine: "カレー・スパイス", distance: "徒歩9分", destination: nil),
        .init(id: "cafe", name: "Cafe Mellow", cuisine: "カフェ・スイーツ", distance: "徒歩2分", destination: nil)
    ]

    var body: some View {
        NavigationStack {
            List(restaurants) { item in
                NavigationLink(value: item.id) {
                    restaurantRow(item: item)
                }
            }
            .navigationDestination(for: String.self) { id in
                if let item = restaurants.first(where: { $0.id == id }) {
                    destinationView(for: item)
                } else {
                    Text("Not Found")
                }
            }
            .navigationTitle(LocalizedStringKey("restaurant_title"))
        }
    }

    @ViewBuilder
    private func destinationView(for item: Restaurant) -> some View {
        switch item.destination {
        case .sketchfab:
            SketchfabHTMLView()
        case .japaneseRestaurant:
            IzakayaARView(step: backend.currentStep)
        case .none:
            Text(item.name)
        }
    }

    private func restaurantRow(item: Restaurant) -> some View {
        HStack {
            VStack(alignment: .leading, spacing: 4) {
                Text(item.name)
                    .font(.headline)
                Text(item.cuisine)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }
            Spacer()
            Text(item.distance)
                .font(.subheadline)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 4)
    }
}
