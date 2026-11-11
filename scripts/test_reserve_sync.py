"""Exercise the actual Makefile recipes with an offline Arrow substitute."""

import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]
FILES = {
    "pipeline-go-maat.thoth.cue": "shared/go/all",
    "pipeline-go-test-maat.thoth.cue": "shared/go/all",
    "pipeline-ts-e2e-maat.thoth.cue": "shared/typescript/cli",
}


class ReserveSyncTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.fixture = self.root / "reserve"
        shutil.copyfile(ROOT / "Makefile", self.root / "Makefile")
        shutil.copytree(ROOT / ".config", self.root / ".config")
        for name, directory in FILES.items():
            source = self.fixture / directory / name
            source.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / name, source)
            shutil.copyfile(source, self.root / name)
        binary = self.root / "bin"
        binary.mkdir()
        gh = binary / "gh"
        gh.write_text(
            "#!/usr/bin/env python3\n"
            "import os, pathlib, shutil, sys\n"
            "a = sys.argv[1:]\n"
            "assert a[0] == 'arrow'\n"
            "assert a[a.index('--config') + 1] == '.config/reserve-arrow.cue'\n"
            "config = pathlib.Path('.config/reserve-arrow.cue').read_text()\n"
            "assert 'local: mappings: []' in config\n"
            "ref = a[a.index('--ref') + 1]\n"
            "assert len(ref) == 40 and all(c in '0123456789abcdef' for c in ref)\n"
            "prefix = 'github:flarebyte:baldrick-reserve:'\n"
            "assert a[2].startswith(prefix)\n"
            "source = pathlib.Path(os.environ['RESERVE_FIXTURE']) / a[2][len(prefix):]\n"
            "if os.environ.get('FAIL_FETCH'):\n"
            "    if a[1] == 'cat': sys.stdout.write('partial response')\n"
            "    sys.exit(1)\n"
            "if not source.is_file(): sys.exit(1)\n"
            "if a[1] == 'cat': sys.stdout.buffer.write(source.read_bytes())\n"
            "elif a[1] == 'cp':\n"
            "    assert '--force' in a\n"
            "    shutil.copyfile(source, a[3])\n"
            "else: sys.exit(2)\n"
        )
        gh.chmod(0o755)
        self.env = dict(os.environ, PATH=str(binary) + os.pathsep + os.environ["PATH"],
                        RESERVE_FIXTURE=str(self.fixture))

    def run_make(self, target, success=True):
        result = subprocess.run(["make", target], cwd=self.root, env=self.env,
                                capture_output=True, text=True)
        if success:
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_equal_and_repeated_sync(self):
        self.run_make("reserve-check")
        self.run_make("reserve-sync")
        self.run_make("reserve-sync")
        self.run_make("reserve-check")

    def test_each_mapping_detects_and_repairs_drift(self):
        for name in FILES:
            with self.subTest(name=name):
                (self.root / name).write_text("local drift\n")
                self.run_make("reserve-check", success=False)
                self.run_make("reserve-sync")
                self.run_make("reserve-check")

    def test_missing_destination_is_reported_and_sync_creates_it(self):
        for name in FILES:
            with self.subTest(name=name):
                (self.root / name).unlink()
                self.run_make("reserve-check", success=False)
                self.run_make("reserve-sync")
                self.run_make("reserve-check")

    def test_missing_source_fails_without_replacing_destination(self):
        for name, directory in FILES.items():
            with self.subTest(name=name):
                source = self.fixture / directory / name
                data = source.read_bytes()
                source.unlink()
                self.run_make("reserve-check", success=False)
                self.run_make("reserve-sync", success=False)
                self.assertEqual((self.root / name).read_bytes(), data)
                source.write_bytes(data)

    def test_partial_fetch_failure_preserves_destinations(self):
        self.env["FAIL_FETCH"] = "1"
        self.run_make("reserve-check", success=False)
        self.run_make("reserve-sync", success=False)
        for name, directory in FILES.items():
            self.assertEqual((self.root / name).read_bytes(),
                             (self.fixture / directory / name).read_bytes())

    def test_check_rejects_symlink_destination(self):
        name = next(iter(FILES))
        path = self.root / name
        path.unlink()
        path.symlink_to(self.fixture / FILES[name] / name)
        self.run_make("reserve-check", success=False)


if __name__ == "__main__":
    unittest.main()
