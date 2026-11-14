package diagnostic

import (
	"errors"
	"io/fs"
	"testing"
)

func TestStableCodesAndExitStatus(t *testing.T) {
	for _, tt := range []struct {
		err  error
		code string
		exit int
	}{
		{New("ARROW_DRIFT", "files differ"), "ARROW_DRIFT", 3},
		{fs.ErrNotExist, "ARROW_NOT_FOUND", 1}, {fs.ErrPermission, "ARROW_PERMISSION", 1}, {fs.ErrExist, "ARROW_EXISTS", 1}, {errors.New("write failed"), "ARROW_IO", 1}, {New("ARROW_USAGE", "bad operand"), "ARROW_USAGE", 2}, {Wrap("ARROW_CONFIG", "decode", fs.ErrPermission), "ARROW_CONFIG", 2}, {New("ARROW_RATE_LIMIT", "retry later"), "ARROW_RATE_LIMIT", 1},
	} {
		if Code(tt.err) != tt.code || Exit(tt.err) != tt.exit {
			t.Errorf("%v: %s %d", tt.err, Code(tt.err), Exit(tt.err))
		}
	}
	if Exit(nil) != 0 {
		t.Fatal("nil error must succeed")
	}
	if !errors.Is(Wrap("ARROW_IO", "write", fs.ErrPermission), fs.ErrPermission) {
		t.Fatal("lost underlying cause")
	}
}
