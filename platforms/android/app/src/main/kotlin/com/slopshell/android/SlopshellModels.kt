package com.slopshell.android

import org.json.JSONArray
import org.json.JSONObject
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import java.util.Base64

data class SlopshellWorkspaceListResponse(
    val activeWorkspaceId: String,
    val workspaces: List<SlopshellWorkspace>,
)

data class SlopshellWorkspace(
    val id: String,
    val name: String,
    val rootPath: String,
    val chatSessionId: String,
    val canvasSessionId: String,
)

data class SlopshellRenderedMessage(
    val id: String,
    val role: String,
    val text: String,
    val html: String = "",
)

data class SlopshellCanvasArtifact(
    val kind: String,
    val title: String,
    val html: String,
    val text: String,
)

data class SlopshellChatEventPayload(
    val type: String,
    val turnId: String = "",
    val role: String = "",
    val message: String = "",
    val markdown: String = "",
    val html: String = "",
    val error: String = "",
    val text: String = "",
    val reason: String = "",
    val state: String = "",
    val workspacePath: String = "",
    val actionType: String = "",
)

data class SlopshellCompanionConfig(
    val companionEnabled: Boolean,
    val idleSurface: String,
)

data class SlopshellCompanionState(
    val companionEnabled: Boolean,
    val idleSurface: String,
    val state: String,
    val reason: String,
)

enum class SlopshellCompanionIdleSurface(val wireValue: String) {
    ROBOT("robot"),
    BLACK("black");

    companion object {
        fun normalize(raw: String): SlopshellCompanionIdleSurface {
            return if (raw.trim().lowercase() == BLACK.wireValue) BLACK else ROBOT
        }
    }
}

enum class SlopshellDialogueRuntimeState {
    IDLE,
    LISTENING,
    RECORDING,
    THINKING,
    TALKING,
    ERROR;

    companion object {
        fun normalize(raw: String): SlopshellDialogueRuntimeState {
            return when (raw.trim().lowercase()) {
                "listening" -> LISTENING
                "recording" -> RECORDING
                "thinking" -> THINKING
                "talking" -> TALKING
                "error" -> ERROR
                else -> IDLE
            }
        }
    }
}

data class SlopshellDialogueModePresentation(
    val isActive: Boolean,
    val isRecording: Boolean,
    val isAwaitingAssistant: Boolean,
    val companionEnabled: Boolean,
    val idleSurface: String,
    val runtimeStateValue: String,
) {
    val effectiveIdleSurface = SlopshellCompanionIdleSurface.normalize(idleSurface)
    val usesBlackScreen = isActive && effectiveIdleSurface == SlopshellCompanionIdleSurface.BLACK
    val keepScreenAwake = usesBlackScreen
    val runtimeState = when {
        !isActive -> SlopshellDialogueRuntimeState.IDLE
        isRecording -> SlopshellDialogueRuntimeState.RECORDING
        isAwaitingAssistant -> SlopshellDialogueRuntimeState.THINKING
        else -> SlopshellDialogueRuntimeState.normalize(runtimeStateValue).let {
            if (it == SlopshellDialogueRuntimeState.IDLE) SlopshellDialogueRuntimeState.LISTENING else it
        }
    }
    val primaryLabel = when (runtimeState) {
        SlopshellDialogueRuntimeState.IDLE -> if (companionEnabled) "Ready" else "Disconnected"
        SlopshellDialogueRuntimeState.LISTENING -> "Listening"
        SlopshellDialogueRuntimeState.RECORDING -> "Recording"
        SlopshellDialogueRuntimeState.THINKING -> "Working"
        SlopshellDialogueRuntimeState.TALKING -> "Reply ready"
        SlopshellDialogueRuntimeState.ERROR -> "Attention needed"
    }
    val secondaryLabel = when (runtimeState) {
        SlopshellDialogueRuntimeState.IDLE -> "Start dialogue to hand the screen to voice."
        SlopshellDialogueRuntimeState.LISTENING -> "Tap anywhere on the dialogue surface to record."
        SlopshellDialogueRuntimeState.RECORDING -> "Android keeps the foreground mic service active while recording."
        SlopshellDialogueRuntimeState.THINKING -> "Slopshell is processing your last recording."
        SlopshellDialogueRuntimeState.TALKING -> "Tap to interrupt and start a new recording."
        SlopshellDialogueRuntimeState.ERROR -> "Check the connection banner for the latest error."
    }
    val tapActionLabel = when (runtimeState) {
        SlopshellDialogueRuntimeState.IDLE -> "Start dialogue"
        SlopshellDialogueRuntimeState.LISTENING -> "Tap to record"
        SlopshellDialogueRuntimeState.RECORDING -> "Tap to stop recording"
        SlopshellDialogueRuntimeState.THINKING -> "Waiting for Slopshell"
        SlopshellDialogueRuntimeState.TALKING -> "Tap to record"
        SlopshellDialogueRuntimeState.ERROR -> "Tap to retry"
    }
}

