import SwiftUI
import RealityKit
import ARKit
import Combine

struct ARViewContainer: UIViewRepresentable {
    @ObservedObject var controller: ARSceneController

    class Coordinator: NSObject, ARSessionDelegate {
        let controller: ARSceneController
        weak var arView: ARView?
        var placed = false

        // ふわふわ用
        var floatCancellable: Cancellable?
        var floatTime: Float = 0
        var floatingEntity: Entity?
        var basePosition: SIMD3<Float> = .zero

        init(controller: ARSceneController) {
            self.controller = controller
        }

        func session(_ session: ARSession, didAdd anchors: [ARAnchor]) {
            guard !placed, let arView else { return }

            // 最初に見つかった水平面に1回だけ置く
            guard let plane = anchors
                .compactMap({ $0 as? ARPlaneAnchor })
                .first(where: { $0.alignment == .horizontal })
            else { return }

            // モデルを1回だけロード（アプリバンドル内のUSDZ）
            guard let modelURL = Bundle.main.url(forResource: "Lowpoly_Kimono_cat", withExtension: "usdz"),
                  let model = try? Entity.load(contentsOf: modelURL) else {
                debugLog("⚠️ Lowpoly_Kimono_cat load failed (bundle URL missing or load error)")
                return
            }
            model.scale = SIMD3<Float>(repeating: 0.5) // 小さければ 0.2〜2.0で調整
            model.transform.rotation = .init() // USDZの初期回転をリセット

            // 補正用コンテナ（ここに回転＆ふわふわを入れる）
            let container = Entity()

            container.addChild(model)

            // 平面アンカーに追従する AnchorEntity
            let anchorEntity = AnchorEntity(anchor: plane)
            container.position = .zero
            anchorEntity.addChild(container)
            arView.scene.addAnchor(anchorEntity)

            // ふわふわ対象を保持
            floatingEntity = container
            basePosition = container.position
            startFloating(on: arView)

            placed = true
            debugLog("✅ Lowpoly_Kimono_cat placed on plane (floating)")

            // ARアクション用に登録
            controller.zashikiEntity = container
        }

        private func startFloating(on arView: ARView) {
            // 既に動いていたら止める
            floatCancellable?.cancel()
            floatTime = 0

            // パラメータ（好みで調整）
            let amplitude: Float = 0.1   // 上下幅（m）
            let speed: Float = 1.5        // 速さ（大きいほど速い）

            floatCancellable = arView.scene.subscribe(to: SceneEvents.Update.self) { [weak self] event in
                guard let self, let e = self.floatingEntity else { return }
                self.floatTime += Float(event.deltaTime)

                var p = self.basePosition
                p.y += sin(self.floatTime * speed) * amplitude
                e.position = p
            }
        }
    }

    func makeCoordinator() -> Coordinator { Coordinator(controller: controller) }

    func makeUIView(context: Context) -> ARView {
        let arView = ARView(frame: .zero, cameraMode: .ar, automaticallyConfigureSession: false)

        context.coordinator.arView = arView
        let worldAnchor = AnchorEntity(world: .zero)
        arView.scene.addAnchor(worldAnchor)
        controller.bind(arView: arView, worldAnchor: worldAnchor)
        arView.session.delegate = context.coordinator

        let config = ARWorldTrackingConfiguration()
        config.environmentTexturing = .automatic
        
        config.planeDetection = [.horizontal, .vertical] // 置くのは水平のみ拾ってるのでOK
        arView.session.run(config, options: [.resetTracking, .removeExistingAnchors])
        
        debugLog("🟢 makeUIView ARView:", ObjectIdentifier(arView))

        return arView
    }

    func updateUIView(_ uiView: ARView, context: Context) {}

    static func dismantleUIView(_ uiView: ARView, coordinator: Coordinator) {
        coordinator.floatCancellable?.cancel()
        uiView.session.pause()
        uiView.session.delegate = nil
    }
}
