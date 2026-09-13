// BackendClient+ConversationControl.swift
import Combine
import Foundation

extension BackendClient {
    
    // MARK: - /conversation/start
    
    func startConversation(languageCode: String? = nil) {
        resetForNewSession(sessionId: nil)
        
        if let lang = languageCode, !lang.isEmpty {
            selectedLanguageCode = lang
        }
        
        // 既にSSEが繋がっている場合は張り直さない（多重接続・二重受信の原因になる）
        if historyTask == nil {
            startHistoryStream()
        }
        
        var request = URLRequest(url: baseURL.appendingPathComponent("/conversation/start"))
        request.httpMethod = "POST"
        let body: [String: String] = ["language": selectedLanguageCode]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        session.dataTask(with: request) { _, response, error in
            if let error = error {
                debugLog("conversation/start error:", error)
                self.retryStartConversation()
                return
            }
            if let http = response as? HTTPURLResponse, http.statusCode >= 400 {
                debugLog("conversation/start failed:", http.statusCode)
                self.retryStartConversation()
            }
        }.resume()
    }
    
    func setLanguage(code: String) {
        selectedLanguageCode = code
        
        var request = URLRequest(url: baseURL.appendingPathComponent("/conversation/language"))
        request.httpMethod = "POST"
        let body: [String: String] = ["language": code]
        request.httpBody = try? JSONSerialization.data(withJSONObject: body, options: [])
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        session.dataTask(with: request) { data, response, error in
            if let error = error {
                debugLog("setLanguage error:", error)
                return
            }
            if let http = response as? HTTPURLResponse, http.statusCode >= 400 {
                debugLog("setLanguage failed with status:", http.statusCode)
                return
            }
            if let data,
               let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
               let lang = json["language"] as? String {
                DispatchQueue.main.async {
                    self.selectedLanguageCode = lang
                }
            }
        }.resume()
    }
    
    func pauseConversation() {
        var request = URLRequest(url: baseURL.appendingPathComponent("/conversation/pause"))
        request.httpMethod = "POST"
        session.dataTask(with: request).resume()
    }
    
    // MARK: /conversation/resume
    
    func resumeConversation() {
        var request = URLRequest(url: baseURL.appendingPathComponent("/conversation/resume"))
        request.httpMethod = "POST"
        session.dataTask(with: request).resume()
    }
    
    private func retryStartConversation() {
        guard startRetryCount < 3 else { return }
        startRetryCount += 1
        DispatchQueue.main.asyncAfter(deadline: .now() + 3) {
            self.startConversation()
        }
    }
}
