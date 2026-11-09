//go:build windows

package local

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestTemporaryFileHasPrivateACL(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	file, err := temporaryFile(root, "private")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	sd, err := windows.GetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil || acl.AceCount != 1 {
		t.Fatal("temporary file must grant access only to its creator", err)
	}
}
