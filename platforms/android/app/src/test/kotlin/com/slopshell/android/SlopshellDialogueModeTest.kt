package com.slopshell.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class SlopshellDialogueModeTest {
    @Test
    fun blackSurfaceNeedsDialogueAndBlackIdleSurface() {
        val inactive = SlopshellDialogueModePresentation(
            isActive = false,
            isRecording = false,
            isAwaitingAssistant = false,
            companionEnabled = true,
            idleSurface = "black",
            runtimeStateValue = "idle",
        )
        assertFalse(inactive.usesBlackScreen)
        assertFalse(inactive.keepScreenAwake)

        val active = SlopshellDialogueModePresentation(
            isActive = true,
            isRecording = false,
            isAwaitingAssistant = false,
            companionEnabled = true,
            idleSurface = "black",
            runtimeStateValue = "idle",
        )
        assertTrue(active.usesBlackScreen)
        assertTrue(active.keepScreenAwake)
        assertEquals(SlopshellDialogueRuntimeState.LISTENING, active.runtimeState)
    }

    @Test
    fun recordingAndAssistantStatesOverrideIdleCompanionState() {
        val recording = SlopshellDialogueModePresentation(
            isActive = true,
            isRecording = true,
            isAwaitingAssistant = false,
            companionEnabled = true,
            idleSurface = "black",
            runtimeStateValue = "listening",
        )
        assertEquals(SlopshellDialogueRuntimeState.RECORDING, recording.runtimeState)
        assertEquals("Tap to stop recording", recording.tapActionLabel)

        val thinking = SlopshellDialogueModePresentation(
            isActive = true,
            isRecording = false,
            isAwaitingAssistant = true,
            companionEnabled = true,
            idleSurface = "black",
            runtimeStateValue = "listening",
        )
        assertEquals(SlopshellDialogueRuntimeState.THINKING, thinking.runtimeState)
        assertEquals("Working", thinking.primaryLabel)
    }

    @Test
    fun explicitServerRuntimeStateIsPreserved() {
        val talking = SlopshellDialogueModePresentation(
            isActive = true,
            isRecording = false,
            isAwaitingAssistant = false,
            companionEnabled = true,
            idleSurface = "robot",
            runtimeStateValue = "talking",
        )
        assertEquals(SlopshellDialogueRuntimeState.TALKING, talking.runtimeState)
        assertEquals("Reply ready", talking.primaryLabel)
        assertEquals("Tap to record", talking.tapActionLabel)
    }
}
