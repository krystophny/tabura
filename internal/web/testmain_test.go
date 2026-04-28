package web

import (
	"os"
	"testing"
)

// TestMain disables the stdio helpy MCP subprocess for all tests in this
// package. Production runtime spawns `helpy mcp-stdio`, but tests use
// httptest-backed MCPs and must not fork a real subprocess.
func TestMain(m *testing.M) {
	_ = os.Setenv("SLOPSHELL_HELPY_BIN", "off")
	os.Exit(m.Run())
}
