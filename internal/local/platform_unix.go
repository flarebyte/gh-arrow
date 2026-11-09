//go:build linux || darwin

package local

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func openRelative(root *os.File, name string) (*os.File, error) {
	fd, err := unix.Openat(int(root.Fd()), ".", unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(name, "/")
	for i, part := range parts {
		flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
		if i < len(parts)-1 {
			flags |= unix.O_DIRECTORY
		}
		next, e := unix.Openat(fd, part, flags, 0)
		unix.Close(fd)
		if e != nil {
			return nil, &os.PathError{Op: "open", Path: name, Err: e}
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), name), nil
}
func temporaryFile(dir *os.Root, name string) (*os.File, error) {
	return dir.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
}
