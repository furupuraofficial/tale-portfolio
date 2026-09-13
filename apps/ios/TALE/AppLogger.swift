import Foundation
import OSLog

private let taleLogger = Logger(
    subsystem: Bundle.main.bundleIdentifier ?? "TALE",
    category: "Application"
)

/// Development diagnostics are compiled out of release builds. Values are
/// private by default so transcripts and server responses are not exposed in logs.
func debugLog(_ items: Any..., separator: String = " ") {
#if DEBUG
    let message = items.map { String(describing: $0) }.joined(separator: separator)
    taleLogger.debug("\(message, privacy: .private)")
#endif
}
