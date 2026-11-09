#!/usr/bin/env python3
"""Validate the searchable design CSV convention using only the standard library."""

import argparse
import csv
import re
import sys
from pathlib import Path


CONCEPT_ID = re.compile(r"[a-z][a-z0-9-]*(?:\.[a-z][a-z0-9-]*)*")


def validate(root, single_line=False, unique=()):
    errors, warnings, tables = [], [], {}
    if not root.is_dir():
        return [f"{root}: examples directory does not exist"], warnings

    for path in sorted(root.rglob("*.csv")):
        name = path.relative_to(root).as_posix()
        rows = []
        try:
            with path.open(encoding="utf-8-sig", newline="") as stream:
                reader = csv.reader(stream, strict=True)
                header = next(reader, None)
                if not header or any(not cell.strip() for cell in header):
                    errors.append(f"{name}: missing or empty header")
                    continue
                if len(set(header)) != len(header):
                    errors.append(f"{name}: duplicate column names")
                    continue
                previous_line = reader.line_num
                if previous_line != 1:
                    errors.append(f"{name}: header must occupy one physical line")
                for row in reader:
                    line = previous_line + 1
                    if reader.line_num > line:
                        message = f"{name}:{line}: record spans physical lines"
                        (errors if single_line else warnings).append(message)
                    previous_line = reader.line_num
                    if len(row) != len(header):
                        errors.append(
                            f"{name}:{line}: expected {len(header)} fields, got {len(row)}"
                        )
                        continue
                    rows.append((line, dict(zip(header, row))))
                tables[name] = (header, rows)
        except (OSError, UnicodeError, csv.Error) as exc:
            errors.append(f"{name}: {exc}")

    registry = tables.get("concepts.csv")
    concepts = {}
    if registry is None:
        errors.append("concepts.csv: canonical registry missing or unreadable")
    elif registry[0] != ["id", "description", "parent"]:
        errors.append("concepts.csv: expected header id,description,parent")
    else:
        for line, row in registry[1]:
            identifier = row["id"]
            if not CONCEPT_ID.fullmatch(identifier):
                errors.append(f"concepts.csv:{line}: invalid concept ID {identifier!r}")
            if identifier in concepts:
                errors.append(f"concepts.csv:{line}: duplicate ID {identifier!r}")
            else:
                concepts[identifier] = row["parent"]
            if not row["description"].strip():
                errors.append(f"concepts.csv:{line}: empty description")

        for identifier, parent in concepts.items():
            if parent and parent not in concepts:
                errors.append(f"concepts.csv: {identifier!r} has unknown parent {parent!r}")
            if parent and not identifier.startswith(parent + "."):
                errors.append(f"concepts.csv: {parent!r} is not an ancestor of {identifier!r}")
        # Follow each parent chain without recursion, including malformed hierarchies.
        completed = set()
        for identifier in concepts:
            chain, node = set(), identifier
            while node in concepts and node not in completed:
                if node in chain:
                    errors.append(f"concepts.csv: parent cycle at {node!r}")
                    break
                chain.add(node)
                node = concepts[node]
            completed.update(chain)

    for name, (header, rows) in tables.items():
        if "concept" in header:
            for line, row in rows:
                if not row["concept"] or row["concept"] not in concepts:
                    errors.append(f"{name}:{line}: unknown concept {row['concept']!r}")

    for constraint in unique:
        name, separator, column = constraint.rpartition(":")
        if not separator or not name or not column:
            errors.append(f"invalid --unique {constraint!r}; expected FILE:COLUMN")
            continue
        if name not in tables or column not in tables[name][0]:
            errors.append(f"{constraint}: file or column missing")
            continue
        seen = set()
        for line, row in tables[name][1]:
            value = row[column]
            if not value.strip() or value in seen:
                errors.append(f"{name}:{line}: empty or duplicate {column} {value!r}")
            seen.add(value)
    return errors, warnings


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("examples", type=Path)
    parser.add_argument("--single-line", action="store_true")
    parser.add_argument("--unique", action="append", default=[], metavar="FILE:COLUMN")
    args = parser.parse_args()
    errors, warnings = validate(args.examples, args.single_line, args.unique)
    for message in warnings:
        print(f"warning: {message}", file=sys.stderr)
    for message in errors:
        print(f"error: {message}", file=sys.stderr)
    if errors:
        return 1
    print("CSV integrity and concept references valid.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
