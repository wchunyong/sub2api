package native

import (
	"encoding/csv"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"strings"
)

func protect(path string, dir bool) error {
	b, err := exec.Command("whoami", "/user", "/fo", "csv", "/nh").Output()
	if err != nil {
		return errors.New("cannot determine user for credential permissions")
	}
	rows, err := csv.NewReader(strings.NewReader(strings.TrimSpace(string(b)))).ReadAll()
	if err != nil || len(rows) != 1 || len(rows[0]) < 2 {
		return errors.New("cannot determine user SID")
	}
	sid := rows[0][len(rows[0])-1]
	if !strings.HasPrefix(sid, "S-1-") {
		return errors.New("invalid user SID")
	}
	grant := "*" + sid + ":F"
	if dir {
		grant = "*" + sid + ":(OI)(CI)F"
	}
	if err = exec.Command("icacls", path, "/inheritance:r", "/grant:r", grant).Run(); err != nil {
		return errors.New("cannot secure credential permissions")
	}
	return nil
}
func rejectLink(path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attrs, err := windows.GetFileAttributes(p)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return errors.New("linked configuration paths require manual setup")
	}
	return nil
}
func replaceFile(from, to string) error {
	src, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	dst, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(src, dst, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
