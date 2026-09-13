import SwiftUI

// MARK: - ContentView Language Support

extension ContentView {
    func greetingText(for language: String) -> String {
        switch language {
        case "日本語":
            return """
            ようこそ！ こんにちは、私は座敷童。
            日本では、その家に幸運をもたらす守り神（精霊）として知られているんだ。
            あなたの滞在が特別で楽しいものになるように、私がお手伝いするよ！

            私が部屋のどこにいるか探してね。
            """
        case "English":
            return "Welcome! Hello, I'm a Zashiki-warashi. In Japan, I am known as a guardian spirit who brings good fortune to the house. I'm here to help make your stay special and fun!"
        case "中文":
            return "歡迎光臨！你好，我是座敷童子。 在日本，我是以為家裡帶來好運的守護神（精靈）而聞名。 為了讓您的住宿體驗特別且愉快，我會為您提供協助！"
        case "한국어":
            return "어서 오세요! 안녕하세요, 저는 자시키와라시라고 해요. 일본에서는 집에 행운을 가져다주는 수호신(정령)으로 알려져 있어요. 여러분의 머무시는 시간이 특별하고 즐거울 수 있도록 제가 도와드릴게요!"
        case "Français":
            return "Bienvenue ! Bonjour, je suis un Zashiki-warashi. Au Japon, je suis connu comme un esprit gardien qui apporte la bonne fortune à la maison. Je suis là pour rendre votre séjour spécial et amusant !"
        case "Español":
            return "¡Bienvenido! Hola, soy un Zashiki-warashi. En Japón, se me conoce como un espíritu guardián que trae buena suerte a la casa. ¡Estoy aquí para ayudar a que tu estancia sea especial y divertida!"
        default:
            return "Move your phone around to find me!"
        }
    }

    func languageCode(for title: String) -> String {
        switch title {
        case "日本語": return "ja"
        case "English": return "en"
        case "中文": return "zh-Hans"
        case "한국어": return "ko"
        case "Français": return "fr"
        case "Español": return "es"
        default: return Locale.current.identifier
        }
    }
}
