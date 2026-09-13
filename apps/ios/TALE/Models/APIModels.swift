import Foundation

// MARK: - APIレスポンスモデル

struct StateResponse: Decodable {
    let step: String
    let mode: Int
    let userName: String?
}

struct HistoryItem: Decodable {
    let role: String?
    let content: String?
    let sessionId: String?
    let step: String?
    let timestamp: String?
}

// MARK: - AR アクション

struct ARAction: Decodable {
    let type: String?
    let target: String?
    let params: [String: String]?
}
