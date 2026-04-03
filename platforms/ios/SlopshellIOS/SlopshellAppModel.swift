import Foundation

@MainActor
final class SlopshellAppModel: ObservableObject {
    @Published var serverURLString = "http://127.0.0.1:8420"
    @Published var password = ""
    @Published var composerText = ""
    @Published var messages: [SlopshellRenderedMessage] = []
    @Published var canvas = SlopshellCanvasArtifact(kind: "", title: "", html: "<p style=\"margin:24px; font: -apple-system-body;\">Connect to a Slopshell server to load the canvas.</p>", text: "")
    @Published var workspaces: [SlopshellWorkspace] = []
    @Published var selectedWorkspaceID = ""
    @Published var statusText = "Disconnected"
    @Published var lastError = ""
    @Published var isRecording = false
    @Published var inkRequestsResponse = true
    @Published var isDialogueModeActive = false
    @Published var isAwaitingAssistantResponse = false
    @Published var companionEnabled = false
    @Published var companionIdleSurface = SlopshellCompanionIdleSurface.robot.rawValue
    @Published var companionRuntimeState = SlopshellDialogueRuntimeState.idle.rawValue

    let discovery = SlopshellServerDiscovery()

    private let session: URLSession
    private lazy var chatTransport = SlopshellChatTransport(session: session, onEvent: { [weak self] event in
        self?.handleChatEvent(event)
    }, onDisconnect: { [weak self] message in
        self?.statusText = "Chat disconnected"
        self?.lastError = message
    })
    private lazy var canvasTransport = SlopshellCanvasTransport(session: session, onArtifact: { [weak self] artifact in
        self?.canvas = artifact
    }, onDisconnect: { [weak self] message in
        self?.statusText = "Canvas disconnected"
        self?.lastError = message
    })
    private lazy var audioCapture = SlopshellAudioCapture(onChunk: { [weak self] data in
        Task {
            await self?.sendAudioChunk(data)
        }
    }, onStateChange: { [weak self] running, message in
        self?.isRecording = running
        if message.isEmpty == false {
            self?.lastError = message
        }
    })

    private var activeWorkspace: SlopshellWorkspace?
    private var restoreCompanionEnabledOnExit: Bool?

    init() {
        let config = URLSessionConfiguration.default
        config.httpCookieAcceptPolicy = .always
        config.httpCookieStorage = HTTPCookieStorage.shared
        config.waitsForConnectivity = true
        self.session = URLSession(configuration: config)
        discovery.start()
    }

    var dialoguePresentation: SlopshellDialogueModePresentation {
        SlopshellDialogueModePresentation(
            isActive: isDialogueModeActive,
            isRecording: isRecording,
            isAwaitingAssistant: isAwaitingAssistantResponse,
            companionEnabled: companionEnabled,
            idleSurface: companionIdleSurface,
            runtimeState: companionRuntimeState
        )
    }

    func useDiscoveredServer(_ server: SlopshellDiscoveredServer) {
        serverURLString = server.baseURLString
    }

    func connect() async {
        guard let baseURL = normalizedBaseURL() else {
            lastError = "Enter a valid server URL."
            return
        }
        do {
            try await loginIfNeeded(baseURL: baseURL)
            let response = try await loadWorkspaces(baseURL: baseURL)
            workspaces = response.workspaces
            if let workspace = response.workspaces.first(where: { $0.id == response.activeWorkspaceID }) ?? response.workspaces.first {
                selectedWorkspaceID = workspace.id
                activeWorkspace = workspace
                try await loadHistory(baseURL: baseURL, workspace: workspace)
                try await attachRealtime(baseURL: baseURL, workspace: workspace)
                statusText = "Connected to \(workspace.name)"
            } else {
                statusText = "Authenticated"
            }
        } catch {
            lastError = error.localizedDescription
            statusText = "Connection failed"
        }
    }

