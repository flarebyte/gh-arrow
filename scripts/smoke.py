"""Exercise the executable against temporary local fixtures, without GitHub access."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile


def main():
    binary = str(Path(sys.argv[1]).resolve())
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        project = root / 'project'
        project.mkdir()
        (project / 'docs').mkdir()
        raw = bytes([0, 255, 13, 10])
        (project / 'docs' / 'a.md').write_bytes(raw)
        (project / 'docs' / 'z.md').write_bytes(b'z')
        config = root / '.gh-arrow'
        config.write_text(json.dumps({
            'local': {'mappings': [{'repo': 'fixture', 'root': str(project)}]},
            'github': {'mappings': [{'repo': 'owner:repo', 'root': str(project)}]},
        }))

        def run(*args):
            return subprocess.run([binary, *args], cwd=root, env={**os.environ, "HOME": str(root), "USERPROFILE": str(root)}, capture_output=True, timeout=10)

        for flag in ('--help', '-h'):
            result = run(flag)
            if result.returncode != 0 or b'Usage: gh arrow' not in result.stdout or result.stderr:
                raise RuntimeError('Help smoke check failed')
        version = run('--version', '--config', str(root / 'missing'))
        if version.returncode != 0 or not version.stdout.startswith(b'gh-arrow ') or version.stderr:
            raise RuntimeError('Offline version smoke check failed')
        for command in ('cp', 'cat', 'ls'):
            if run(command, '--help', '--config', str(root / 'missing')).returncode != 0:
                raise RuntimeError('Command help should not load config')
        result = run('cat', 'local:fixture:docs/a.md')
        if result.returncode != 0 or result.stdout != raw or result.stderr:
            raise RuntimeError('Binary cat smoke check failed')
        result = run('ls', 'local:fixture:docs/*.md', '--json')
        if result.returncode != 0 or json.loads(result.stdout) != [{'path': 'a.md'}, {'path': 'z.md'}]:
            raise RuntimeError('JSON listing smoke check failed')
        result = run('cp', 'local:fixture:docs/a.md', 'output')
        if result.returncode != 0 or result.stdout or (root / 'output').read_bytes() != raw:
            raise RuntimeError('Copy smoke check failed')
        result = run('cp', 'local:fixture:docs/z.md', 'output')
        if result.returncode != 1 or not result.stderr.startswith(b'ARROW_EXISTS:') or (root / 'output').read_bytes() != raw:
            raise RuntimeError('Copy no-clobber check failed')
        if run('cp', 'local:fixture:docs/z.md', 'output', '--force').returncode != 0:
            raise RuntimeError('Forced copy smoke check failed')
        result = run('cat', 'github:owner:repo:missing')
        if result.returncode != 1 or result.stdout or not result.stderr.startswith(b'ARROW_NOT_FOUND:'):
            raise RuntimeError('Mapped source must not fall back to GitHub')
        for args in ([], ['unknown'], ['cat', 'local:fixture:../outside'], ['cat', 'local:fixture:docs/a.md', '--json']):
            result = run(*args)
            if result.returncode != 2 or result.stdout or not result.stderr.startswith(b'ARROW_USAGE:'):
                raise RuntimeError(f'Usage check failed for {args!r}')
    print('CLI smoke checks passed')


if __name__ == '__main__':
    main()
