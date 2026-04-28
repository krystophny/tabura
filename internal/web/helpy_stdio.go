package web

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// helpyStdioClient runs a single `helpy mcp-stdio` subprocess and dispatches
// JSON-RPC tool calls over its stdin/stdout. The process inherits the agent's
// UID so no other local user can speak to it; there is no listening socket at
// all. One process per slopshell server is sufficient — helpy is stateless
// for our purposes (web search/fetch).
type helpyStdioClient struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	bin     string
	args    []string
	nextID  int64
	dead    bool
}

func newHelpyStdioClient(bin string, args []string) *helpyStdioClient {
	return &helpyStdioClient{bin: bin, args: args}
}

func (c *helpyStdioClient) ensureStarted() error {
	if c.cmd != nil && !c.dead {
		return nil
	}
	cmd := exec.Command(c.bin, c.args...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("helpy stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("helpy stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("helpy start: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	c.cmd = cmd
	c.stdin = stdin
	c.scanner = scanner
	c.dead = false
	go func() {
		_ = cmd.Wait()
		c.mu.Lock()
		c.dead = true
		c.mu.Unlock()
	}()
	if err := c.initialize(); err != nil {
		c.shutdownLocked()
		return fmt.Errorf("helpy initialize: %w", err)
	}
	return nil
}

func (c *helpyStdioClient) initialize() error {
	_, err := c.callLocked("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "slopshell", "version": "1"},
	})
	return err
}

// Close terminates the helpy subprocess.
func (c *helpyStdioClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdownLocked()
	return nil
}

func (c *helpyStdioClient) shutdownLocked() {
	if c.stdin != nil {
		_ = c.stdin.Close()
		c.stdin = nil
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	c.cmd = nil
	c.scanner = nil
	c.dead = true
}

// ListTools returns the tools advertised by the helpy stdio MCP.
func (c *helpyStdioClient) ListTools() ([]mcpListedTool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureStarted(); err != nil {
		return nil, err
	}
	result, err := c.callLocked("tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	rawTools, _ := result["tools"].([]any)
	tools := make([]mcpListedTool, 0, len(rawTools))
	for _, raw := range rawTools {
		obj, _ := raw.(map[string]any)
		if obj == nil {
			continue
		}
		schema, _ := obj["inputSchema"].(map[string]any)
		tools = append(tools, mcpListedTool{
			Name:        strings.TrimSpace(fmt.Sprint(obj["name"])),
			Description: strings.TrimSpace(fmt.Sprint(obj["description"])),
			InputSchema: schema,
		})
	}
	return tools, nil
}

// CallTool invokes a tool over stdio.
func (c *helpyStdioClient) CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureStarted(); err != nil {
		return nil, err
	}
	type callOutcome struct {
		result map[string]any
		err    error
	}
	outCh := make(chan callOutcome, 1)
	go func() {
		result, err := c.callLocked("tools/call", map[string]any{"name": name, "arguments": args})
		outCh <- callOutcome{result: result, err: err}
	}()
	select {
	case <-ctx.Done():
		c.shutdownLocked()
		return nil, ctx.Err()
	case res := <-outCh:
		if res.err != nil {
			return nil, res.err
		}
		if isErr, _ := res.result["isError"].(bool); isErr {
			return nil, fmt.Errorf("helpy tool %q failed: %s", name, mcpResultErrorText(res.result))
		}
		sc, _ := res.result["structuredContent"].(map[string]any)
		if sc == nil {
			return nil, errors.New("helpy call: missing structuredContent")
		}
		return sc, nil
	}
}

// callLocked sends a JSON-RPC request and reads exactly one response. The
// caller must hold c.mu so that request/response framing stays consistent.
func (c *helpyStdioClient) callLocked(method string, params map[string]any) (map[string]any, error) {
	if c.dead || c.cmd == nil || c.stdin == nil || c.scanner == nil {
		if err := c.ensureStartedLocked(); err != nil {
			return nil, err
		}
	}
	c.nextID++
	id := c.nextID
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	deadline := time.Now().Add(mcpToolsCallTimeout)
	if d, ok := c.stdin.(interface{ SetWriteDeadline(time.Time) error }); ok {
		_ = d.SetWriteDeadline(deadline)
	}
	if _, err := c.stdin.Write(b); err != nil {
		c.shutdownLocked()
		return nil, fmt.Errorf("helpy write: %w", err)
	}
	if !c.scanner.Scan() {
		err := c.scanner.Err()
		c.shutdownLocked()
		if err != nil {
			return nil, fmt.Errorf("helpy read: %w", err)
		}
		return nil, errors.New("helpy: subprocess closed stdout")
	}
	line := c.scanner.Bytes()
	var resp map[string]any
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("helpy decode: %w (line: %s)", err, string(line))
	}
	if e, ok := resp["error"].(map[string]any); ok {
		return nil, fmt.Errorf("helpy error: %v", e["message"])
	}
	result, _ := resp["result"].(map[string]any)
	if result == nil {
		return map[string]any{}, nil
	}
	return result, nil
}

// ensureStartedLocked is the internal variant called from within callLocked
// when an earlier call killed the subprocess and we need to restart it.
func (c *helpyStdioClient) ensureStartedLocked() error {
	cmd := exec.Command(c.bin, c.args...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("helpy stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("helpy stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("helpy start: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	c.cmd = cmd
	c.stdin = stdin
	c.scanner = scanner
	c.dead = false
	go func() {
		_ = cmd.Wait()
		c.mu.Lock()
		c.dead = true
		c.mu.Unlock()
	}()
	// run initialize as part of startup so the MCP handshake completes before
	// we send a real tool/list or tool/call.
	c.nextID++
	id := c.nextID
	initReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "slopshell", "version": "1"},
		},
	}
	b, _ := json.Marshal(initReq)
	b = append(b, '\n')
	if _, err := c.stdin.Write(b); err != nil {
		c.shutdownLocked()
		return fmt.Errorf("helpy initialize write: %w", err)
	}
	if !c.scanner.Scan() {
		err := c.scanner.Err()
		c.shutdownLocked()
		if err != nil {
			return fmt.Errorf("helpy initialize read: %w", err)
		}
		return errors.New("helpy: subprocess closed stdout during initialize")
	}
	return nil
}

// App-level glue.

func (a *App) helpyEnabled() bool {
	if a == nil {
		return false
	}
	return strings.TrimSpace(a.helpyBin) != ""
}

func (a *App) listHelpyTools() ([]mcpListedTool, error) {
	if !a.helpyEnabled() {
		return nil, nil
	}
	a.helpyMu.Lock()
	if a.helpy == nil {
		a.helpy = newHelpyStdioClient(a.helpyBin, a.helpyArgs)
	}
	client := a.helpy
	a.helpyMu.Unlock()
	return client.ListTools()
}

func (a *App) callHelpyTool(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if !a.helpyEnabled() {
		return nil, errors.New("helpy stdio MCP is not configured")
	}
	a.helpyMu.Lock()
	if a.helpy == nil {
		a.helpy = newHelpyStdioClient(a.helpyBin, a.helpyArgs)
	}
	client := a.helpy
	a.helpyMu.Unlock()
	return client.CallTool(ctx, name, args)
}

func (a *App) shutdownHelpy() {
	a.helpyMu.Lock()
	client := a.helpy
	a.helpy = nil
	a.helpyMu.Unlock()
	if client != nil {
		_ = client.Close()
	}
}
