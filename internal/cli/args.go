package cli

import (
	"strings"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/listing"
	"github.com/flarebyte/gh-arrow/internal/source"
)

type options struct {
	command, config, ref, owner, topic, file, syncAction string
	explicit, force, json, help, version, dryRun         bool
	args                                                 []string
}

func usage(message string) error { return diagnostic.New("ARROW_USAGE", message) }
func parse(args []string) (options, error) {
	o := options{}
	seen := map[string]bool{}
	positional := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" && !positional {
			positional = true
			continue
		}
		if !positional && strings.HasPrefix(arg, "-") && arg != "-" {
			name, value, inline := strings.Cut(arg, "=")
			if name == "-h" {
				name = "--help"
			}
			if seen[name] {
				return o, usage("duplicate flag " + name)
			}
			seen[name] = true
			switch name {
			case "--help", "--force", "--json", "--version", "--dry-run":
				if inline {
					return o, usage("boolean flags do not take a value")
				}
				switch name {
				case "--dry-run":
					o.dryRun = true
				case "--version":
					o.version = true
				case "--help":
					o.help = true
				case "--force":
					o.force = true
				case "--json":
					o.json = true
				}
			case "--config", "--ref", "--org", "--topic", "--file":
				if !inline {
					if i+1 == len(args) || strings.HasPrefix(args[i+1], "--") {
						return o, usage("missing flag value for " + name)
					}
					i++
					value = args[i]
				}
				if value == "" {
					return o, usage("empty flag value for " + name)
				}
				switch name {
				case "--file":
					o.file = value
				case "--config":
					o.config = value
					o.explicit = true
				case "--ref":
					o.ref = value
				case "--org":
					o.owner = value
				case "--topic":
					o.topic = value
				}
			default:
				return o, usage("unknown flag " + name)
			}
			continue
		}
		if o.command == "" {
			o.command = arg
		} else {
			o.args = append(o.args, arg)
		}
	}
	if o.command != "" && o.command != "cat" && o.command != "cp" && o.command != "ls" && o.command != "sync" {
		return o, usage("unknown command")
	}
	if o.help {
		return o, nil
	}
	if o.command != "sync" && (o.file != "" || o.dryRun) {
		return o, usage("--file and --dry-run require sync")
	}
	if o.command == "sync" {
		if o.ref != "" || o.force || o.owner != "" || o.topic != "" || o.version || len(o.args) > 1 {
			return o, usage("invalid sync options")
		}
		if len(o.args) == 1 {
			o.syncAction = o.args[0]
		}
		switch o.syncAction {
		case "", "status", "diff", "check":
		default:
			return o, usage("unknown sync subcommand")
		}
		if o.dryRun && o.syncAction != "" || o.json && o.syncAction == "diff" {
			return o, usage("flag is not valid for this sync command")
		}
		if o.file == "" {
			o.file = "sync.gh-arrow.cue"
		}
		return o, nil
	}
	if o.version {
		if o.command != "" || len(o.args) != 0 || o.force || o.json || o.ref != "" || o.owner != "" || o.topic != "" {
			return o, usage("--version is a root-only option")
		}
		return o, nil
	}
	if o.command == "" {
		return o, usage("a command is required; use --help")
	}
	expected := 1
	if o.command == "cp" {
		expected = 2
	}
	if len(o.args) != expected {
		return o, usage("incorrect number of operands")
	}
	if o.force && o.command != "cp" || o.json && o.command != "ls" || o.owner != "" && o.command != "ls" || o.topic != "" && o.owner == "" {
		return o, usage("flag is not valid for this command")
	}
	if o.owner != "" {
		p, err := listing.Parse(o.args[0])
		if err != nil {
			return o, err
		}
		if !source.ValidOwner(o.owner) || o.args[0] == "" || p.Glob || o.ref != "" {
			return o, usage("discovery requires an owner and exact nonempty path; --ref is unsupported")
		}
	} else {
		s, err := source.Parse(o.args[0])
		if err != nil {
			return o, err
		}
		if o.command != "ls" {
			// cat/cp resolve exact names; glob metacharacters are literal here.
			if s.Path == "" {
				return o, usage("file commands require an exact nonempty path")
			}
		} else {
			if _, err := listing.Parse(s.Path); err != nil {
				return o, err
			}
		}
	}
	return o, nil
}
