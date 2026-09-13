import SwiftUI
import RealityKit

struct ZashikiPreviewView: UIViewRepresentable {
    func makeUIView(context: Context) -> ARView {
        
        debugLog("makeUIView called")
        let view = ARView(frame: .zero, cameraMode: .nonAR, automaticallyConfigureSession: false)
        view.environment.background = .color(.clear)

        debugLog("🔍 ARPreview cameraMode:", view.cameraMode)

        let anchor = AnchorEntity()
        if let catURL = Bundle.main.url(forResource: "Lowpoly_Kimono_cat", withExtension: "usdz"),
           let cat = try? Entity.load(contentsOf: catURL) {
            cat.scale = SIMD3<Float>(repeating: 1.2)
            cat.position = SIMD3<Float>(0, 0, 0)
            cat.transform.rotation = .init() // USDZ 側の初期回転をリセットして正面に
            anchor.addChild(cat)
        }
        view.scene.addAnchor(anchor)

        return view
    }

    func updateUIView(_ uiView: ARView, context: Context) {}
}
