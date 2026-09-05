package quickimport

import (
	"embed"
	"strings"
)

//go:embed native-assets/*
var nativeAssets embed.FS

func NativeAsset(name string) ([]byte, bool) {
	base := strings.TrimSuffix(name, ".sha256")
	valid := false
	for _, os := range []string{"windows", "darwin", "linux"} {
		for _, arch := range []string{"amd64", "arm64"} {
			candidate := "quick-import-" + os + "-" + arch
			if os == "windows" {
				candidate += ".exe"
			}
			if base == candidate {
				valid = true
			}
		}
	}
	if !valid {
		return nil, false
	}
	data, err := nativeAssets.ReadFile("native-assets/" + name)
	return data, err == nil
}
