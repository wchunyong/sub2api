package quickimport

import (
	"os"
	"strings"
	"testing"
)

func TestDockerBuildsEmbedQuickImportHelpers(t *testing.T) {
	for _, path := range []string{
		"../../../Dockerfile",
		"../../../deploy/Dockerfile",
		"../../Dockerfile",
	} {
		t.Run(path, func(t *testing.T) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(body)
			generation := strings.Index(text, "go run ./cmd/build-quick-import")
			serverBuild := strings.Index(text, "./cmd/server")
			if generation < 0 || serverBuild < 0 || generation > serverBuild {
				t.Fatal("Dockerfile must generate quick-import helpers before building the server")
			}
		})
	}
}
