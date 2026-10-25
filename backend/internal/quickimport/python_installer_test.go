package quickimport

import (
	"os/exec"
	"strings"
	"testing"
)

func TestLegacyPythonInstallerNeverInjectsImageMCP(t *testing.T) {
	python := ""
	for _, name := range []string{"python3.11", "python", "python3"} {
		candidate, err := exec.LookPath(name)
		if err == nil && exec.Command(candidate, "-c", "import sys; sys.exit(0 if sys.version_info >= (3, 11) else 1)").Run() == nil {
			python = candidate
			break
		}
	}
	if python == "" {
		t.Skip("legacy installer requires Python 3.11 or newer")
	}
	const script = `import runpy
installer = runpy.run_path('assets/installer.py')
for agent in ('claude', 'codex', 'opencode'):
    payload = {'version': 1, 'agent': agent, 'api_key': 'test-secret', 'base_url': 'https://example.test/v1', 'model': 'test-model', 'mcp_endpoint': 'https://example.test/mcp', 'mcp_image_tools_enabled': True}
    changes = installer['configuration'](payload)
    if any(change['path'][0] in ('mcpServers', 'mcp_servers', 'mcp') for change in changes):
        raise AssertionError(agent + ' one-click installer injected image MCP')
`
	cmd := exec.Command(python, "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("legacy Python one-click script still injects MCP: %v %s", err, strings.TrimSpace(string(output)))
	}
}
