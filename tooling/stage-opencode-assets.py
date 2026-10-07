#!/usr/bin/env python3
"""Stage OpenCode's portable resources from their canonical SDK sources."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil

ROOT = Path(__file__).resolve().parents[1]


def stage(destination):
    certificates = ROOT / 'third_party/ca-certificates'
    provenance = json.loads((certificates / 'UPSTREAM.json').read_text(encoding='utf-8'))
    bundle = certificates / 'ca-bundle.crt'
    if hashlib.sha256(bundle.read_bytes()).hexdigest() != provenance['files']['ca-bundle.crt']['sha256']:
        raise RuntimeError('CA bundle differs from its upstream provenance')
    shell = ROOT / 'apps/tools/gosh/gosh.kex'
    if not shell.is_file():
        raise RuntimeError('Build apps/tools/gosh before staging assets')
    files = {}

    def copy(source, relative):
        data = source.read_bytes()
        target = destination / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        if not target.is_file() or target.read_bytes() != data:
            shutil.copyfile(source, target)
        files[Path(relative).as_posix()] = hashlib.sha256(data).hexdigest()

    copy(shell, 'gosh.kex')
    copy(bundle, 'ca-bundle.crt')
    copy(certificates / 'UPSTREAM.json', 'licenses/ca-provenance.json')
    copy(certificates / 'MPL-2.0.txt', 'licenses/certificates-MPL-2.0.txt')
    copy(ROOT / 'LICENSE', 'licenses/SDK-MIT.txt')
    copy(ROOT / 'THIRD_PARTY_LICENSES.md', 'licenses/THIRD_PARTY_LICENSES.md')
    roots = {
        'vendor': ROOT / '.build-cache/opencode/vendor',
        'opencode': ROOT / '.build-cache/opencode/third_party/github.com/opencode-ai/opencode',
        'stdlib': ROOT / 'stdlib',
        'shell': ROOT / 'third_party/mvdan.cc/sh/v3',
        'watcher': ROOT / 'third_party/github.com/radovskyb/watcher',
        'adapters': ROOT / 'third_party/kolibrios-compat/licenses',
        'font': ROOT / 'third_party/unifont',
    }
    names = ('LICENSE', 'COPYING', 'COPYRIGHT', 'NOTICE', 'OFL', 'MPL')
    for label, source in roots.items():
        for path in sorted(source.rglob('*')):
            if path.is_file() and path.name.upper().startswith(names):
                copy(path, Path('licenses') / label / path.relative_to(source))
    manifest = {'files': files, 'ca_provenance': provenance,
                'shell_source': 'apps/tools/gosh/gosh.kex'}
    manifest_path = destination / 'manifest.json'
    text = json.dumps(manifest, indent=2) + '\n'
    if not manifest_path.is_file() or manifest_path.read_text(encoding='utf-8') != text:
        manifest_path.write_text(text, encoding='utf-8')
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--destination', type=Path, default=ROOT / 'apps/opencode/assets')
    args = parser.parse_args()
    manifest = stage(args.destination)
    print(f"Staged {len(manifest['files'])} OpenCode resource files in {args.destination}")


if __name__ == '__main__':
    main()
