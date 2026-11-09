import hashlib
from pathlib import Path
import tempfile
import unittest

from build_release import main, stage
from release_preflight import PreflightError, release_metadata
from unittest.mock import patch


class ReleaseStagingTests(unittest.TestCase):
    def test_build_stages_manifest_assets_and_excludes_temporary_binaries(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            metadata = {'artifact_dir': 'build/release', 'checksum_name': 'checksums.txt'}

            def build(command, check):
                self.assertEqual(command[:4], ['gh', 'flarebyte', 'build', '--output-dir'])
                self.assertTrue(check)
                output = Path(command[4])
                (output / '.tmp').mkdir(parents=True)
                (output / '.tmp/cli').write_bytes(b'temporary binary')
                (output / 'cli').write_bytes(b'CLI')
                digest = hashlib.sha256(b'CLI').hexdigest()
                (output / 'checksums.txt').write_text(f'{digest}  cli\n')

            with (patch('build_release.Path.cwd', return_value=root),
                  patch('build_release.release_metadata', return_value=metadata),
                  patch('build_release.subprocess.run', side_effect=build)):
                main()
            self.assertEqual(sorted(p.name for p in (root / 'build/release').iterdir()),
                             ['checksums.txt', 'cli'])

    def test_only_verified_assets_can_be_staged(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            source = root / 'assets'
            source.mkdir()
            (source / 'cli').write_bytes(b'CLI')
            digest = hashlib.sha256(b'CLI').hexdigest()
            (source / 'checksums.txt').write_text(f'{digest}  cli\n')
            destination = root / 'release'
            stage(source, destination, 'checksums.txt')
            self.assertEqual((destination / 'cli').read_bytes(), b'CLI')
            with self.assertRaises(FileExistsError):
                stage(source, destination, 'checksums.txt')
            (source / 'cli').write_bytes(b'corrupt')
            with self.assertRaisesRegex(PreflightError, 'Checksum mismatch'):
                stage(source, root / 'corrupt-release', 'checksums.txt')
            self.assertFalse((root / 'corrupt-release').exists())

    def test_scaffold_version_cannot_be_published(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / '.gh-flarebyte.cue').write_text('''build: { checksumFile: "build/checksums.txt" }
release: {
 versionSource: "version.yaml"
 tagPrefix: "v"
 artifactDir: "build/release"
 notesMode: "generate-notes"
}
''')
            (root / 'version.yaml').write_text('version: 0.0.0-dev\n')
            with self.assertRaisesRegex(PreflightError, 'scaffold-only'):
                release_metadata(root)


if __name__ == '__main__':
    unittest.main()
