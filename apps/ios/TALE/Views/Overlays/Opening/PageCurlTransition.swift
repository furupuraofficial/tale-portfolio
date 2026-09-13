import SwiftUI
import UIKit

struct PageCurlTransition<Content: View>: UIViewRepresentable {
    var trigger: Bool
    @ViewBuilder var content: Content

    class Coordinator {
        var hosting: UIHostingController<Content>
        var lastTrigger: Bool
        init(content: Content, trigger: Bool) {
            hosting = UIHostingController(rootView: content)
            hosting.view.backgroundColor = .clear
            lastTrigger = trigger
        }
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(content: content, trigger: trigger)
    }

    func makeUIView(context: Context) -> UIView {
        let view = UIView()
        let hosted = context.coordinator.hosting.view!
        hosted.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(hosted)
        NSLayoutConstraint.activate([
            hosted.topAnchor.constraint(equalTo: view.topAnchor),
            hosted.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            hosted.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            hosted.trailingAnchor.constraint(equalTo: view.trailingAnchor)
        ])
        return view
    }

    func updateUIView(_ uiView: UIView, context: Context) {
        context.coordinator.hosting.rootView = content
        if trigger != context.coordinator.lastTrigger {
            UIView.transition(
                with: context.coordinator.hosting.view,
                duration: 1.0,
                options: [.transitionCurlDown]
            ) {
            }
            context.coordinator.lastTrigger = trigger
        }
    }
}
