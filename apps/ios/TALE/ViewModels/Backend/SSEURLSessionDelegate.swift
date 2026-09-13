// BackendClient+URLSessionDelegate.swift
import Combine
import Foundation

extension BackendClient: URLSessionDataDelegate {

    func urlSession(_ session: URLSession,
                    dataTask: URLSessionDataTask,
                    didReceive data: Data) {
        let separator = Data("\n\n".utf8)

        if dataTask === historyTask {
            historyBuffer.append(data)
            while let range = historyBuffer.range(of: separator) {
                let eventData = historyBuffer.subdata(in: 0..<range.lowerBound)
                historyBuffer.removeSubrange(0..<range.upperBound)
                handleHistoryEvent(eventData)
            }
        } else if dataTask === audioTask {
            audioBuffer.append(data)
            while let range = audioBuffer.range(of: separator) {
                let eventData = audioBuffer.subdata(in: 0..<range.lowerBound)
                audioBuffer.removeSubrange(0..<range.upperBound)
                handleAudioEvent(eventData)
            }
        }
    }

    func urlSession(_ session: URLSession,
                    dataTask: URLSessionDataTask,
                    didReceive response: URLResponse,
                    completionHandler: @escaping (URLSession.ResponseDisposition) -> Void) {
        if let http = response as? HTTPURLResponse {
            debugLog("🔎 SSE response status:", http.statusCode,
                  "url:", http.url?.absoluteString ?? "-")
        }
        completionHandler(.allow)
    }

    func urlSession(_ session: URLSession,
                    task: URLSessionTask,
                    didCompleteWithError error: Error?) {
        if task === historyTask {
            historyTask = nil
            if let err = error as? URLError, err.code == .cancelled {
                debugLog("historyTask cancelled, not restarting SSE")
                return
            }
            DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                self.startHistoryStream()
            }
        } else if task === audioTask {
            audioTask = nil
            if let err = error as? URLError, err.code == .cancelled {
                debugLog("audioTask cancelled, not restarting SSE")
                return
            }
            DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                self.startAudioStream()
            }
        }
    }
}
