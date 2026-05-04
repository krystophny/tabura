package web

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStartLocalServeReusesPrimaryStore(t *testing.T) {
	t.Setenv("SLOPSHELL_BACKGROUND_SYNC", "off")
	t.Setenv("SLOPSHELL_BRAIN_GTD_SYNC", "off")
	t.Setenv("SLOPSHELL_HELPY_SOCKET", "off")

	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	socketPath := filepath.Join(t.TempDir(), "control.sock")

	app, err := New(dataDir, workspaceDir, socketPath, "", "", "", "", false)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	if err := app.startLocalServe(); err != nil {
		t.Fatalf("start local serve: %v", err)
	}
	if app.tunnels.localApp == nil {
		t.Fatal("expected local serve app")
	}
	if app.tunnels.localApp.Store == nil {
		t.Fatal("expected local serve app store")
	}
	if app.tunnels.localApp.Store != app.store {
		t.Fatal("expected local serve to reuse primary store")
	}
}
