import Foundation

// MARK: - APIレスポンスモデル

nonisolated struct StateResponse: Decodable, Sendable {
    let step: String
    let mode: Int
    let userName: String?
}

nonisolated struct HistoryItem: Decodable, Sendable {
    let role: String?
    let content: String?
    let sessionId: String?
    let step: String?
    let timestamp: String?
}

// MARK: - AR アクション

nonisolated struct ARAction: Decodable, Sendable {
    let type: String?
    let target: String?
    let params: [String: String]?
}
