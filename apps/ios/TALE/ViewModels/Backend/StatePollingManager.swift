// BackendClient+StatePolling.swift
import Combine
import Foundation

extension BackendClient {

    // MARK: - /conversation/state polling

    func startStatePolling() {
        Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            self?.fetchState()
        }
    }

    func fetchState() {
        let url = baseURL.appendingPathComponent("/conversation/state")

        session.dataTask(with: url) { data, _, error in
            if let error = error {
                print("state error:", error)
                return
            }
            guard let data = data else { return }

            do {
                let state = try JSONDecoder().decode(StateResponse.self, from: data)
                DispatchQueue.main.async {
                    if let step = ConversationStep(rawValue: state.step) {
                        self.currentStep = step
                    } else {
                        self.currentStep = .intro
                    }
                    self.currentMode = state.mode
                }
            } catch {
                print("state decode error:", error)
            }
        }.resume()
    }
}

