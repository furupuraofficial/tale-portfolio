import SwiftUI

@main
struct TALEApp: App {
    private let isRunningUnitTests =
        ProcessInfo.processInfo.environment["XCTestConfigurationFilePath"] != nil

    var body: some Scene {
        WindowGroup {
            if isRunningUnitTests {
                Color.clear
            } else {
                ContentView()
            }
        }
    }
}
