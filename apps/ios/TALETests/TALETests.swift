import XCTest
@testable import TALE

final class TALETests: XCTestCase {
    func testBackendURLUsesEnvironmentBeforeInfoDictionary() {
        let result = BackendConfiguration.resolve(
            environment: ["TALE_BACKEND_URL": "https://api.example.com"],
            infoDictionary: ["TALEBackendURL": "https://plist.example.com"]
        )

        XCTAssertEqual(result.absoluteString, "https://api.example.com")
    }

    func testBackendURLUsesInfoDictionaryWhenEnvironmentIsMissing() {
        let result = BackendConfiguration.resolve(
            environment: [:],
            infoDictionary: ["TALEBackendURL": " https://plist.example.com/base "]
        )

        XCTAssertEqual(result.absoluteString, "https://plist.example.com/base")
    }

    func testBackendURLFallsBackForUnsafeOrInvalidValues() {
        let result = BackendConfiguration.resolve(
            environment: ["TALE_BACKEND_URL": "file:///tmp/tale"],
            infoDictionary: ["TALEBackendURL": "not a URL"]
        )

        XCTAssertEqual(result, BackendConfiguration.fallbackURL)
    }
}
