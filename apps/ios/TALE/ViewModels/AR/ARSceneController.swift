import SwiftUI
import RealityKit
import ARKit
import Combine
import AVFoundation
import AVKit
import WebKit
import MessageUI
import UIKit
import simd

final class ARSceneController: ObservableObject {
    weak var arView: ARView?
    weak var worldAnchor: AnchorEntity?
    var questMap: Entity?
    var walkManEntity: Entity?
    weak var cameraEntity: PerspectiveCamera?
    weak var cameraAnchor: AnchorEntity?
    @Published var showSakuraPopup = false
    @Published var showCastlePopup = false
    @Published var showCorrectOverlay = false
    var onZashikiReady: (() -> Void)?   // ✅ 追加
    private var answerTargetsAnchor: Entity?
    private var targetPrototype: Entity?
    private var lastRaycastHit: SIMD3<Float>?
    private var sfxPlayers: [String: AVAudioPlayer] = [:]
    
    var zashikiEntity: Entity? {
        didSet {
            zashikiBaseOrientation = zashikiEntity?.orientation
            zashikiBasePosition = zashikiEntity?.position
            print("✅ zashikiEntity ready:", zashikiEntity != nil)
            if zashikiEntity != nil {
                onZashikiReady?()       // ✅ ready になった瞬間に通知
            }
        }
    }
    private var zashikiBaseOrientation: simd_quatf?
    private var zashikiBasePosition: SIMD3<Float>?
    private var walkAnimationController: AnimationPlaybackController?
    private var walkStopWorkItem: DispatchWorkItem?
    private var walkBaseOrientation: simd_quatf?

    func animateCameraIntro() {
        guard let camera = cameraEntity else { return }

        let endPos: SIMD3<Float> = [0, 1.5, 10.0]
        let startPos: SIMD3<Float> = [0, 15.0, 20.0]

        camera.position = startPos
        camera.look(at: [0, 1.0, -8.0], from: startPos, relativeTo: nil)

        camera.move(
            to: Transform(
                scale: .one,
                rotation: camera.orientation,
                translation: endPos
            ),
            relativeTo: camera.parent,
            duration: 1.2,
            timingFunction: .easeInOut
        )
    }
    
    func bind(arView: ARView, worldAnchor: AnchorEntity) {
        if self.arView != nil {
            print("⚠️ bind ignored (already bound)")
            return
        }
        
        self.arView = arView
        self.worldAnchor = worldAnchor
        print("🔗 bind ARView:", ObjectIdentifier(arView))
    }
    
    func move(dx: Float, dz: Float) {
        guard let map = questMap else {
            print("questMap is nil")
            return
        }
        
        // マップの移動をアニメーション
        var target = map.transform
        target.translation.x += dx
        target.translation.z += dz
        let moveDuration: Double = 0.5
        map.move(to: target, relativeTo: map.parent, duration: moveDuration, timingFunction: .easeInOut)
        
        // クエストマップ上の歩行キャラを移動中だけ歩かせる＆向きを合わせる
        if let walker = walkManEntity {
            let baseOrientation = walkBaseOrientation ?? walker.orientation
            walkBaseOrientation = baseOrientation
            if dx != 0 || dz != 0 {
                let angle = atan2(-dx, -dz)
                let facing = simd_quatf(angle: angle, axis: SIMD3<Float>(0, 1, 0))
                var faceTransform = walker.transform
                faceTransform.rotation = facing
                walker.move(to: faceTransform, relativeTo: walker.parent, duration: 0.2, timingFunction: .easeInOut)
            }
            startWalkAnimation(for: moveDuration + 0.2)
        }
        
        // 座敷童を進行方向に向けて戻す
        guard dx != 0 || dz != 0,
              let zashiki = zashikiEntity else { return }
        let baseOrientation = zashikiBaseOrientation ?? zashiki.orientation
        zashikiBaseOrientation = baseOrientation
        
        // 進行方向のYaw角を算出（Z前方、X右手）
        let angle = atan2(dx, dz)
        let facing = simd_quatf(angle: angle, axis: SIMD3<Float>(0, 1, 0))
        
        var faceTransform = zashiki.transform
        faceTransform.rotation = facing
        zashiki.move(to: faceTransform, relativeTo: zashiki.parent, duration: 0.25, timingFunction: .easeInOut)
        
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.6) {
            var restore = zashiki.transform
            restore.rotation = baseOrientation
            zashiki.move(to: restore, relativeTo: zashiki.parent, duration: 0.25, timingFunction: .easeInOut)
        }
        
