package quickimport

import (
	"os/exec"
	"strings"
	"testing"
)

func TestLegacyPythonInstallerNeverInjectsImageMCP(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		python, err = exec.LookPath("python3")
	}
	if err != nil {
		t.Skip("Python is unavailable")
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
