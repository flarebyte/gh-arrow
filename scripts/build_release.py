"""Build with flarebyte and stage only checksum-listed release assets."""

from pathlib import Path
import shutil
import subprocess
import tempfile

from release_preflight import check_artifacts, release_metadata


def stage(source, destination, checksum_name):
    check_artifacts(source.parent, {
        'artifact_dir': source.name, 'checksum_name': checksum_name,
    })
    # Refuse an existing destination rather than mixing stale and fresh assets.
    destination.mkdir(parents=True, exist_ok=False)
    for path in source.iterdir():
        shutil.copy2(path, destination / path.name)


def main():
    root = Path.cwd()
    metadata = release_metadata(root)
    destination = root / metadata['artifact_dir']
    # Flarebyte leaves .tmp in its output. Copy only manifest-listed assets
    # into a separate directory before strict validation and staging.
    with tempfile.TemporaryDirectory() as directory:
        temporary = Path(directory).resolve()
        output = temporary / 'output'
        subprocess.run(['gh', 'flarebyte', 'build', '--output-dir', str(output)], check=True)
        clean = temporary / 'assets'
        clean.mkdir()
        manifest = output / metadata['checksum_name']
        shutil.copy2(manifest, clean / manifest.name)
        for line in manifest.read_text().splitlines():
            _, name = line.split('  ', 1)
            if Path(name).name != name:
                raise ValueError('Expected flat release asset filenames')
            shutil.copy2(output / name, clean / name, follow_symlinks=False)
        stage(clean, destination, metadata['checksum_name'])
    print(f'Release assets verified and staged in {destination}')


if __name__ == '__main__':
    main()
