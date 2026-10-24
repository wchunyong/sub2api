package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestQuickImportBootstrapShortCommands(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/setup/:target", QuickImportBootstrap)
	for _, tc := range []struct{ path, expected string }{
		{"opencode.ps1", "$Agent = 'opencode'"},
		{"codex.sh", "set -- install codex 'https://example.com'"},
		{"claude-clean.ps1", "$Action = 'clean'"},
		{"opencode-clean.sh", "</dev/tty"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "https://example.com/setup/"+tc.path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), tc.expected) {
			t.Fatalf("bad bootstrap %s: %d", tc.path, w.Code)
		}
		if strings.Contains(w.Body.String(), "python3") || strings.Contains(w.Body.String(), "python ") {
			t.Fatal("Python dependency remains")
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("bootstrap should not be cached")
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "https://example.com/setup/invalid.ps1", nil))
	if w.Code != 404 {
		t.Fatal("unknown agent accepted")
	}
}

func TestQuickImportPowerShellBootstrapDownloadsAndInvokesVerifiedHelper(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows PowerShell bootstrap")
	}
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		t.Skip("Windows PowerShell unavailable")
	}
	root := t.TempDir()
	stubSource := filepath.Join(root, "helper.go")
	stubBinary := filepath.Join(root, "helper.exe")
	if err := os.WriteFile(stubSource, []byte(`package main
import ("os"; "strings")
func main() { _ = os.WriteFile(os.Getenv("QUICK_IMPORT_TEST_ARGS"), []byte(strings.Join(os.Args[1:], " ")), 0600) }
`), 0600); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", stubBinary, stubSource)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v: %s", err, output)
	}
	binary, err := os.ReadFile(stubBinary)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(binary)
	checksum := hex.EncodeToString(sum[:])
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/setup/:target", QuickImportBootstrap)
	const ticket = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	powershell := `$ErrorActionPreference='Stop'; Import-Module Microsoft.PowerShell.Utility; function Invoke-WebRequest { param([switch]$UseBasicParsing,[int]$MaximumRedirection,[int]$TimeoutSec,[hashtable]$Headers,[string]$Uri,[string]$OutFile); if ($Uri.EndsWith('.sha256')) { return [pscustomobject]@{ Content=$env:QUICK_IMPORT_TEST_CHECKSUM } }; Copy-Item -LiteralPath $env:QUICK_IMPORT_TEST_BINARY -Destination $OutFile }; & ([scriptblock]::Create([IO.File]::ReadAllText($env:QUICK_IMPORT_TEST_SCRIPT))) '` + ticket + `'`
	for _, agent := range []string{"claude", "codex", "deepseek-harness", "workbuddy", "opencode", "pi"} {
		t.Run(agent, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", "https://example.com/setup/"+agent+".ps1", nil))
			if w.Code != 200 {
				t.Fatalf("bootstrap status: %d", w.Code)
			}
			scriptPath := filepath.Join(root, agent+".ps1")
			if err := os.WriteFile(scriptPath, w.Body.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			argsPath := filepath.Join(root, agent+"-args.txt")
			home := filepath.Join(root, agent+"-user")
			if err := os.Mkdir(home, 0700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", powershell)
			cmd.Env = append(os.Environ(), "USERPROFILE="+home, "QUICK_IMPORT_TEST_CHECKSUM="+checksum, "QUICK_IMPORT_TEST_BINARY="+stubBinary, "QUICK_IMPORT_TEST_SCRIPT="+scriptPath, "QUICK_IMPORT_TEST_ARGS="+argsPath)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("PowerShell bootstrap: %v: %s", err, output)
			}
			invocation, err := os.ReadFile(argsPath)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(invocation); got != "install --agent "+agent+" --server https://example.com --ticket "+ticket {
				t.Fatalf("unexpected helper invocation: %q", got)
			}
			if _, err := os.Stat(filepath.Join(home, ".sub2api-quick-import", "bin", checksum+".exe")); err != nil {
				t.Fatalf("verified helper cache: %v", err)
			}
			restore, err := os.ReadFile(filepath.Join(home, ".sub2api-quick-import", agent, "restore.ps1"))
			if err != nil || !strings.Contains(string(restore), "clean --agent '"+agent+"'") {
				t.Fatalf("recovery script: %v", err)
			}
		})
	}
}
