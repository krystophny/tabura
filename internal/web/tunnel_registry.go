package web

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sloppy-org/slopshell/internal/serve"
)

// extractPort parses a TCP-style URL and returns the port. Used only by the
// legacy tunnel-registry test shim.
func extractPort(raw string) (int, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return 0, err
	}
	p := u.Port()
	if p == "" {
		switch u.Scheme {
		case "https":
			return 443, nil
		case "http":
			return 80, nil
		default:
			return 0, errors.New("cannot infer port")
		}
	}
	return strconv.Atoi(p)
}

type tunnelRegistry struct {
	mu             sync.Mutex
	endpoints      map[string]mcpEndpoint
	remoteCanvas   map[string]*websocket.Conn
	relayCancel    map[string]context.CancelFunc
	projectApps    map[string]*serve.App
	projectStop    map[string]context.CancelFunc
	localApp       *serve.App
	localAppCancel context.CancelFunc
}

func newTunnelRegistry() *tunnelRegistry {
	return &tunnelRegistry{
		endpoints:    map[string]mcpEndpoint{},
		remoteCanvas: map[string]*websocket.Conn{},
		relayCancel:  map[string]context.CancelFunc{},
		projectApps:  map[string]*serve.App{},
		projectStop:  map[string]context.CancelFunc{},
	}
}

func (t *tunnelRegistry) getEndpoint(sessionID string) (mcpEndpoint, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	ep, ok := t.endpoints[sessionID]
	return ep, ok
}

func (t *tunnelRegistry) setEndpoint(sessionID string, ep mcpEndpoint) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.endpoints[sessionID] = ep
}

func (t *tunnelRegistry) hasEndpoint(sessionID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.endpoints[sessionID]
	return ok
}

// setPort is a test-only shim. Production code switched to unix-socket
// endpoints; tests still spin up httptest.NewServer instances and need a way
// to register the resulting TCP loopback URL. Do not use from non-test code.
func (t *tunnelRegistry) setPort(sessionID string, port int) {
	if port <= 0 {
		return
	}
	t.setEndpoint(sessionID, mcpEndpoint{httpURL: fmt.Sprintf("http://127.0.0.1:%d", port)})
}

// getPort returns a TCP port if the stored endpoint happens to be an
// httptest URL. Test-only.
func (t *tunnelRegistry) getPort(sessionID string) (int, bool) {
	ep, ok := t.getEndpoint(sessionID)
	if !ok || ep.httpURL == "" {
		return 0, false
	}
	port, err := extractPort(ep.httpURL)
	if err != nil {
		return 0, false
	}
	return port, true
}

// hasPort is preserved as a test-only shim and reports whether any endpoint
// is registered for sessionID.
func (t *tunnelRegistry) hasPort(sessionID string) bool { return t.hasEndpoint(sessionID) }

func (t *tunnelRegistry) getRemoteCanvas(sessionID string) *websocket.Conn {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.remoteCanvas[sessionID]
}

func (t *tunnelRegistry) setRemoteCanvas(sessionID string, ws *websocket.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.remoteCanvas[sessionID] = ws
}

func (t *tunnelRegistry) deleteRemoteCanvas(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if rc := t.remoteCanvas[sessionID]; rc != nil {
		_ = rc.Close()
	}
	delete(t.remoteCanvas, sessionID)
}

func (t *tunnelRegistry) replaceRelayCancel(sessionID string) context.Context {
	t.mu.Lock()
	if cancel := t.relayCancel[sessionID]; cancel != nil {
		cancel()
		delete(t.relayCancel, sessionID)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.relayCancel[sessionID] = cancel
	t.mu.Unlock()
	return ctx
}

func (t *tunnelRegistry) deleteRelayCancel(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.relayCancel, sessionID)
}

func (t *tunnelRegistry) setProjectServe(sessionID string, app *serve.App, cancel context.CancelFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.projectApps[sessionID] = app
	t.projectStop[sessionID] = cancel
}

func (t *tunnelRegistry) setLocalServe(app *serve.App, cancel context.CancelFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.localApp = app
	t.localAppCancel = cancel
}

func (t *tunnelRegistry) shutdown(ctx context.Context) {
	t.mu.Lock()
	for _, cancel := range t.relayCancel {
		cancel()
	}
	for _, ws := range t.remoteCanvas {
		_ = ws.Close()
	}
	projectStops := make(map[string]context.CancelFunc, len(t.projectStop))
	for sid, cancel := range t.projectStop {
		projectStops[sid] = cancel
	}
	projectApps := make(map[string]*serve.App, len(t.projectApps))
	for sid, app := range t.projectApps {
		projectApps[sid] = app
	}
	localApp := t.localApp
	localCancel := t.localAppCancel
	t.mu.Unlock()

	for _, cancel := range projectStops {
		cancel()
	}
	for _, app := range projectApps {
		if app != nil {
			_ = app.Stop(ctx)
		}
	}
	if localApp != nil {
		_ = localApp.Stop(ctx)
	}
	if localCancel != nil {
		localCancel()
	}
}