    func switchWorkspace() async {
        guard let baseURL = normalizedBaseURL() else {
            return
        }
        if let workspace = activeWorkspace {
            await stopDialogueMode(baseURL: baseURL, workspace: workspace, restoreCompanion: true)
        }
        guard let workspace = workspaces.first(where: { $0.id == selectedWorkspaceID }) else {
            return
        }
        do {
            activeWorkspace = workspace
            try await loadHistory(baseURL: baseURL, workspace: workspace)
            try await attachRealtime(baseURL: baseURL, workspace: workspace)
            statusText = "Connected to \(workspace.name)"
        } catch {
            lastError = error.localizedDescription
        }
    }

    func toggleDialogueMode() async {
        guard let baseURL = normalizedBaseURL(), let workspace = activeWorkspace else {
            return
        }
        if isDialogueModeActive {
            await stopDialogueMode(baseURL: baseURL, workspace: workspace, restoreCompanion: true)
            return
        }
        do {
            restoreCompanionEnabledOnExit = companionEnabled
            try await updateLivePolicy(baseURL: baseURL, policy: "dialogue")
            if companionEnabled == false {
                let cfg = try await updateCompanionConfig(
                    baseURL: baseURL,
                    workspace: workspace,
                    patch: SlopshellCompanionConfigPatch(companionEnabled: true, idleSurface: nil)
                )
                applyCompanionConfig(cfg)
            }
            isDialogueModeActive = true
            isAwaitingAssistantResponse = false
            statusText = "Dialogue mode on"
        } catch {
            lastError = error.localizedDescription
        }
    }

    func setDialogueIdleSurface(_ surface: SlopshellCompanionIdleSurface) async {
        guard let baseURL = normalizedBaseURL(), let workspace = activeWorkspace else {
            companionIdleSurface = surface.rawValue
            return
        }
        do {
            let cfg = try await updateCompanionConfig(
                baseURL: baseURL,
                workspace: workspace,
                patch: SlopshellCompanionConfigPatch(companionEnabled: nil, idleSurface: surface.rawValue)
            )
            applyCompanionConfig(cfg)
            statusText = surface == .black ? "Black dialogue surface ready" : "Robot dialogue surface ready"
        } catch {
            lastError = error.localizedDescription
        }
    }

    func sendComposerMessage() async {
        guard let baseURL = normalizedBaseURL(), let workspace = activeWorkspace else {
            return
        }
        let text = composerText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard text.isEmpty == false else {
            return
        }
        composerText = ""
        do {
            var request = URLRequest(url: slopshellAPIURL(baseURL: baseURL, path: "chat/sessions/\(workspace.chatSessionID)/messages"))
            request.httpMethod = "POST"
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try JSONEncoder().encode(SlopshellChatSendRequest(text: text, outputMode: "voice"))
            _ = try await session.data(for: request)
            messages.append(SlopshellRenderedMessage(id: UUID().uuidString, role: "user", text: text, html: ""))
        } catch {
            lastError = error.localizedDescription
        }
    }

    func toggleRecording() async {
        if isRecording {
            audioCapture.stop()
            do {
                try await chatTransport.send(SlopshellAudioCaptureMessage(type: "audio_stop", mimeType: nil, data: nil))
                if isDialogueModeActive {
                    isAwaitingAssistantResponse = true
                    companionRuntimeState = SlopshellDialogueRuntimeState.thinking.rawValue
                }
            } catch {
                lastError = error.localizedDescription
            }
            return
        }
        do {
            audioCapture.stop()
            try audioCapture.start()
            if isDialogueModeActive {
                isAwaitingAssistantResponse = false
                companionRuntimeState = SlopshellDialogueRuntimeState.recording.rawValue
            }
        } catch {
            lastError = error.localizedDescription
        }
    }

    func submitInk(_ strokes: [SlopshellInkStroke]) async {
        guard !strokes.isEmpty else {
            return
        }
        let payload = SlopshellInkCommitMessage(
            type: "ink_stroke",
            artifactKind: "text",
            requestResponse: inkRequestsResponse,
            outputMode: "voice",
            totalStrokes: strokes.count,
            strokes: strokes
        )
        do {
            try await chatTransport.send(payload)
            statusText = inkRequestsResponse ? "Ink sent to Slopshell" : "Ink captured"
        } catch {
            lastError = error.localizedDescription
        }
    }

