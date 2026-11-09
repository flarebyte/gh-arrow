"""Exercise the scaffold without accessing GitHub or modifying user files."""

from pathlib import Path
import subprocess
import sys
import tempfile


def main():
    binary = str(Path(sys.argv[1]).resolve())
    # A throwaway working directory prevents accidental reads of repository config.
    with tempfile.TemporaryDirectory() as directory:
        for flag in ('--help', '-h'):
            result = subprocess.run([binary, flag], cwd=directory, capture_output=True, timeout=10)
            if result.returncode != 0 or b'Usage: gh arrow' not in result.stdout or result.stderr:
                raise RuntimeError('Help smoke check failed')
        for args in ([], ['cp', 'local:fixture:file', 'output'],
                     ['cat', 'local:fixture:file'], ['ls', 'local:fixture:'], ['unknown']):
            result = subprocess.run([binary, *args], cwd=directory, capture_output=True, timeout=10)
            if result.returncode != 2 or result.stdout or not result.stderr.startswith(b'ARROW_USAGE:'):
                raise RuntimeError(f'Scaffold failure check failed for {args!r}')
        if list(Path(directory).iterdir()):
            raise RuntimeError('Scaffold commands unexpectedly created files')
    print('CLI smoke checks passed')


if __name__ == '__main__':
    main()
