import Foundation

enum BackendConfiguration {
    static let fallbackURL = URL(string: "http://127.0.0.1:8080")!

    static func resolve(
        environment: [String: String] = ProcessInfo.processInfo.environment,
        infoDictionary: [String: Any] = Bundle.main.infoDictionary ?? [:]
    ) -> URL {
        let candidates = [
            environment["TALE_BACKEND_URL"],
            infoDictionary["TALEBackendURL"] as? String,
        ]

        for case let value? in candidates {
            let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
            guard
                let url = URL(string: trimmed),
                let scheme = url.scheme?.lowercased(),
                scheme == "http" || scheme == "https",
                url.host != nil,
                url.user == nil
            else {
                continue
            }
            return url
        }

        return fallbackURL
    }
}