    private func normalizedBaseURL() -> URL? {
        let trimmed = serverURLString.trimmingCharacters(in: .whitespacesAndNewlines)
        return URL(string: trimmed)
    }

    private func loginIfNeeded(baseURL: URL) async throws {
        var setupRequest = URLRequest(url: slopshellAPIURL(baseURL: baseURL, path: "setup"))
        setupRequest.httpMethod = "GET"
        let (setupData, _) = try await session.data(for: setupRequest)
        let setupObject = try JSONSerialization.jsonObject(with: setupData) as? [String: Any]
        let authenticated = setupObject?["authenticated"] as? Bool ?? false
        let hasPassword = setupObject?["has_password"] as? Bool ?? false
        if authenticated || !hasPassword {
            return
        }
        var request = URLRequest(url: slopshellAPIURL(baseURL: baseURL, path: "login"))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.httpBody = try JSONEncoder().encode(SlopshellLoginRequest(password: password))
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw URLError(.userAuthenticationRequired)
        }
    }

    private func loadWorkspaces(baseURL: URL) async throws -> SlopshellWorkspaceListResponse {
        let (data, _) = try await session.data(from: slopshellAPIURL(baseURL: baseURL, path: "runtime/workspaces"))
        return try JSONDecoder().decode(SlopshellWorkspaceListResponse.self, from: data)
    }

    private func loadHistory(baseURL: URL, workspace: SlopshellWorkspace) async throws {
        let (data, _) = try await session.data(from: slopshellAPIURL(baseURL: baseURL, path: "chat/sessions/\(workspace.chatSessionID)/history"))
        let history = try JSONDecoder().decode(SlopshellChatHistoryResponse.self, from: data)
        messages = history.messages.map {
            SlopshellRenderedMessage(id: "persisted-\($0.id)", role: $0.role, text: $0.content, html: "")
        }
    }

    private func attachRealtime(baseURL: URL, workspace: SlopshellWorkspace) async throws {
        chatTransport.connect(baseURL: baseURL, sessionID: workspace.chatSessionID)
        canvasTransport.connect(baseURL: baseURL, sessionID: workspace.canvasSessionID)
        try await canvasTransport.loadSnapshot(baseURL: baseURL, sessionID: workspace.canvasSessionID)
        async let config = loadCompanionConfig(baseURL: baseURL, workspace: workspace)
        async let state = loadCompanionState(baseURL: baseURL, workspace: workspace)
        applyCompanionConfig(try await config)
        applyCompanionState(try await state)
        isDialogueModeActive = false
        isAwaitingAssistantResponse = false
        restoreCompanionEnabledOnExit = nil
    }

    private func loadCompanionConfig(baseURL: URL, workspace: SlopshellWorkspace) async throws -> SlopshellCompanionConfig {
        let (data, _) = try await session.data(from: slopshellAPIURL(baseURL: baseURL, path: "workspaces/\(workspace.id)/companion/config"))
        return try JSONDecoder().decode(SlopshellCompanionConfig.self, from: data)
    }

    private func loadCompanionState(baseURL: URL, workspace: SlopshellWorkspace) async throws -> SlopshellCompanionStateResponse {
        let (data, _) = try await session.data(from: slopshellAPIURL(baseURL: baseURL, path: "workspaces/\(workspace.id)/companion/state"))
        return try JSONDecoder().decode(SlopshellCompanionStateResponse.self, from: data)
    }

    private func updateCompanionConfig(baseURL: URL, workspace: SlopshellWorkspace, patch: SlopshellCompanionConfigPatch) async throws -> SlopshellCompanionConfig {
        var request = URLRequest(url: slopshellAPIURL(baseURL: baseURL, path: "workspaces/\(workspace.id)/companion/config"))
        request.httpMethod = "PUT"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(patch)
        let (data, _) = try await session.data(for: request)
        return try JSONDecoder().decode(SlopshellCompanionConfig.self, from: data)
    }

    private func updateLivePolicy(baseURL: URL, policy: String) async throws {
        var request = URLRequest(url: slopshellAPIURL(baseURL: baseURL, path: "live-policy"))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(SlopshellLivePolicyRequest(policy: policy))
        let (_, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, (200..<300).contains(http.statusCode) else {
            throw URLError(.badServerResponse)
        }
    }

    private func stopDialogueMode(baseURL: URL, workspace: SlopshellWorkspace, restoreCompanion: Bool) async {
        if isRecording {
            audioCapture.stop()
            do {
                try await chatTransport.send(SlopshellAudioCaptureMessage(type: "audio_stop", mimeType: nil, data: nil))
            } catch {
                lastError = error.localizedDescription
            }
        }
        isDialogueModeActive = false
        isAwaitingAssistantResponse = false
        companionRuntimeState = SlopshellDialogueRuntimeState.idle.rawValue
        if restoreCompanion, let restore = restoreCompanionEnabledOnExit, restore != companionEnabled {
            do {
                let cfg = try await updateCompanionConfig(
                    baseURL: baseURL,
                    workspace: workspace,
                    patch: SlopshellCompanionConfigPatch(companionEnabled: restore, idleSurface: nil)
                )
                applyCompanionConfig(cfg)
            } catch {
                lastError = error.localizedDescription
            }
        }
        restoreCompanionEnabledOnExit = nil
        statusText = "Dialogue mode off"
    }

    private func sendAudioChunk(_ data: Data) async {
        do {
            try await chatTransport.send(SlopshellAudioCaptureMessage(
                type: "audio_pcm",
                mimeType: "audio/L16;rate=16000;channels=1",
                data: data.base64EncodedString()
            ))
        } catch {
            lastError = error.localizedDescription
        }
    }

    private func handleChatEvent(_ event: SlopshellChatEventPayload) {
        switch event.type {
        case "action":
            if event.actionType == "toggle_live_dialogue" {
                Task { await toggleDialogueMode() }
            }
        case "companion_state":
            if event.workspacePath == nil || event.workspacePath == activeWorkspace?.rootPath {
                companionRuntimeState = SlopshellDialogueRuntimeState(raw: event.state ?? "idle").rawValue
            }
        case "render_chat", "assistant_output", "message_persisted":
            let text = event.markdown ?? event.message ?? event.text ?? ""
            if text.isEmpty {
                return
            }
            messages.append(SlopshellRenderedMessage(
                id: event.turnID ?? UUID().uuidString,
                role: event.role ?? "assistant",
                text: text,
                html: event.html ?? ""
            ))
            isAwaitingAssistantResponse = false
            if isDialogueModeActive && isRecording == false {
                companionRuntimeState = SlopshellDialogueRuntimeState.listening.rawValue
            }
        case "stt_result":
            if let text = event.text, text.isEmpty == false {
                composerText = text
                statusText = "Transcription ready"
            }
        case "stt_empty":
            statusText = event.reason ?? "No speech detected"
            isAwaitingAssistantResponse = false
            if isDialogueModeActive {
                companionRuntimeState = SlopshellDialogueRuntimeState.listening.rawValue
            }
        case "stt_error", "error":
            lastError = event.error ?? "Unknown server error"
            isAwaitingAssistantResponse = false
            if isDialogueModeActive {
                companionRuntimeState = SlopshellDialogueRuntimeState.error.rawValue
            }
        default:
            break
        }
    }

    private func applyCompanionConfig(_ config: SlopshellCompanionConfig) {
        companionEnabled = config.companionEnabled
        companionIdleSurface = SlopshellCompanionIdleSurface(raw: config.idleSurface).rawValue
    }

    private func applyCompanionState(_ state: SlopshellCompanionStateResponse) {
        companionEnabled = state.companionEnabled
        companionIdleSurface = SlopshellCompanionIdleSurface(raw: state.idleSurface).rawValue
        companionRuntimeState = SlopshellDialogueRuntimeState(raw: state.state).rawValue
    }
}
