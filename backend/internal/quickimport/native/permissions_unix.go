//go:build !windows

package native

import (
	"errors"
	"os"
)

func protect(path string, dir bool) error {
	mode := os.FileMode(0600)
	if dir {
		mode = 0700
	}
	return os.Chmod(path, mode)
}
func rejectLink(path string) error {
	fi, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return errors.New("linked configuration paths require manual setup")
	}
	return nil
}
func replaceFile(from, to string) error { return os.Rename(from, to) }
