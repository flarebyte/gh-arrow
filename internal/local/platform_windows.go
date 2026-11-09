package local

import (
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

func openHandle(root *os.File, name string, access uint32) (windows.Handle, error) {
	object, err := windows.NewNTUnicodeString(filepath.FromSlash(name))
	if err != nil {
		return 0, err
	}
	oa := windows.OBJECT_ATTRIBUTES{RootDirectory: windows.Handle(root.Fd()), ObjectName: object, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var handle windows.Handle
	var status windows.IO_STATUS_BLOCK
	err = windows.NtCreateFile(&handle, access, &oa, &status, nil, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, windows.FILE_OPEN, windows.FILE_SYNCHRONOUS_IO_NONALERT|windows.FILE_OPEN_REPARSE_POINT, 0, 0)
	if nt, ok := err.(windows.NTStatus); ok {
		err = nt.Errno()
	}
	return handle, err
}
func openRelative(root *os.File, name string) (*os.File, error) {
	// OBJ_DONT_REPARSE rejects reparse points in all components, relative to
	// the pinned directory handle, rather than relying on a prior path check.
	handle, err := openHandle(root, name, windows.FILE_GENERIC_READ)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}
	return os.NewFile(uintptr(handle), name), nil
}
func temporaryFile(dir *os.Root, name string) (*os.File, error) {
	parent, err := dir.Open(".")
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	// Go's 0600 mode does not set a Windows ACL. Protect the DACL explicitly
	// so a shared destination directory cannot expose staged private content.
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return nil, err
	}
	object, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return nil, err
	}
	oa := windows.OBJECT_ATTRIBUTES{RootDirectory: windows.Handle(parent.Fd()), ObjectName: object, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE, SecurityDescriptor: sd}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var handle windows.Handle
	var status windows.IO_STATUS_BLOCK
	err = windows.NtCreateFile(&handle, windows.FILE_GENERIC_WRITE|windows.READ_CONTROL, &oa, &status, nil, windows.FILE_ATTRIBUTE_NORMAL, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, windows.FILE_CREATE, windows.FILE_NON_DIRECTORY_FILE|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	runtime.KeepAlive(sd)
	if nt, ok := err.(windows.NTStatus); ok {
		err = nt.Errno()
	}
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), name), nil
}
