// Command gh-arrow is the scaffold for the gh arrow GitHub CLI extension.
package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Println(`Usage: gh arrow <command>

Development scaffold. File operations are not implemented yet.

Planned commands:
  cp   Copy one file
  cat  Write exact file bytes
  ls   List or discover files

See doc/design/specification.md for the planned interface.`)
		return
	}
	fmt.Fprintln(os.Stderr, "ARROW_USAGE: development scaffold; commands are not implemented; use --help")
	os.Exit(2)
}
