import SwiftUI
import RealityKit
import Combine

struct ZashikiPreviewView: UIViewRepresentable {
    class Coordinator {
        var cancellable: Cancellable?
        var cat: Entity?
        var baseTransform: Transform = .identity
        var basePosition: SIMD3<Float> = .zero
        var stepTime: Float = 0
    }

    func makeCoordinator() -> Coordinator { Coordinator() }

    func makeUIView(context: Context) -> ARView {
        
        print("makeUIView called")
        let view = ARView(frame: .zero, cameraMode: .nonAR, automaticallyConfigureSession: false)
        view.environment.background = .color(.clear)

        print("🔍 ARPreview cameraMode:", view.cameraMode)

        let anchor = AnchorEntity()
        if let catURL = Bundle.main.url(forResource: "Lowpoly_Kimono_cat", withExtension: "usdz"),
           let cat = try? Entity.load(contentsOf: catURL) {
            cat.scale = SIMD3<Float>(repeating: 1.2)
            cat.position = SIMD3<Float>(0, 0, 0)
            cat.transform.rotation = .init() // USDZ 側の初期回転をリセットして正面に
            anchor.addChild(cat)
            context.coordinator.cat = cat
            context.coordinator.baseTransform = cat.transform
            context.coordinator.basePosition = cat.transform.translation
        }
        view.scene.addAnchor(anchor)

        let introBowDuration: Float = 20.0
        let maxBowAngle: Float = -(.pi / 3)

        context.coordinator.cancellable = view.scene.subscribe(to: SceneEvents.Update.self) { event in
            guard let cat = context.coordinator.cat else { return }
            let dt = Float(event.deltaTime)
            context.coordinator.stepTime += dt
            let t = context.coordinator.stepTime
            
            if Int(t) % 2 == 0 { print("t:", t) }

            // アニメーションを一時的に無効化して初期向きを確認
            /*
            if t <= introBowDuration {
                let progress = t / introBowDuration
                let angle = sin(progress * Float.pi) * maxBowAngle
                let baseRot = context.coordinator.baseTransform.rotation
                let bowRot  = simd_quatf(angle: angle, axis: SIMD3<Float>(1, 0, 0))

                var tr = context.coordinator.baseTransform
                tr.rotation = baseRot * bowRot
                tr.translation = context.coordinator.basePosition
                cat.transform = tr
            }
            */
        }

        return view
    }

    func updateUIView(_ uiView: ARView, context: Context) {}
}
