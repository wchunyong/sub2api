package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/quickimport"
	"github.com/gin-gonic/gin"
)

// QuickImportBootstrap supplies the same verified downloader through a short URL.
// Configuration tickets remain command arguments and are never put in this URL.
func QuickImportBootstrap(c *gin.Context) {
	target := c.Param("target")
	ext := ""
	for _, candidate := range []string{".ps1", ".sh"} {
		if strings.HasSuffix(target, candidate) {
			ext = candidate
			break
		}
	}
	agent := strings.TrimSuffix(target, ext)
	clean := strings.HasSuffix(agent, "-clean")
	agent = strings.TrimSuffix(agent, "-clean")
	if ext == "" || (agent != "claude" && agent != "codex" && agent != "opencode") {
		c.Status(http.StatusNotFound)
		return
	}
	base := "https://" + c.Request.Host
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		c.Status(http.StatusBadRequest)
		return
	}
	body, err := quickimport.Assets.ReadFile("assets/install" + ext)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	script := strings.ReplaceAll(string(body), "\r\n", "\n")
	action := "install"
	if clean {
		action = "clean"
	}
	if ext == ".ps1" {
		_, rest, ok := strings.Cut(script, "$ErrorActionPreference = 'Stop'")
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		script = "param([Parameter(Position=0)][string]$Ticket)\n$Action = '" + action + "'\n$Agent = '" + agent + "'\n$Server = '" + strings.ReplaceAll(base, "'", "''") + "'\n$ErrorActionPreference = 'Stop'" + rest
	} else {
		base = "'" + strings.ReplaceAll(base, "'", "'\"'\"'") + "'"
		script = strings.Replace(script, "#!/bin/sh\n", "#!/bin/sh\nset -- "+action+" "+agent+" "+base+" \"$@\"\n", 1)
		if clean {
			script = strings.ReplaceAll(script, "\"$binary\" clean --agent \"$agent\"", "\"$binary\" clean --agent \"$agent\" </dev/tty")
		}
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(script))
}
