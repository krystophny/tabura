package ios

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlopshellIOSProjectIncludesExpectedFiles(t *testing.T) {
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	projectFile := filepath.Join(projectRoot, "SlopshellIOS.xcodeproj", "project.pbxproj")
	data, err := os.ReadFile(projectFile)
	if err != nil {
		t.Fatalf("ReadFile(project): %v", err)
	}
	project := string(data)
	expected := []string{
		"Package.swift",
		filepath.Join("Sources", "SlopshellFlowContract", "FlowFixture.swift"),
		filepath.Join("Sources", "SlopshellFlowContract", "FlowRunner.swift"),
		filepath.Join("Tests", "SlopshellFlowContractTests", "SlopshellFlowContractTests.swift"),
		filepath.Join("Tests", "SlopshellFlowContractTests", "Resources", "flow-fixtures.json"),
		filepath.Join("Tests", "SlopshellIOSModelsTests", "SlopshellDialogueModeTests.swift"),
		"SlopshellIOSApp.swift",
		"ContentView.swift",
		"SlopshellAppModel.swift",
		"SlopshellModels.swift",
		"SlopshellServerDiscovery.swift",
		"SlopshellChatTransport.swift",
		"SlopshellCanvasTransport.swift",
		"SlopshellAudioCapture.swift",
		"SlopshellInkCaptureView.swift",
		"SlopshellCanvasWebView.swift",
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
		path = filepath.Join(projectRoot, "SlopshellIOS", name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing expected file %q: %v", path, err)
		}
		if !strings.Contains(project, name) {
			t.Fatalf("project.pbxproj missing reference to %q", name)
		}
	}
}

func TestSlopshellIOSInfoPlistDeclaresMobileCapabilities(t *testing.T) {
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	infoPath := filepath.Join(projectRoot, "SlopshellIOS", "Info.plist")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		t.Fatalf("ReadFile(Info.plist): %v", err)
	}
	info := string(data)
	requiredSnippets := []string{
		"<key>UIBackgroundModes</key>",
		"<string>audio</string>",
		"<key>NSBonjourServices</key>",
		"<string>_slopshell._tcp</string>",
		"<key>NSMicrophoneUsageDescription</key>",
		"<key>NSLocalNetworkUsageDescription</key>",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(info, snippet) {
			t.Fatalf("Info.plist missing %q", snippet)
		}
	}
}

func TestSlopshellIOSSourcesCoverBlackScreenDialogueMode(t *testing.T) {
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	checks := []struct {
		relative string
		snippets []string
	}{
		{
			relative: filepath.Join("SlopshellIOS", "ContentView.swift"),
			snippets: []string{"blackScreenDialoguePanel", "Exit Dialogue", "isIdleTimerDisabled"},
		},
		{
			relative: filepath.Join("SlopshellIOS", "SlopshellAppModel.swift"),
			snippets: []string{"toggleDialogueMode()", "companion/config", "live-policy", "toggle_live_dialogue"},
		},
		{
			relative: filepath.Join("SlopshellIOS", "SlopshellModels.swift"),
			snippets: []string{"SlopshellDialogueModePresentation", "usesBlackScreen", "keepScreenAwake", "Tap to stop recording"},
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
