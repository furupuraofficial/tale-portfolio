import SwiftUI
import RealityKit
import ARKit
import Combine
import AVFoundation
import AVKit
import WebKit
import MessageUI
import UIKit

struct IzakayaARView: View {
    let step: ConversationStep
    @State private var showDessertBubble = false

    var body: some View {
        ZStack(alignment: .top) {
            IzakayaARContainer(
                step: step,
                showDessertBubble: $showDessertBubble
            )
            .edgesIgnoringSafeArea(.all)
        }
        .navigationTitle("Izakaya AR")
        .navigationBarTitleDisplayMode(.inline)
        .animation(.easeInOut, value: showDessertBubble)
    }
}

struct IzakayaARContainer: UIViewRepresentable {
    let step: ConversationStep
    @Binding var showDessertBubble: Bool

    func makeCoordinator() -> Coordinator {
        Coordinator(showDessertBubble: $showDessertBubble)
    }

    func makeUIView(context: Context) -> ARView {
        let arView = ARView(frame: .zero)

        let configuration = ARWorldTrackingConfiguration()
        arView.session.run(configuration)

        let tap = UITapGestureRecognizer(
            target: context.coordinator,
            action: #selector(Coordinator.handleTap(_:))
        )
        arView.addGestureRecognizer(tap)

        let candidates = [
            "Japanese_Restaurant_Inakaya",
        ]

        var loaded = false
        for name in candidates {
            if let entity = try? Entity.load(named: name) {
                // カメラ正面 1m に固定配置し、画面いっぱいに見えるようスケールアップ
                let anchor = AnchorEntity(world: SIMD3<Float>(-2, -2, -2))
                entity.scale = [0.02, 0.02, 0.02]
                anchor.addChild(entity)
                arView.scene.addAnchor(anchor)
                loaded = true
                break
            }
        }

        // Lowpoly_Kimono_cat を同時に配置
        if let cat = try? Entity.load(named: "Lowpoly_Kimono_cat") {
            let catAnchor = AnchorEntity(world: SIMD3<Float>(0, -0.5, -1))
            cat.name = "introCat"
            cat.scale = [0.5, 0.5, 0.5]
            catAnchor.addChild(cat)
            arView.scene.addAnchor(catAnchor)
            context.coordinator.catAnchor = catAnchor

            // スポットライトを猫に向ける（STEP_INTRO時にのみ有効）
            if step == .intro {
                let lightAnchor = AnchorEntity(world: SIMD3<Float>(0, 0, -1))
                let lightEntity = SpotLight()
                lightEntity.light.color = .red
                lightEntity.light.intensity = 3000
                lightEntity.light.innerAngleInDegrees = 45
                lightEntity.light.outerAngleInDegrees = 60
                lightEntity.light.attenuationRadius = 10
                lightEntity.shadow = SpotLightComponent.Shadow()
                lightEntity.look(
                    at: cat.position,
                    from: [0, 0.05, 0.3],
                    relativeTo: catAnchor
                )
                lightEntity.name = "introLight"
                lightAnchor.addChild(lightEntity)
                arView.scene.addAnchor(lightAnchor)
            }
        } else {
            debugLog("⚠️ Lowpoly_Kimono_cat.usdz が見つかりませんでした")
        }

        if !loaded {
            debugLog("⚠️ Izakayaモデルが見つかりませんでした")
        }

        return arView
    }

    func updateUIView(_ uiView: ARView, context: Context) {}

    class Coordinator: NSObject {
        private var showDessertBubble: Binding<Bool>
        var catAnchor: AnchorEntity?
        var dessertAnchor: AnchorEntity?

        init(showDessertBubble: Binding<Bool>) {
            self.showDessertBubble = showDessertBubble
        }

        @objc
        func handleTap(_ gesture: UITapGestureRecognizer) {
            guard let view = gesture.view as? ARView else { return }
            let location = gesture.location(in: view)
            if let entity = view.entity(at: location),
               entity.name == "dessert" || entity.parent?.name == "dessert" {
                showDessertBubble.wrappedValue = true
            }
        }
    }
}