data class SlopshellInkPoint(
    val x: Float,
    val y: Float,
    val pressure: Float,
    val tiltX: Float,
    val tiltY: Float,
    val roll: Float,
    val timestampMs: Long,
)

data class SlopshellInkStroke(
    val pointerType: String,
    val width: Float,
    val points: List<SlopshellInkPoint>,
)

data class SlopshellDiscoveredServer(
    val id: String,
    val name: String,
    val host: String,
    val port: Int,
) {
    val baseUrlString: String
        get() = "http://$host:$port"
}

fun slopshellWsUrl(baseUrl: String, path: String): String {
    val base = URI(baseUrl.trim())
    val scheme = if (base.scheme.equals("https", ignoreCase = true)) "wss" else "ws"
    val authority = base.rawAuthority ?: error("base URL is missing an authority: $baseUrl")
    val encodedPath = path
        .split("/")
        .joinToString("/") { segment -> URLEncoder.encode(segment, StandardCharsets.UTF_8).replace("+", "%20") }
    return "$scheme://$authority/ws/$encodedPath"
}

fun slopshellApiUrl(baseUrl: String, path: String): String {
    return "${baseUrl.trim().trimEnd('/')}/api/$path"
}

fun parseWorkspaceListResponse(body: String): SlopshellWorkspaceListResponse {
    val json = JSONObject(body)
    val workspaces = buildList {
        val items = json.optJSONArray("workspaces") ?: JSONArray()
        for (index in 0 until items.length()) {
            val item = items.optJSONObject(index) ?: continue
            add(
                SlopshellWorkspace(
                    id = item.optString("id"),
                    name = item.optString("name"),
                    rootPath = item.optString("root_path"),
                    chatSessionId = item.optString("chat_session_id"),
                    canvasSessionId = item.optString("canvas_session_id"),
                )
            )
        }
    }
    return SlopshellWorkspaceListResponse(
        activeWorkspaceId = json.optString("active_workspace_id"),
        workspaces = workspaces,
    )
}

fun parseChatHistory(body: String): List<SlopshellRenderedMessage> {
    val json = JSONObject(body)
    val messages = json.optJSONArray("messages") ?: JSONArray()
    return buildList {
        for (index in 0 until messages.length()) {
            val item = messages.optJSONObject(index) ?: continue
            val markdown = item.optString("content_markdown")
            val plain = item.optString("content_plain")
            add(
                SlopshellRenderedMessage(
                    id = "persisted-${item.optLong("id")}",
                    role = item.optString("role"),
                    text = markdown.takeIf { it.isNotBlank() } ?: plain,
                )
            )
        }
    }
}

fun parseCanvasSnapshot(body: String): SlopshellCanvasArtifact? {
    val event = JSONObject(body).optJSONObject("event") ?: return null
    return parseCanvasArtifact(event)
}

