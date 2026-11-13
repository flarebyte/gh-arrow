package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/flarebyte/gh-arrow/internal/listing"
	"github.com/flarebyte/gh-arrow/internal/local"
)

const diffLimit = 8 * 1024 * 1024

func readDiff(r io.ReadCloser) ([]byte, error) {
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, diffLimit+1))
}
func syncDiff(ctx context.Context, current *local.Backend, items []syncItem, out io.Writer) error {
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		record := item.record
		if record.Status == "unchanged" {
			continue
		}
		var before []byte
		if record.Status != "missing" {
			f, err := current.Open(ctx, record.Destination)
			if err != nil {
				return err
			}
			before, err = readDiff(f)
			if err != nil {
				return err
			}
		}
		f, err := os.Open(item.staged)
		if err != nil {
			return err
		}
		after, err := readDiff(f)
		if err != nil {
			return err
		}
		oldLabel, newLabel := "a/"+listing.Escape(record.Destination), "b/"+listing.Escape(record.Destination)
		if record.Status == "missing" {
			oldLabel = "/dev/null"
		}
		if record.Status == "missing" && len(after) == 0 {
			_, err = fmt.Fprintf(out, "Empty file %s is missing locally\n", newLabel)
		} else if len(before) > diffLimit || len(after) > diffLimit {
			_, err = fmt.Fprintf(out, "Files %s and %s differ (8 MiB diff display limit)\n", oldLabel, newLabel)
		} else if bytes.ContainsRune(before, 0) || bytes.ContainsRune(after, 0) || !utf8.Valid(before) || !utf8.Valid(after) {
			_, err = fmt.Fprintf(out, "Binary files %s and %s differ\n", oldLabel, newLabel)
		} else {
			err = unifiedDiff(out, oldLabel, newLabel, before, after)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// A single replacement hunk trims common prefix/suffix and keeps three context
// lines. Linear work avoids quadratic diff behaviour for large generated files.
func unifiedDiff(out io.Writer, oldLabel, newLabel string, before, after []byte) error {
	lines := func(data []byte) []string {
		if len(data) == 0 {
			return nil
		}
		result := strings.SplitAfter(string(data), "\n")
		if result[len(result)-1] == "" {
			result = result[:len(result)-1]
		}
		return result
	}
	a, b := lines(before), lines(after)
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	start := max(0, prefix-3)
	endA, endB := len(a)-max(0, suffix-3), len(b)-max(0, suffix-3)
	countA, countB := endA-start, endB-start
	lineA, lineB := start+1, start+1
	if countA == 0 {
		lineA = start
	}
	if countB == 0 {
		lineB = start
	}
	var patch strings.Builder
	fmt.Fprintf(&patch, "--- %s\n+++ %s\n@@ -%d,%d +%d,%d @@\n", oldLabel, newLabel, lineA, countA, lineB, countB)
	write := func(prefix byte, line string) {
		patch.WriteByte(prefix)
		patch.WriteString(line)
		if !strings.HasSuffix(line, "\n") {
			patch.WriteString("\n\\ No newline at end of file\n")
		}
	}
	for i := start; i < prefix; i++ {
		write(' ', a[i])
	}
	for i := prefix; i < len(a)-suffix; i++ {
		write('-', a[i])
	}
	for i := prefix; i < len(b)-suffix; i++ {
		write('+', b[i])
	}
	for i := len(a) - suffix; i < endA; i++ {
		write(' ', a[i])
	}
	_, err := io.WriteString(out, patch.String())
	return err
}
