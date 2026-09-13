import SwiftUI
import RealityKit

struct FireworkBackground: UIViewRepresentable {
    func makeUIView(context: Context) -> ARView {
        let arView = ARView(
            frame: .zero,
            cameraMode: .nonAR,
            automaticallyConfigureSession: false
        )
        arView.environment.background = .color(.clear)

        let anchor = AnchorEntity(world: .zero)
        arView.scene.addAnchor(anchor)

        if let model = try? Entity.load(named: "Firework") {
            model.scale = [0.005, 0.005, 0.005]
            anchor.addChild(model)
            if model.availableAnimations.isEmpty {
                print("⚠️ Firework has no animations")
            } else {
                print("✅ Firework animations:", model.availableAnimations.count)
                model.availableAnimations.forEach {
                    model.playAnimation($0.repeat(), transitionDuration: 0.0, startsPaused: false)
                }
            }
        } else {
            print("⚠️ Firework.usdz not found")
        }

        let camera = PerspectiveCamera()
        camera.position = [0, 0, 3.0]
        camera.look(at: .zero, from: camera.position, relativeTo: nil)
        let cameraAnchor = AnchorEntity(world: .zero)
        cameraAnchor.addChild(camera)
        arView.scene.addAnchor(cameraAnchor)

        return arView
    }

    func updateUIView(_ uiView: ARView, context: Context) {}
}
