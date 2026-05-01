package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestHandleGtdCaptureSendsTitleAndDefaults(t *testing.T) {
	srv, token, captures := newGtdMutationHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/items/capture" {
			t.Fatalf("request = %s %s, want POST /api/items/capture", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"item":{"id":50,"title":"Reply to grant","kind":"action","state":"inbox","sphere":"work"}}`))
	})

	out, stderr, code := runGtdMutationCommand(t, srv, token, "capture", "Reply to grant")
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr)
	}
	if got := (*captures)[0].body["title"]; got != "Reply to grant" {
		t.Fatalf("title body = %#v, want Reply to grant", (*captures)[0].body)
	}
	if _, set := (*captures)[0].body["kind"]; set {
		t.Fatalf("kind should be omitted by default; body = %#v", (*captures)[0].body)
	}
	if !strings.Contains(out, "captured #50") || !strings.Contains(out, "kind=action") || !strings.Contains(out, "state=inbox") {
		t.Fatalf("output = %q", out)
	}
}

func TestHandleGtdCaptureProjectKindAndProjectLink(t *testing.T) {
	srv, token, captures := newGtdMutationHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"item":{"id":51,"title":"Schedule kickoff","kind":"action","state":"inbox","sphere":"work"},"project_item":{"project_item_id":12,"role":"next_action"}}`))
	})

	out, stderr, code := runGtdMutationCommand(t, srv, token, "capture", "Schedule kickoff", "--project-item-id", "12")
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr)
	}
	body := (*captures)[0].body
	if body["project_item_id"] != float64(12) {
		t.Fatalf("project_item_id body = %#v", body)
	}
	if !strings.Contains(out, "project_item=12") || !strings.Contains(out, "role=next_action") {
		t.Fatalf("output = %q", out)
	}
}

func TestHandleGtdCaptureKindProjectFlag(t *testing.T) {
	srv, token, captures := newGtdMutationHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"item":{"id":52,"title":"Ship dialog","kind":"project","state":"inbox","sphere":"work"}}`))
	})

	_, stderr, code := runGtdMutationCommand(t, srv, token, "capture", "Ship dialog", "--kind", "project")
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr)
	}
	if (*captures)[0].body["kind"] != "project" {
		t.Fatalf("kind body = %#v", (*captures)[0].body)
	}
}

func TestHandleGtdCaptureRejectsMissingTitle(t *testing.T) {
	srv, token, _ := newGtdMutationHarness(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not receive a request")
	})

	_, stderr, code := runGtdMutationCommand(t, srv, token, "capture")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage)", code)
	}
	if !strings.Contains(stderr, "title is required") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestHandleGtdCaptureRejectsBadFlagValue(t *testing.T) {
	srv, token, _ := newGtdMutationHarness(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not receive a request")
	})

	_, stderr, code := runGtdMutationCommand(t, srv, token, "capture", "Title", "--workspace", "abc")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage); stderr=%s", code, stderr)
	}
	if !strings.Contains(stderr, "workspace must be a positive integer") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestHandleGtdCaptureSurfacesValidationError(t *testing.T) {
	srv, token, _ := newGtdMutationHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"kind must be action or project"}`))
	})

	_, stderr, code := runGtdMutationCommand(t, srv, token, "capture", "x", "--kind", "epic")
	if code == 0 {
		t.Fatalf("expected non-zero exit; stderr=%s", stderr)
	}
	if !strings.Contains(stderr, "kind must be action or project") {
		t.Fatalf("stderr = %q", stderr)
	}
}