fun parseCanvasArtifact(payload: JSONObject): SlopshellCanvasArtifact {
    val text = payload.optString("text").ifBlank { payload.optString("markdown_or_text") }
    return SlopshellCanvasArtifact(
        kind = payload.optString("kind"),
        title = payload.optString("title"),
        html = payload.optString("html").ifBlank { wrapCanvasText(text) },
        text = text,
    )
}

fun parseChatEvent(raw: String): SlopshellChatEventPayload {
    val json = JSONObject(raw)
    val action = json.optJSONObject("action")
    return SlopshellChatEventPayload(
        type = json.optString("type"),
        turnId = json.optString("turn_id"),
        role = json.optString("role"),
        message = json.optString("message"),
        markdown = json.optString("markdown"),
        html = json.optString("html"),
        error = json.optString("error"),
        text = json.optString("text"),
        reason = json.optString("reason"),
        state = json.optString("state"),
        workspacePath = json.optString("workspace_path"),
        actionType = action?.optString("type").orEmpty(),
    )
}

fun parseCompanionConfig(body: String): SlopshellCompanionConfig {
    val json = JSONObject(body)
    return SlopshellCompanionConfig(
        companionEnabled = json.optBoolean("companion_enabled"),
        idleSurface = json.optString("idle_surface", SlopshellCompanionIdleSurface.ROBOT.wireValue),
    )
}

fun parseCompanionState(body: String): SlopshellCompanionState {
    val json = JSONObject(body)
    return SlopshellCompanionState(
        companionEnabled = json.optBoolean("companion_enabled"),
        idleSurface = json.optString("idle_surface", SlopshellCompanionIdleSurface.ROBOT.wireValue),
        state = json.optString("state"),
        reason = json.optString("reason"),
    )
}

fun loginRequest(password: String): String {
    return JSONObject().put("password", password).toString()
}

fun composerRequest(text: String): String {
    return JSONObject()
        .put("text", text)
        .put("output_mode", "voice")
        .toString()
}

fun companionConfigPatch(companionEnabled: Boolean? = null, idleSurface: String? = null): String {
    val json = JSONObject()
    if (companionEnabled != null) {
        json.put("companion_enabled", companionEnabled)
    }
    if (!idleSurface.isNullOrBlank()) {
        json.put("idle_surface", idleSurface)
    }
    return json.toString()
}

fun livePolicyRequest(policy: String): String {
    return JSONObject().put("policy", policy).toString()
}

fun audioPcmMessage(data: ByteArray): String {
    return JSONObject()
        .put("type", "audio_pcm")
        .put("mime_type", "audio/L16;rate=16000;channels=1")
        .put("data", Base64.getEncoder().withoutPadding().encodeToString(data))
        .toString()
}

fun audioStopMessage(): String {
    return JSONObject().put("type", "audio_stop").toString()
}

fun inkCommitMessage(strokes: List<SlopshellInkStroke>, requestResponse: Boolean): String {
    val items = JSONArray()
    for (stroke in strokes) {
        val points = JSONArray()
        for (point in stroke.points) {
            points.put(
                JSONObject()
                    .put("x", point.x)
                    .put("y", point.y)
                    .put("pressure", point.pressure)
                    .put("tilt_x", point.tiltX)
                    .put("tilt_y", point.tiltY)
                    .put("roll", point.roll)
                    .put("timestamp_ms", point.timestampMs)
            )
        }
        items.put(
            JSONObject()
                .put("pointer_type", stroke.pointerType)
                .put("width", stroke.width)
                .put("points", points)
        )
    }
    return JSONObject()
        .put("type", "ink_stroke")
        .put("artifact_kind", "text")
        .put("request_response", requestResponse)
        .put("output_mode", "voice")
        .put("total_strokes", strokes.size)
        .put("strokes", items)
        .toString()
}

private fun wrapCanvasText(text: String): String {
    val escaped = text
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
    return "<pre style=\"white-space: pre-wrap; margin: 24px; font: sans-serif;\">$escaped</pre>"
}
