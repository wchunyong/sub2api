// Build the six dependency-free desktop helpers before embedding the server.
package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err = os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			panic("run inside backend module")
		}
		root = parent
	}
	dir := filepath.Join(root, "internal", "quickimport", "native-assets")
	if err = os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	for _, target := range []string{"windows/amd64", "windows/arm64", "darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64"} {
		parts := strings.Split(target, "/")
		name := "quick-import-" + parts[0] + "-" + parts[1]
		if parts[0] == "windows" {
			name += ".exe"
		}
		path := filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", path, "./cmd/quick-import")
		cmd.Dir = root
		for _, entry := range os.Environ() {
			key := strings.SplitN(entry, "=", 2)[0]
			if key != "GOOS" && key != "GOARCH" && key != "CGO_ENABLED" {
				cmd.Env = append(cmd.Env, entry)
			}
		}
		cmd.Env = append(cmd.Env, "GOOS="+parts[0], "GOARCH="+parts[1], "CGO_ENABLED=0")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err = cmd.Run(); err != nil {
			panic(err)
		}
		data, e := os.ReadFile(path)
		if e != nil {
			panic(e)
		}
		if err = os.WriteFile(path+".sha256", []byte(fmt.Sprintf("%x\n", sha256.Sum256(data))), 0644); err != nil {
			panic(err)
		}
		fmt.Println("Built", name)
	}
}
