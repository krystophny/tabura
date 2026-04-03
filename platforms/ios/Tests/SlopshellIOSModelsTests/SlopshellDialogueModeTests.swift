import XCTest
@testable import SlopshellIOSModels

final class SlopshellDialogueModeTests: XCTestCase {
    func testBlackSurfaceNeedsDialogueAndBlackIdleSurface() {
        let inactive = SlopshellDialogueModePresentation(
            isActive: false,
            isRecording: false,
            isAwaitingAssistant: false,
            companionEnabled: true,
            idleSurface: "black",
            runtimeState: "idle"
        )
        XCTAssertFalse(inactive.usesBlackScreen)
        XCTAssertFalse(inactive.keepScreenAwake)

        let active = SlopshellDialogueModePresentation(
            isActive: true,
            isRecording: false,
            isAwaitingAssistant: false,
            companionEnabled: true,
            idleSurface: "black",
            runtimeState: "idle"
        )
        XCTAssertTrue(active.usesBlackScreen)
        XCTAssertTrue(active.keepScreenAwake)
        XCTAssertEqual(active.runtimeState, .listening)
    }

    func testRecordingAndAssistantStatesOverrideCompanionIdle() {
        let recording = SlopshellDialogueModePresentation(
            isActive: true,
            isRecording: true,
            isAwaitingAssistant: false,
            companionEnabled: true,
            idleSurface: "black",
            runtimeState: "listening"
        )
        XCTAssertEqual(recording.runtimeState, .recording)
        XCTAssertEqual(recording.primaryLabel, "Recording")
        XCTAssertEqual(recording.tapActionLabel, "Tap to stop recording")

        let thinking = SlopshellDialogueModePresentation(
            isActive: true,
            isRecording: false,
            isAwaitingAssistant: true,
            companionEnabled: true,
            idleSurface: "black",
            runtimeState: "listening"
        )
        XCTAssertEqual(thinking.runtimeState, .thinking)
        XCTAssertEqual(thinking.primaryLabel, "Working")
        XCTAssertEqual(thinking.tapActionLabel, "Waiting for Slopshell")
    }

    func testCompanionRuntimeStateFallsBackToListeningDuringDialogue() {
        let talking = SlopshellDialogueModePresentation(
            isActive: true,
            isRecording: false,
            isAwaitingAssistant: false,
            companionEnabled: true,
            idleSurface: "robot",
            runtimeState: "talking"
        )
        XCTAssertEqual(talking.runtimeState, .talking)
        XCTAssertEqual(talking.primaryLabel, "Reply ready")

        let defaulted = SlopshellDialogueModePresentation(
            isActive: true,
            isRecording: false,
            isAwaitingAssistant: false,
            companionEnabled: false,
            idleSurface: "robot",
            runtimeState: "idle"
        )
        XCTAssertEqual(defaulted.runtimeState, .listening)
        XCTAssertEqual(defaulted.secondaryLabel, "Tap anywhere on the dialogue surface to record.")
    }
}
