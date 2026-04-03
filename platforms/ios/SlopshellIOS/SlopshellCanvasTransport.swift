import Foundation

final class SlopshellCanvasTransport {
    private let session: URLSession
    private let decoder = JSONDecoder()
    private var task: URLSessionWebSocketTask?
    private let onArtifact: @MainActor (SlopshellCanvasArtifact) -> Void
    private let onDisconnect: @MainActor (String) -> Void

    init(session: URLSession, onArtifact: @escaping @MainActor (SlopshellCanvasArtifact) -> Void, onDisconnect: @escaping @MainActor (String) -> Void) {
        self.session = session
        self.onArtifact = onArtifact
        self.onDisconnect = onDisconnect
    }

    func connect(baseURL: URL, sessionID: String) {
        disconnect()
        guard let wsURL = slopshellWSURL(baseURL: baseURL, path: "canvas/\(sessionID.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? sessionID)") else {
            return
        }
        let task = session.webSocketTask(with: wsURL)
        self.task = task
        task.resume()
        receiveLoop(task: task)
    }

    func disconnect() {
        task?.cancel(with: .normalClosure, reason: nil)
        task = nil
    }

    func loadSnapshot(baseURL: URL, sessionID: String) async throws {
        let url = slopshellAPIURL(baseURL: baseURL, path: "canvas/\(sessionID)/snapshot")
        let (data, _) = try await session.data(from: url)
        let snapshot = try decoder.decode(SlopshellCanvasSnapshotResponse.self, from: data)
        guard let event = snapshot.event else {
            return
        }
        await onArtifact(SlopshellCanvasArtifact(
            kind: event.kind ?? "",
            title: event.title ?? "",
            html: slopshellCanvasHTML(from: event),
            text: event.text ?? event.markdownOrText ?? ""
        ))
    }

    private func receiveLoop(task: URLSessionWebSocketTask) {
        task.receive { [weak self] result in
            guard let self else { return }
            switch result {
            case let .success(message):
                self.handle(message: message)
                self.receiveLoop(task: task)
            case let .failure(error):
                Task { @MainActor in
                    self.onDisconnect(error.localizedDescription)
                }
            }
        }
    }

    private func handle(message: URLSessionWebSocketTask.Message) {
        let data: Data
        switch message {
        case let .data(raw):
            data = raw
        case let .string(raw):
            data = Data(raw.utf8)
        @unknown default:
            return
        }
        guard let payload = try? decoder.decode(SlopshellCanvasEventPayload.self, from: data) else {
            return
        }
        let artifact = SlopshellCanvasArtifact(
            kind: payload.kind ?? "",
            title: payload.title ?? "",
            html: slopshellCanvasHTML(from: payload),
            text: payload.text ?? payload.markdownOrText ?? ""
        )
        Task { @MainActor in
            self.onArtifact(artifact)
        }
    }
}
