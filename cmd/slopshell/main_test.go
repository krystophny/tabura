package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	updater "github.com/sloppy-org/slopshell/internal/update"
)

func TestParseServerConfigDefaultsToLoopbackWebHost(t *testing.T) {
	cfg, status := parseServerConfig([]string{})
	if status != 0 {
		t.Fatalf("parseServerConfig() status = %d, want 0", status)
	}
	if cfg.webHost != "127.0.0.1" {
		t.Fatalf("default webHost = %q, want 127.0.0.1", cfg.webHost)
	}
}

func TestParseServerConfigRejectsRemovedMCPHostFlag(t *testing.T) {
	_, status := parseServerConfig([]string{"--mcp-host", "0.0.0.0"})
	if status != 2 {
		t.Fatalf("parseServerConfig(removed --mcp-host) status = %d, want 2", status)
	}
}

func TestParseServerConfigAcceptsMCPSocketPath(t *testing.T) {
	cfg, status := parseServerConfig([]string{"--mcp-socket", "/run/user/1000/sloppy/mcp.sock"})
	if status != 0 {
		t.Fatalf("parseServerConfig(mcp-socket) status = %d, want 0", status)
	}
	if cfg.mcpSocket != "/run/user/1000/sloppy/mcp.sock" {
		t.Fatalf("mcpSocket = %q, want /run/user/1000/sloppy/mcp.sock", cfg.mcpSocket)
	}
}

func TestParseServerConfigRejectsIncompleteTLSConfig(t *testing.T) {
	_, status := parseServerConfig([]string{"--web-cert-file", "/tmp/cert.pem"})
	if status != 2 {
		t.Fatalf("parseServerConfig(incomplete tls) status = %d, want 2", status)
	}
}

func TestParseServerConfigAcceptsTLSConfigPair(t *testing.T) {
	cfg, status := parseServerConfig([]string{"--web-cert-file", "/tmp/cert.pem", "--web-key-file", "/tmp/key.pem"})
	if status != 0 {
		t.Fatalf("parseServerConfig(tls pair) status = %d, want 0", status)
	}
	if cfg.webCertFile != "/tmp/cert.pem" {
		t.Fatalf("webCertFile = %q, want /tmp/cert.pem", cfg.webCertFile)
	}
	if cfg.webKeyFile != "/tmp/key.pem" {
		t.Fatalf("webKeyFile = %q, want /tmp/key.pem", cfg.webKeyFile)
	}
}

func TestFormatVersionLinePrefixesVersion(t *testing.T) {
	got := formatVersionLine("0.1.4", "abc1234", "linux", "amd64")
	want := "slopshell v0.1.4 (abc1234) linux/amd64"
	if got != want {
		t.Fatalf("formatVersionLine() = %q, want %q", got, want)
	}
}

func TestFormatVersionLineKeepsPrefixedVersionAndHandlesMissingCommit(t *testing.T) {
	got := formatVersionLine("v1.2.3", "", "windows", "arm64")
	want := "slopshell v1.2.3 (unknown) windows/arm64"
	if got != want {
		t.Fatalf("formatVersionLine() = %q, want %q", got, want)
	}
}

func TestRunDispatchesUpdateCommand(t *testing.T) {
	prev := runUpdate
	t.Cleanup(func() { runUpdate = prev })
	called := false
	runUpdate = func(opts updater.Options) (updater.Result, error) {
		called = true
		if opts.CurrentVersion != defaultBinaryVersion {
			return updater.Result{}, fmt.Errorf("unexpected version %q", opts.CurrentVersion)
		}
		return updater.Result{CurrentVersion: "v0.1.4", LatestVersion: "v0.1.4", Updated: false}, nil
	}

	status := run([]string{"update"})
	if status != 0 {
		t.Fatalf("run(update) status = %d, want 0", status)
	}
	if !called {
		t.Fatalf("expected updater to be called")
	}
}

func TestRunUnknownCommandReturnsUsageStatus(t *testing.T) {
	status := run([]string{"not-a-command"})
	if status != 2 {
		t.Fatalf("run(unknown) status = %d, want 2", status)
	}
}

func TestCmdSchemaOutputsProtocolJSON(t *testing.T) {
	out := captureStdout(t, func() {
		status := cmdSchema()
		if status != 0 {
			t.Fatalf("cmdSchema() status = %d, want 0", status)
		}
	})
	if !strings.Contains(out, `"title": "SlopshellCanvasEvent"`) {
		t.Fatalf("cmdSchema output missing title: %q", out)
	}
	if !strings.Contains(out, `"const": "text_artifact"`) {
		t.Fatalf("cmdSchema output missing text_artifact schema: %q", out)
	}
}


func TestCmdUpdateFailureReturnsStatusOne(t *testing.T) {
	prev := runUpdate
	t.Cleanup(func() { runUpdate = prev })
	runUpdate = func(opts updater.Options) (updater.Result, error) {
		return updater.Result{}, errors.New("update failed")
	}

	status := cmdUpdate(nil)
	if status != 1 {
		t.Fatalf("cmdUpdate() status = %d, want 1", status)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("capture stdout copy: %v", err)
	}
	return buf.String()
}
