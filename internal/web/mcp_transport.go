package web

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// mcpEndpoint addresses an embedded sloptools MCP listener. In production
// the listener binds a Unix domain socket (mode 0600) — TCP MCP listeners
// are not allowed because they leak to other UIDs on the host (cf.
// multi-user threat model). The httpURL field is reserved for httptest-style
// in-process test servers and must not be used outside tests.
type mcpEndpoint struct {
	socket  string
	httpURL string
}

func (e mcpEndpoint) ok() bool {
	return strings.TrimSpace(e.socket) != "" || strings.TrimSpace(e.httpURL) != ""
}

// HTTPURL returns the absolute URL to POST against for the given route.
func (e mcpEndpoint) HTTPURL(route string) string {
	if route == "" {
		route = "/"
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	if e.socket != "" {
		return "http://unix" + route
	}
	return strings.TrimRight(e.httpURL, "/") + route
}

// WSURL returns the websocket URL for the given route.
func (e mcpEndpoint) WSURL(route string) string {
	if route == "" {
		route = "/"
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	if e.socket != "" {
		return "ws://unix" + route
	}
	base := strings.TrimRight(e.httpURL, "/")
	switch {
	case strings.HasPrefix(base, "https://"):
		base = "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		base = "ws://" + strings.TrimPrefix(base, "http://")
	}
	return base + route
}

// HTTPClient returns an *http.Client appropriate for the endpoint's transport.
func (e mcpEndpoint) HTTPClient(timeout time.Duration) *http.Client {
	if e.socket != "" {
		socket := e.socket
		dial := func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socket)
		}
		return &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext:           dial,
				ResponseHeaderTimeout: timeout,
				MaxIdleConns:          16,
				MaxIdleConnsPerHost:   16,
				IdleConnTimeout:       30 * time.Second,
			},
		}
	}
	return &http.Client{Timeout: timeout}
}

// WSDialer returns a websocket.Dialer appropriate for the endpoint's transport.
func (e mcpEndpoint) WSDialer() *websocket.Dialer {
	if e.socket != "" {
		socket := e.socket
		return &websocket.Dialer{
			HandshakeTimeout: 5 * time.Second,
			NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socket)
			},
		}
	}
	return &websocket.Dialer{HandshakeTimeout: 5 * time.Second}
}

// parseEndpoint accepts either an empty string (returns zero-value endpoint),
// a unix:/path/to/sock URL, a bare absolute path, or an http(s):// URL. The
// http(s):// form is reserved for in-process httptest servers; production
// configuration must use a unix socket because plaintext loopback HTTP leaks
// to other UIDs on the host.
func parseEndpoint(raw string) (mcpEndpoint, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return mcpEndpoint{}, nil
	}
	if strings.HasPrefix(s, "unix:") {
		path := strings.TrimPrefix(s, "unix:")
		path = strings.TrimPrefix(path, "//")
		if path == "" {
			return mcpEndpoint{}, errors.New("empty unix socket path")
		}
		return mcpEndpoint{socket: filepath.Clean(path)}, nil
	}
	if strings.HasPrefix(s, "/") {
		return mcpEndpoint{socket: filepath.Clean(s)}, nil
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		s = strings.TrimSuffix(s, "/mcp")
		s = strings.TrimRight(s, "/")
		return mcpEndpoint{httpURL: s}, nil
	}
	return mcpEndpoint{}, fmt.Errorf("unrecognized MCP endpoint: %q", s)
}

