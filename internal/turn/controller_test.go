package turn

import (
	"testing"
	"time"
)

func TestConsumeSegmentContinuesShortFragment(t *testing.T) {
	controller := NewController(Callbacks{})
	decision := controller.ConsumeSegment(Segment{
		Text:       "I think",
		DurationMS: 320,
	})
	if decision.Action != ActionContinueListen {
		t.Fatalf("action = %q, want %q", decision.Action, ActionContinueListen)
	}
	if decision.Reason == "" {
		t.Fatal("reason is empty")
	}
}

func TestConsumeSegmentFinalizesSemanticCompletion(t *testing.T) {
	controller := NewController(Callbacks{})
	controller.ConsumeSegment(Segment{
		Text:       "I think",
		DurationMS: 280,
	})
	decision := controller.ConsumeSegment(Segment{
		Text:       "that's enough.",
		DurationMS: 1250,
	})
	if decision.Action != ActionFinalizeUserTurn {
		t.Fatalf("action = %q, want %q", decision.Action, ActionFinalizeUserTurn)
	}
	if decision.Text != "I think that's enough." {
		t.Fatalf("text = %q", decision.Text)
	}
}

func TestConsumeSegmentBackchannelsInterruptedAcknowledgement(t *testing.T) {
	controller := NewController(Callbacks{})
	decision := controller.ConsumeSegment(Segment{
		Text:                 "okay",
		DurationMS:           180,
		InterruptedAssistant: true,
	})
	if decision.Action != ActionBackchannel {
		t.Fatalf("action = %q, want %q", decision.Action, ActionBackchannel)
	}
}

func TestHandleSpeechProbabilityYieldsDuringPlayback(t *testing.T) {
	controller := NewController(Callbacks{})
	controller.UpdatePlayback(true, 480)
	for i := 0; i < 2; i++ {
		if signal := controller.HandleSpeechProbability(0.9, true); signal != nil {
			t.Fatalf("signal on frame %d = %#v, want nil", i, signal)
		}
	}
	signal := controller.HandleSpeechProbability(0.9, true)
	if signal == nil {
		t.Fatal("expected yield signal")
	}
	if signal.Action != ActionYield {
		t.Fatalf("action = %q, want %q", signal.Action, ActionYield)
	}
	if signal.RollbackAudioMS <= 0 {
		t.Fatalf("rollback_audio_ms = %d, want > 0", signal.RollbackAudioMS)
	}
}

func TestFlushFinalizesPendingText(t *testing.T) {
	controller := NewController(Callbacks{})
	controller.ConsumeSegment(Segment{
		Text:       "when we get to",
		DurationMS: 420,
	})
	signal := controller.Flush("test_timeout")
	if signal == nil {
		t.Fatal("expected flush signal")
	}
	if signal.Action != ActionFinalizeUserTurn {
		t.Fatalf("action = %q, want %q", signal.Action, ActionFinalizeUserTurn)
	}
	if signal.Text != "when we get to" {
		t.Fatalf("text = %q", signal.Text)
	}
}

func TestContinuationTimeoutEmitsFinalize(t *testing.T) {
	done := make(chan Signal, 1)
	controller := NewController(Callbacks{
		OnAction: func(signal Signal) {
			if signal.Action == ActionFinalizeUserTurn && signal.Reason == "continuation_timeout" {
				done <- signal
			}
		},
	})
	defer controller.Close()

	controller.ConsumeSegment(Segment{
		Text:       "I think",
		DurationMS: 250,
	})

	select {
	case signal := <-done:
		if signal.Text != "I think" {
			t.Fatalf("text = %q", signal.Text)
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("timed out waiting for continuation finalize")
	}
}

func TestSetProfileAffectsContinuationBehavior(t *testing.T) {
	patient := NewController(Callbacks{}, WithProfile(ProfilePatient))
	defer patient.Close()
	patientDecision := patient.ConsumeSegment(Segment{
		Text:       "I think",
		DurationMS: 320,
	})
	if patientDecision.Action != ActionContinueListen {
		t.Fatalf("patient action = %q, want %q", patientDecision.Action, ActionContinueListen)
	}
	if patientDecision.WaitMS < 800 {
		t.Fatalf("patient wait_ms = %d, want >= 800", patientDecision.WaitMS)
	}

	assertive := NewController(Callbacks{}, WithProfile(ProfileAssertive))
	defer assertive.Close()
	assertiveDecision := assertive.ConsumeSegment(Segment{
		Text:       "I think",
		DurationMS: 320,
	})
	if assertiveDecision.Action != ActionContinueListen {
		t.Fatalf("assertive action = %q, want %q", assertiveDecision.Action, ActionContinueListen)
	}
	if assertiveDecision.WaitMS >= patientDecision.WaitMS {
		t.Fatalf("assertive wait_ms = %d, want less than patient wait_ms %d", assertiveDecision.WaitMS, patientDecision.WaitMS)
	}
}

func TestMetricsSnapshotIncludesProfileAndLogging(t *testing.T) {
	controller := NewController(Callbacks{}, WithProfile(ProfileAssertive), WithEvalLogging(false))
	defer controller.Close()

	controller.UpdatePlayback(true, 220)
	controller.ConsumeSegment(Segment{
		Text:       "I think",
		DurationMS: 300,
	})
	metrics := controller.SnapshotMetrics()
	if metrics.Profile != ProfileAssertive {
		t.Fatalf("profile = %q, want %q", metrics.Profile, ProfileAssertive)
	}
	if metrics.EvalLoggingEnabled {
		t.Fatal("eval logging enabled = true, want false")
	}
	if !metrics.PlaybackActive {
		t.Fatal("playback active = false, want true")
	}
	if metrics.PendingTextChars == 0 {
		t.Fatal("pending text chars = 0, want > 0")
	}
	if metrics.Actions[string(ActionContinueListen)] == 0 {
		t.Fatalf("continue_listening count = %d, want > 0", metrics.Actions[string(ActionContinueListen)])
	}
}
