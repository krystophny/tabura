package ios

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSloppadIOSProjectIncludesExpectedFiles(t *testing.T) {
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	projectFile := filepath.Join(projectRoot, "SloppadIOS.xcodeproj", "project.pbxproj")
	data, err := os.ReadFile(projectFile)
	if err != nil {
		t.Fatalf("ReadFile(project): %v", err)
	}
	project := string(data)
	expected := []string{
		"Package.swift",
		filepath.Join("Sources", "SloppadFlowContract", "FlowFixture.swift"),
		filepath.Join("Sources", "SloppadFlowContract", "FlowRunner.swift"),
		filepath.Join("Tests", "SloppadFlowContractTests", "SloppadFlowContractTests.swift"),
		filepath.Join("Tests", "SloppadFlowContractTests", "Resources", "flow-fixtures.json"),
		filepath.Join("Tests", "SloppadIOSModelsTests", "SloppadDialogueModeTests.swift"),
		"SloppadIOSApp.swift",
		"ContentView.swift",
		"SloppadAppModel.swift",
		"SloppadModels.swift",
		"SloppadServerDiscovery.swift",
		"SloppadChatTransport.swift",
		"SloppadCanvasTransport.swift",
		"SloppadAudioCapture.swift",
		"SloppadInkCaptureView.swift",
		"SloppadCanvasWebView.swift",
		"Info.plist",
	}
	for _, name := range expected {
		path := filepath.Join(projectRoot, name)
		if strings.HasPrefix(name, "Sources") || strings.HasPrefix(name, "Tests") || name == "Package.swift" {
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("missing expected file %q: %v", path, err)
			}
			continue
		}
		path = filepath.Join(projectRoot, "SloppadIOS", name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing expected file %q: %v", path, err)
		}
		if !strings.Contains(project, name) {
			t.Fatalf("project.pbxproj missing reference to %q", name)
		}
	}
}

func TestSloppadIOSInfoPlistDeclaresMobileCapabilities(t *testing.T) {
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	infoPath := filepath.Join(projectRoot, "SloppadIOS", "Info.plist")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		t.Fatalf("ReadFile(Info.plist): %v", err)
	}
	info := string(data)
	requiredSnippets := []string{
		"<key>UIBackgroundModes</key>",
		"<string>audio</string>",
		"<key>NSBonjourServices</key>",
		"<string>_sloppad._tcp</string>",
		"<key>NSMicrophoneUsageDescription</key>",
		"<key>NSLocalNetworkUsageDescription</key>",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(info, snippet) {
			t.Fatalf("Info.plist missing %q", snippet)
		}
	}
}

func TestSloppadIOSSourcesCoverBlackScreenDialogueMode(t *testing.T) {
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	checks := []struct {
		relative string
		snippets []string
	}{
		{
			relative: filepath.Join("SloppadIOS", "ContentView.swift"),
			snippets: []string{"blackScreenDialoguePanel", "Exit Dialogue", "isIdleTimerDisabled"},
		},
		{
			relative: filepath.Join("SloppadIOS", "SloppadAppModel.swift"),
			snippets: []string{"toggleDialogueMode()", "companion/config", "live-policy", "toggle_live_dialogue"},
		},
		{
			relative: filepath.Join("SloppadIOS", "SloppadModels.swift"),
			snippets: []string{"SloppadDialogueModePresentation", "usesBlackScreen", "keepScreenAwake", "Tap to stop recording"},
		},
	}
	for _, check := range checks {
		path := filepath.Join(projectRoot, check.relative)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		content := string(data)
		for _, snippet := range check.snippets {
			if !strings.Contains(content, snippet) {
				t.Fatalf("%s missing %q", check.relative, snippet)
			}
		}
	}
}
