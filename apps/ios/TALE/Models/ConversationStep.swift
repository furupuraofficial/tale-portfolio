import Foundation

// MARK: - 会話ステップ（BackendのSTATEに対応）

enum ConversationStep: String, Codable {
    case intro        = "STEP_INTRO"
    case askName      = "STEP_ASK_NAME"
    case greetByName  = "STEP_GREET_BY_NAME"
    case startGuide   = "STEP_START_GUIDE"
    case freeQuestion = "STEP_FREE_QUESTION"
    case askRule      = "STEP_ASK_RULE"
    case askShop      = "STEP_ASK_SHOP"
}