        print("moved to:", target.translation)
    }
    
    private func startWalkAnimation(for duration: Double) {
        guard let walker = walkManEntity else {
            print("⚠️ walkManEntity is nil; cannot play walk animation")
            return
        }
        guard let animation = walker.availableAnimations.first else {
            print("⚠️ walkManEntity has no available animations")
            return
        }
        
        walkStopWorkItem?.cancel()
        walkAnimationController?.stop()
        
        walkAnimationController = walker.playAnimation(
            animation.repeat(),
            transitionDuration: 0.05,
            startsPaused: false
        )
        
        let stopItem = DispatchWorkItem { [weak self] in
            self?.stopWalkAnimation()
        }
        walkStopWorkItem = stopItem
        DispatchQueue.main.asyncAfter(deadline: .now() + duration, execute: stopItem)
    }
    
    private func stopWalkAnimation() {
        walkAnimationController?.stop()
        walkAnimationController = nil
    }
    
    // MARK: - Animations
    
    func perform(action name: String) {
        print("🎭 perform(action:) called with name: \(name)")
        guard let zashiki = zashikiEntity else {
            print("⚠️ zashikiEntity is nil in perform(action:)")
            return
        }
        let key = name.lowercased()
        print("🎭 Performing action key: \(key)")
        switch key {
        case "wave":
            print("🎭 Executing wave animation")
            playWave(on: zashiki)
        case "show_speech_bubble":
            print("🎭 Executing speech lean animation")
            playSpeechLean(on: zashiki)
        case "bow":
            print("🎭 Executing bow animation")
            playBow(on: zashiki)
        default:
            print("⚠️ Unknown action key: \(key)")
            break
        }
    }
    
    private func playWave(on entity: Entity) {
        let base = zashikiBaseOrientation ?? entity.orientation
        zashikiBaseOrientation = base
        let yawAngles: [Float] = [0.12, -0.12, 0.1, 0] // rad
        var delay: Double = 0
        for angle in yawAngles {
            let rot = base * simd_quatf(angle: angle, axis: SIMD3<Float>(0, 1, 0))
            DispatchQueue.main.asyncAfter(deadline: .now() + delay) {
                var t = entity.transform
                t.rotation = rot
                entity.move(to: t, relativeTo: entity.parent, duration: 0.18, timingFunction: .easeInOut)
            }
            delay += 0.30
        }
    }
    
    private func playSpeechLean(on entity: Entity) {
        let base = zashikiBaseOrientation ?? entity.orientation
        zashikiBaseOrientation = base
        let pitch: Float = -0.18 // forward lean
        
        let forward = base * simd_quatf(angle: pitch, axis: SIMD3<Float>(1, 0, 0))
        var t = entity.transform
        t.rotation = forward
        entity.move(to: t, relativeTo: entity.parent, duration: 0.18, timingFunction: .easeInOut)
        
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.22) {
            var restore = entity.transform
            restore.rotation = base
            entity.move(to: restore, relativeTo: entity.parent, duration: 0.2, timingFunction: .easeInOut)
        }
    }
    
    private func playBow(on entity: Entity) {
        let basePos = zashikiBasePosition ?? entity.position
        zashikiBasePosition = basePos
        let offsets: [Float] = [0.05, -0.03, 0]
        var delay: Double = 0
        for dy in offsets {
            DispatchQueue.main.asyncAfter(deadline: .now() + delay) {
                var t = entity.transform
                t.translation = basePos + SIMD3<Float>(0, dy, 0)
                entity.move(to: t, relativeTo: entity.parent, duration: 0.16, timingFunction: .easeInOut)
            }
            delay += 0.16
        }
    }
    
    // MARK: - Quiz Targets
    
    struct AnswerComponent: Component {
        let isCorrect: Bool
    }
    
    func showAnswerTargets() {
        guard let arView else {
            print("⚠️ arView is nil; cannot place answer targets")
            return
        }
        
        guard let prototype = loadTargetPrototype() else {
            print("⚠️ target.usdz not found or failed to load")
            return
        }
        
        answerTargetsAnchor?.removeFromParent()
        answerTargetsAnchor = nil
        
        let container: Entity
        if let worldAnchor {
            let newContainer = Entity()
            newContainer.transform = Transform(matrix: arView.cameraTransform.matrix)
            worldAnchor.addChild(newContainer)
            container = newContainer
        } else {
            let newAnchor = AnchorEntity(world: .zero)
            arView.scene.addAnchor(newAnchor)
            container = newAnchor
        }
        answerTargetsAnchor = container
        
        let placement = defaultTargetPlacement()
        
        let targets = [
            makeAnswerTarget(
                id: "answer_1",
                numberText: "1",
                position: placement.center + placement.right * -0.25,
                isCorrect: true,
                prototype: prototype
            ),
            makeAnswerTarget(
                id: "answer_2",
                numberText: "2",
                position: placement.center,
                isCorrect: false,
                prototype: prototype
            ),
            makeAnswerTarget(
                id: "answer_3",
                numberText: "3",
                position: placement.center + placement.right * 0.25,
                isCorrect: false,
                prototype: prototype
            )
        ]
        
        targets.forEach { container.addChild($0) }
        print("ARView:", ObjectIdentifier(arView))
    }
    
    func clearAnswerTargets() {
        answerTargetsAnchor?.removeFromParent()
        answerTargetsAnchor = nil
    }
    
    private func makeAnswerTarget(
        id: String,
        numberText: String,
        position: SIMD3<Float>,
        isCorrect: Bool,
        prototype: Entity
    ) -> Entity {
        let container = Entity()
        container.name = id
        container.transform.translation = position
        container.transform.rotation = .init(angle: 0, axis: [0, 1, 0])
        container.scale = SIMD3<Float>(repeating: 0.2)
        container.components[AnswerComponent.self] = AnswerComponent(isCorrect: isCorrect)
        
        let model = prototype.clone(recursive: true)
        stripAnchoring(from: model)
        let bounds = model.visualBounds(relativeTo: nil)
        model.position = -bounds.center
        container.addChild(model)
        container.generateCollisionShapes(recursive: true)
        
        let labelMesh = MeshResource.generateText(
            numberText,
            extrusionDepth: 0.01,
            font: .systemFont(ofSize: 0.2, weight: .bold),
            containerFrame: .zero,
            alignment: .center,
            lineBreakMode: .byWordWrapping
        )
        let labelMaterial = UnlitMaterial(color: .white)
        let label = ModelEntity(mesh: labelMesh, materials: [labelMaterial])
        label.position = [0, 0.35, 0]
        container.addChild(label)
        
        print("✅ placed answer target:", id, "at", position, "bounds:", bounds)
        return container
    }
    
    private func defaultTargetPlacement() -> (center: SIMD3<Float>, right: SIMD3<Float>) {
        let center: SIMD3<Float> = [0, -0.15, -1.0]
        let right: SIMD3<Float> = [1, 0, 0]
        return (center: center, right: right)
    }
    
    private func loadTargetPrototype() -> Entity? {
        if let targetPrototype { return targetPrototype }
        guard let url = Bundle.main.url(forResource: "target", withExtension: "usdz") else {
            return nil
        }
        let loaded = try? Entity.load(contentsOf: url)
        if let loaded {
            stripAnchoring(from: loaded)
        }
        targetPrototype = loaded
        return loaded
    }
    
    private func stripAnchoring(from entity: Entity) {
        entity.components.remove(AnchoringComponent.self)
        for child in entity.children {
            stripAnchoring(from: child)
        }
    }
    
    // MARK: - Raycast Shot
    
    func fire() {
        performRaycastShot()
    }
    
    private func performRaycastShot() {
        guard let arView else { return }
        
        let center = CGPoint(x: arView.bounds.midX, y: arView.bounds.midY)
        guard let ray = arView.ray(through: center) else {
            print("❌ ray(through:) failed")
            return
        }
        
        let hits = arView.scene.raycast(
            origin: ray.origin,
            direction: ray.direction,
            length: 10,
            query: .nearest,
            mask: .all
        )
        
        if let hit = hits.first {
            lastRaycastHit = hit.position
            fireProjectile(from: ray.origin, to: hit.position)
            handleHit(entity: hit.entity)
        } else {
            let missPoint = ray.origin + ray.direction * 6
            lastRaycastHit = missPoint
            fireProjectile(from: ray.origin, to: missPoint)
            print("❌ 何にも当たらなかった")
        }
    }
    
    private func handleHit(entity: Entity) {
        var current: Entity? = entity
        while let check = current {
            if let answer = check.components[AnswerComponent.self] {
                if answer.isCorrect {
                    missionClear(target: check)
                } else {
                    wrongAnswer(target: check)
                }
                break
            }
            current = check.parent
        }
    }
    
    private func missionClear(target: Entity) {
        print("🎉 正解！")

        playSfx(name: "explosion", ext: "mp3", volume: 0.9)

        DispatchQueue.main.async {
            self.showCorrectOverlay = true
        }

        target.removeFromParent()
    }

    
    private func wrongAnswer(target: Entity) {
        playSfx(name: "bounds", ext: "mp3", volume: 0.8)
        let original = target.position
        target.move(
            to: Transform(
                scale: .one,
                rotation: target.orientation,
                translation: original + SIMD3<Float>(0.1, 0, 0)
            ),
            relativeTo: target.parent,
            duration: 0.1
        )
    }
    
    private func playSfx(name: String, ext: String, volume: Float) {
        let key = "\(name).\(ext)"
        if let existing = sfxPlayers[key] {
            existing.currentTime = 0
            existing.volume = volume
            existing.play()
            return
        }
        guard let url = Bundle.main.url(forResource: name, withExtension: ext) else {
            print("⚠️ SFX not found in bundle:", key)
            return
        }
        do {
            let player = try AVAudioPlayer(contentsOf: url)
            player.volume = volume
            player.prepareToPlay()
            player.play()
            sfxPlayers[key] = player
        } catch {
            print("⚠️ SFX play failed:", key, error)
        }
    }
    
    private func fireProjectile(from start: SIMD3<Float>, to hit: SIMD3<Float>) {
        guard let arView else { return }

        // 🔑 絶対に同じ world
        let parent: Entity
        if let answerTargetsAnchor {
            parent = answerTargetsAnchor
        } else if let worldAnchor {
            parent = worldAnchor
        } else {
            let anchor = AnchorEntity(world: .zero)
            arView.scene.addAnchor(anchor)
            parent = anchor
        }

        let projectile = ModelEntity(
            mesh: .generateSphere(radius: 0.05),
            materials: [UnlitMaterial(color: .white)]
        )

        // ⚠️ ワールド → ローカル変換
        projectile.position = parent.convert(position: start, from: nil)
        parent.addChild(projectile)

        let localHit = parent.convert(position: hit, from: nil)

        projectile.move(
            to: Transform(translation: localHit),
            relativeTo: parent,
            duration: 0.25,
            timingFunction: .easeOut
        )

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
            projectile.removeFromParent()
        }
    }

    private func spawnFireworkAndComplete(
        at position: SIMD3<Float>,
        parent: Entity
    ) {
        let container = Entity()
        container.position = position
        parent.addChild(container)

        // 🎆 Firework
        if let url = Bundle.main.url(forResource: "Firework", withExtension: "usdz"),
           let firework = try? Entity.load(contentsOf: url) {

            stripAnchoring(from: firework)

            let bounds = firework.visualBounds(relativeTo: nil)
            firework.position = -bounds.center
            firework.scale = .init(repeating: 0.5)

            container.addChild(firework)

            firework.availableAnimations.forEach {
                firework.playAnimation($0.repeat(), transitionDuration: 0)
            }
        }

        // 🏆 Complete Text
        let textMesh = MeshResource.generateText(
            "Complete",
            extrusionDepth: 0.02,
            font: .systemFont(ofSize: 0.25, weight: .bold),
            containerFrame: .zero,
            alignment: .center,
            lineBreakMode: .byWordWrapping
        )

        var mat = UnlitMaterial()
        mat.color = .init(tint: .yellow)

        let text = ModelEntity(mesh: textMesh, materials: [mat])
        text.position = [0, 0.6, 0]
        container.addChild(text)

        // ⏳ 後始末
        DispatchQueue.main.async {
            self.showCorrectOverlay = true
        }
    }
}

