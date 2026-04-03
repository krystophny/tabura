package com.sloppad.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class SloppadDialogueModeTest {
    @Test
    fun blackSurfaceNeedsDialogueAndBlackIdleSurface() {
        val inactive = SloppadDialogueModePresentation(
            isActive = false,
            isRecording = false,
            isAwaitingAssistant = false,
            companionEnabled = true,
            idleSurface = "black",
            runtimeStateValue = "idle",
        )
        assertFalse(inactive.usesBlackScreen)
        assertFalse(inactive.keepScreenAwake)

        val active = SloppadDialogueModePresentation(
            isActive = true,
            isRecording = false,
            isAwaitingAssistant = false,
            companionEnabled = true,
            idleSurface = "black",
            runtimeStateValue = "idle",
        )
        assertTrue(active.usesBlackScreen)
        assertTrue(active.keepScreenAwake)
        assertEquals(SloppadDialogueRuntimeState.LISTENING, active.runtimeState)
    }

    @Test
    fun recordingAndAssistantStatesOverrideIdleCompanionState() {
        val recording = SloppadDialogueModePresentation(
            isActive = true,
            isRecording = true,
            isAwaitingAssistant = false,
            companionEnabled = true,
            idleSurface = "black",
            runtimeStateValue = "listening",
        )
        assertEquals(SloppadDialogueRuntimeState.RECORDING, recording.runtimeState)
        assertEquals("Tap to stop recording", recording.tapActionLabel)

        val thinking = SloppadDialogueModePresentation(
            isActive = true,
            isRecording = false,
            isAwaitingAssistant = true,
            companionEnabled = true,
            idleSurface = "black",
            runtimeStateValue = "listening",
        )
        assertEquals(SloppadDialogueRuntimeState.THINKING, thinking.runtimeState)
        assertEquals("Working", thinking.primaryLabel)
    }

    @Test
    fun explicitServerRuntimeStateIsPreserved() {
        val talking = SloppadDialogueModePresentation(
            isActive = true,
            isRecording = false,
            isAwaitingAssistant = false,
            companionEnabled = true,
            idleSurface = "robot",
            runtimeStateValue = "talking",
        )
        assertEquals(SloppadDialogueRuntimeState.TALKING, talking.runtimeState)
        assertEquals("Reply ready", talking.primaryLabel)
        assertEquals("Tap to record", talking.tapActionLabel)
    }
}
