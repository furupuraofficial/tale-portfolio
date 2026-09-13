import SwiftUI
import RealityKit
import Combine
import AVFoundation
import AVKit
import WebKit
import MessageUI


struct SupportFormOverlay: View {
    @Binding var isPresented: Bool
    @State private var subject: String = ""
    @State private var message: String = ""
    @State private var showMailComposer = false
    @State private var showMailUnavailable = false

    private let toAddress = "support@example.com"

    var body: some View {
        NavigationView {
            VStack(alignment: .leading, spacing: 12) {
                Text(LocalizedStringKey("support_prompt"))
                    .font(.headline)

                TextField(LocalizedStringKey("support_subject_placeholder"), text: $subject)
                    .textFieldStyle(RoundedBorderTextFieldStyle())

                TextEditor(text: $message)
                    .frame(minHeight: 180)
                    (
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(Color.gray.opacity(0.3))
                    )

                Spacer()

                Button(action: sendTapped) {
                    Text(LocalizedStringKey("support_send_button"))
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.blue)
                        .foregroundColor(.white)
                        .cornerRadius(8)
                }
            }
            .padding()
            .navigationTitle(LocalizedStringKey("support_title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(LocalizedStringKey("support_close")) {
                        isPresented = false
                    }
                }
            }
            .sheet(isPresented: $showMailComposer) {
                MailComposer(
                    to: toAddress,
                    subject: subject,
                    body: message
                ) { _ in
                    isPresented = false
                }
            }
            .alert(isPresented: $showMailUnavailable) {
                Alert(
                    title: Text(LocalizedStringKey("support_mail_unavailable_title")),
                    message: Text(LocalizedStringKey("support_mail_unavailable_body")),
                    primaryButton: .default(Text(LocalizedStringKey("support_copy"))) {
                        UIPasteboard.general.string =
                            "\(NSLocalizedString("support_subject_placeholder", comment: "")): \(subject)\n\n\(message)"
                    },
                    secondaryButton: .cancel()
                )
            }
        }
    }

    private func sendTapped() {
        if MFMailComposeViewController.canSendMail() {
            showMailComposer = true
        } else {
            showMailUnavailable = true
        }
    }
}

struct MailComposer: UIViewControllerRepresentable {
    let to: String
    let subject: String
    let body: String
    let onFinish: (Result<MFMailComposeResult, Error>) -> Void

    class Coordinator: NSObject, MFMailComposeViewControllerDelegate {
        let parent: MailComposer
        init(parent: MailComposer) { self.parent = parent }

        func mailComposeController(
            _ controller: MFMailComposeViewController,
            didFinishWith result: MFMailComposeResult,
            error: Error?
        ) {
            if let error = error {
                parent.onFinish(.failure(error))
            } else {
                parent.onFinish(.success(result))
            }
            controller.dismiss(animated: true, completion: nil)
        }
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    func makeUIViewController(context: Context) -> MFMailComposeViewController {
        let vc = MFMailComposeViewController()
        vc.setToRecipients([to])
        vc.setSubject(subject)
        vc.setMessageBody(body, isHTML: false)
        vc.mailComposeDelegate = context.coordinator
        return vc
    }

    func updateUIViewController(
        _ uiViewController: MFMailComposeViewController,
        context: Context
    ) {}
}