// defaultLocalMCPSocket returns the conventional per-user socket path for the
// slopshell-embedded sloppy MCP. Linux: $XDG_RUNTIME_DIR/sloppy/mcp.sock. On
// macOS launchd does not export XDG_RUNTIME_DIR by default, so we fall back
// to $HOME/Library/Caches/sloppy/mcp.sock. The parent dir is created with
// 0700 by the StartUnix path; we just compute the location here.
func defaultLocalMCPSocket() string {
	if v := strings.TrimSpace(os.Getenv("SLOPSHELL_MCP_SOCKET")); v != "" {
		return v
	}
	if runtime.GOOS == "darwin" {
		home := strings.TrimSpace(os.Getenv("HOME"))
		if home == "" {
			return ""
		}
		return filepath.Join(home, "Library", "Caches", "sloppy", "mcp.sock")
	}
	if rt := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); rt != "" {
		return filepath.Join(rt, "sloppy", "mcp.sock")
	}
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".cache", "sloppy", "mcp.sock")
}

// workspaceSocketPath returns a unique per-session unix socket path for an
// embedded sloptools MCP serving a workspace project.
func workspaceSocketPath(sessionID string) string {
	base := strings.TrimSpace(os.Getenv("SLOPSHELL_WORKSPACE_SOCKET_DIR"))
	if base == "" {
		if rt := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); rt != "" {
			base = filepath.Join(rt, "sloppy", "workspaces")
		} else if runtime.GOOS == "darwin" {
			base = filepath.Join(strings.TrimSpace(os.Getenv("HOME")), "Library", "Caches", "sloppy", "workspaces")
		} else {
			base = filepath.Join(strings.TrimSpace(os.Getenv("HOME")), ".cache", "sloppy", "workspaces")
		}
	}
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, sessionID)
	if clean == "" {
		clean = "session"
	}
	return filepath.Join(base, clean+".sock")
}

// waitForUnixMCPReady blocks until the socket exists and a /health probe
// returns 200, or the deadline elapses. errCh signals an early exit if the
// listener goroutine returned.
func waitForUnixMCPReady(ep mcpEndpoint, timeout time.Duration, errCh <-chan error) error {
	if !ep.ok() {
		return errors.New("waitForUnixMCPReady: endpoint not configured")
	}
	deadline := time.Now().Add(timeout)
	client := ep.HTTPClient(750 * time.Millisecond)
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			if err == nil {
				return errors.New("mcp listener exited before becoming healthy")
			}
			return fmt.Errorf("mcp listener failed to start: %w", err)
		default:
		}
		resp, err := client.Get(ep.HTTPURL("/health"))
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	select {
	case err := <-errCh:
		if err == nil {
			return errors.New("mcp listener exited before becoming healthy")
		}
		return fmt.Errorf("mcp listener failed to start: %w", err)
	default:
	}
	return errors.New("mcp health check timeout")
}

// httpClientCache caches the per-socket *http.Client so we reuse the
// underlying *http.Transport (and its idle connections) across calls.
var (
	httpClientMu    sync.Mutex
	httpClientCache = map[string]*http.Client{}
)

func cachedHTTPClientForEndpoint(ep mcpEndpoint, timeout time.Duration) *http.Client {
	if !ep.ok() {
		return &http.Client{Timeout: timeout}
	}
	if ep.socket != "" {
		httpClientMu.Lock()
		defer httpClientMu.Unlock()
		if c, ok := httpClientCache[ep.socket]; ok {
			return c
		}
		c := ep.HTTPClient(timeout)
		httpClientCache[ep.socket] = c
		return c
	}
	return ep.HTTPClient(timeout)
}

// rejectURLForEndpoint returns an error if a caller still passes an http://…
// URL. Used by transitional shims while migrating call sites.
func rejectURLForEndpoint(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return fmt.Errorf("plaintext MCP URLs are no longer supported (got %q); use a unix socket", raw)
	}
	return nil
}

// localMCPEndpointURL returns the MCPURL string stored in the workspace row
// for the slopshell-embedded local sloppy MCP. Empty when no endpoint is
// configured.
func (a *App) localMCPEndpointURL() string {
	if a == nil {
		return ""
	}
	if a.localMCPEndpoint.socket != "" {
		return "unix:" + a.localMCPEndpoint.socket
	}
	if a.localMCPEndpoint.httpURL != "" {
		return a.localMCPEndpoint.httpURL
	}
	return ""
}