extension ARSceneController: ARActionManaging {
    var isZashikiReady: Bool { zashikiEntity != nil }
    func playAnimation(named name: String) {
        print("🎭 ARSceneController.playAnimation called with name: \(name)")
        if zashikiEntity == nil {
            print("⚠️ zashikiEntity is nil, cannot play animation")
            return
        }
        perform(action: name)
    }

    func showSpeechBubble() {
        print("🎭 ARSceneController.showSpeechBubble called")
        if zashikiEntity == nil {
            print("⚠️ zashikiEntity is nil, cannot show speech bubble")
            return
        }
        perform(action: "show_speech_bubble")
    }

    func hideSpeechBubble() {
        print("🎭 ARSceneController.hideSpeechBubble called")
        guard let zashiki = zashikiEntity else {
            print("⚠️ zashikiEntity is nil, cannot hide speech bubble")
            return
        }
        let baseOrientation = zashikiBaseOrientation ?? zashiki.orientation
        var restore = zashiki.transform
        restore.rotation = baseOrientation
        zashiki.move(to: restore, relativeTo: zashiki.parent, duration: 0.2, timingFunction: .easeInOut)
    }

    func setIdle() {
        print("🎭 ARSceneController.setIdle called")
        guard let zashiki = zashikiEntity else {
            print("⚠️ zashikiEntity is nil, cannot set idle")
            return
        }
        let baseOrientation = zashikiBaseOrientation ?? zashiki.orientation
        let basePosition = zashikiBasePosition ?? zashiki.position

        var t = zashiki.transform
        t.rotation = baseOrientation
        t.translation = basePosition
        zashiki.move(to: t, relativeTo: zashiki.parent, duration: 0.25, timingFunction: .easeInOut)
    }
}
