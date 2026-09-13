import SwiftUI
import RealityKit
import ARKit
import Combine

struct SolidBackgroundARView: UIViewRepresentable {
    @ObservedObject var controller: ARSceneController

    // ✅ makeCoordinator は struct 側
    func makeCoordinator() -> Coordinator {
        Coordinator(controller: controller)
    }

    func makeUIView(context: Context) -> ARView {
        let arView = ARView(
            frame: .zero,
            cameraMode: .nonAR,
            automaticallyConfigureSession: false
        )

        // 背景色
        arView.environment.background = .color(
            UIColor(red: 0.60, green: 0.80, blue: 1.0, alpha: 1.0)
        )

        // デバッグ
  //      arView.debugOptions = [.showWorldOrigin, .showPhysics]

        context.coordinator.arView = arView

        setupScene(on: arView, coordinator: context.coordinator)

        // ✅ Tap Gesture 定義
        let tap = UITapGestureRecognizer(
            target: context.coordinator,
            action: #selector(Coordinator.handleTap(_:))
        )
        arView.addGestureRecognizer(tap)

        return arView
    }

    func updateUIView(_ uiView: ARView, context: Context) {}

    // MARK: - Scene Setup
    private func setupScene(on arView: ARView, coordinator: Coordinator) {
        let rootAnchor = AnchorEntity(world: .zero)
        arView.scene.addAnchor(rootAnchor)
        controller.bind(arView: arView, worldAnchor: rootAnchor)

        // --------------------------------------------------
        // 1. Lighting
        // --------------------------------------------------
        let keyLight = DirectionalLight()
        keyLight.light.intensity = 1000
        keyLight.look(at: .zero, from: [10, 10, 10], relativeTo: nil)

        let fillLight = DirectionalLight()
        fillLight.light.intensity = 600
        fillLight.look(at: .zero, from: [-10, 5, -10], relativeTo: nil)

        let backLight = DirectionalLight()
        backLight.light.intensity = 800
        backLight.look(at: .zero, from: [0, 5, -10], relativeTo: nil)

        let lightAnchor = AnchorEntity(world: .zero)
        lightAnchor.addChild(keyLight)
        lightAnchor.addChild(fillLight)
        lightAnchor.addChild(backLight)
        arView.scene.addAnchor(lightAnchor)

        // --------------------------------------------------
        // 2. Map
        // --------------------------------------------------
        if let mapURL = Bundle.main.url(forResource: "Quest_map_edited", withExtension: "usdz"),
           let questMap = try? Entity.load(contentsOf: mapURL) {

            stripAR(from: questMap)

            let mapScale: Float = 30.0
            questMap.scale = .init(repeating: mapScale)

            let bounds = questMap.visualBounds(relativeTo: nil)
            questMap.position = -bounds.center
            questMap.position.y -= 1.0
            questMap.position.x -= 2.0

            questMap.generateCollisionShapes(recursive: true)

            let mapContainer = ModelEntity()
            mapContainer.addChild(questMap)
            mapContainer.generateCollisionShapes(recursive: true)

            arView.installGestures([.translation, .rotation, .scale], for: mapContainer)
            rootAnchor.addChild(mapContainer)

            playQuestMapAnimations(in: questMap)

            controller.questMap = mapContainer
        }

        // --------------------------------------------------
        // 3. Character
        // --------------------------------------------------
        if let url = Bundle.main.url(forResource: "walk_men", withExtension: "usdz"),
           let model = try? Entity.load(contentsOf: url) {

            stripAR(from: model)

            let wrapper = Entity()
            let bounds = model.visualBounds(relativeTo: nil)
            model.position = -bounds.center
            wrapper.addChild(model)

            wrapper.scale = .init(repeating: 0.003)
            wrapper.orientation = simd_quatf(angle: .pi, axis: [0, 1, 0])

            addMenCollision(in: wrapper)

            rootAnchor.addChild(wrapper)
            controller.walkManEntity = wrapper
        }

        // --------------------------------------------------
        // 4. Camera
        // --------------------------------------------------
        let camera = PerspectiveCamera()
        let cameraAnchor = AnchorEntity(world: .zero)

        camera.position = [0, 1.5, 5.0]
        camera.look(at: [0, 1.0, 0], from: camera.position, relativeTo: nil)

        cameraAnchor.addChild(camera)
        arView.scene.addAnchor(cameraAnchor)
        controller.cameraEntity = camera
        controller.cameraAnchor = cameraAnchor

        // --------------------------------------------------
        // 5. Proximity Check
        // --------------------------------------------------
        if controller.walkManEntity != nil,
           let mapContainer = controller.questMap,
           let sakura = mapContainer.findEntity(named: "sakura") {

            coordinator.updateSubscription =
                arView.scene.subscribe(to: SceneEvents.Update.self) { [weak coordinator] _ in
                    guard let coordinator else { return }

                    let z = sakura.position(relativeTo: nil).z
                    let isInside = (-5500.0 ... -5000.0).contains(z)
                    let isCastleInside = (1500.0 ... 2000.0).contains(abs(z))

                    if isInside && !coordinator.wasInsideSakuraRange {
                        coordinator.wasInsideSakuraRange = true
                        DispatchQueue.main.async {
                            coordinator.controller?.showSakuraPopup = true
                        }
                        NotificationCenter.default.post(
                            name: Notification.Name("SakuraAction"),
                            object: nil
                        )
                    } else if !isInside && coordinator.wasInsideSakuraRange {
                        coordinator.wasInsideSakuraRange = false
                    }

                    if isCastleInside && !coordinator.wasInsideCastleRange {
                        coordinator.wasInsideCastleRange = true
                        DispatchQueue.main.async {
                            coordinator.controller?.showCastlePopup = true
                        }
                    } else if !isCastleInside && coordinator.wasInsideCastleRange {
                        coordinator.wasInsideCastleRange = false
                    }
                }
        }
    }

    // MARK: - Coordinator
    class Coordinator: NSObject {
        weak var arView: ARView?
        weak var controller: ARSceneController?
        var updateSubscription: Cancellable?
        var wasInsideSakuraRange = false
        var wasInsideCastleRange = false

        init(controller: ARSceneController) {
            self.controller = controller
        }

        @objc func handleTap(_ sender: UITapGestureRecognizer) {
            debugLog("👆 ARView tapped")
        }
    }

    // MARK: - Helpers
    private func stripAR(from entity: Entity) {
        entity.components.remove(AnchoringComponent.self)
        entity.children.forEach { stripAR(from: $0) }
    }

    private func playQuestMapAnimations(in entity: Entity) {
        func play(_ e: Entity) {
            e.availableAnimations.forEach {
                e.playAnimation($0.repeat(), transitionDuration: 0)
            }
            e.children.forEach(play)
        }
        play(entity)
    }

    private func addMenCollision(in root: Entity) {
        guard let target = root.findEntity(named: "walk_men") else { return }
        let bounds = target.visualBounds(relativeTo: root)
        let box = ShapeResource.generateBox(size: bounds.extents * 1.5)
        target.components[CollisionComponent.self] =
            CollisionComponent(shapes: [box])
    }
}
