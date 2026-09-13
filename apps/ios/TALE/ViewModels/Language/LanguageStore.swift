import Foundation
import SwiftUI
import Combine

final class LanguageStore: ObservableObject {
    @Published var code: String {
        didSet {
            UserDefaults.standard.set(code, forKey: Self.storageKey)
        }
    }

    static let storageKey = "app.languageCode"

    init() {
        if let saved = UserDefaults.standard.string(forKey: Self.storageKey) {
            self.code = saved
        } else {
            self.code = Locale.current.identifier
        }
    }

    var locale: Locale {
        Locale(identifier: code)
    }

    func setLanguage(code: String) {
        self.code = code
    }
}
