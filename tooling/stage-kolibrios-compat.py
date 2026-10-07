#!/usr/bin/env python3
"""Apply shared KolibriOS adapters to an upstream vendor tree for any SDK app."""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_SOURCE = ROOT / 'third_party'


def vendor_versions(vendor):
    versions = {}
    path = vendor / 'modules.txt'
    if path.is_file():
        for line in path.read_text().splitlines():
            fields = line.split()
            if len(fields) >= 3 and fields[0] == '#' and fields[2].startswith('v'):
                versions[fields[1]] = fields[2]
    return versions


def stage(vendor, record_path, source=DEFAULT_SOURCE, modules=None, check=False):
    vendor, record_path, source = Path(vendor), Path(record_path), Path(source)
    manifest_path = source / 'ADAPTERS.json'
    if not manifest_path.is_file():
        manifest_path = source / 'kolibrios-compat/ADAPTERS.json'
    manifest = json.loads(manifest_path.read_text())
    versions = vendor_versions(vendor)
    selected = set(modules) if modules else set(manifest['tested_modules'])
    unknown = selected - manifest['tested_modules'].keys()
    if unknown:
        raise RuntimeError('unknown adapter modules: ' + ', '.join(sorted(unknown)))
    if versions and not modules:
        selected.intersection_update(versions)
    for module in selected:
        actual = versions.get(module)
        expected = manifest['tested_modules'][module]
        if actual is not None and actual != expected:
            raise RuntimeError(f'adapter version mismatch for {module}: expected {expected}, found {actual}')
    previous = json.loads(record_path.read_text()) if record_path.exists() else {}
    records = dict(previous)
    pending = []
    for relative, entry in sorted(manifest['adapters'].items()):
        if entry['module'] not in selected:
            continue
        path = PurePosixPath(relative)
        if path.is_absolute() or '..' in path.parts or '\\' in relative or ':' in relative:
            raise RuntimeError('invalid adapter path: ' + relative)
        origin_path = PurePosixPath(entry.get('origin', relative))
        if origin_path.is_absolute() or '..' in origin_path.parts or '\\' in str(origin_path) or ':' in str(origin_path):
            raise RuntimeError('invalid adapter origin: ' + str(origin_path))
        origin = source / origin_path
        destination = vendor / path
        if not origin.resolve().is_relative_to(source.resolve()) or not destination.resolve().is_relative_to(vendor.resolve()):
            raise RuntimeError('adapter path escapes source or vendor tree: ' + relative)
        data = origin.read_bytes()
        expected = hashlib.sha256(data).hexdigest()
        if expected != entry['sha256']:
            raise RuntimeError('adapter fingerprint mismatch: ' + relative)
        existing = destination.read_bytes() if destination.exists() else None
        if check and existing != data:
            raise RuntimeError('staged adapter differs or is missing: ' + relative)
        if existing is not None and existing != data:
            actual = hashlib.sha256(existing).hexdigest()
            if previous.get(relative) != actual:
                raise RuntimeError('refusing to overwrite a modified compatibility file: ' + relative)
        records[relative] = expected
        pending.append((destination, data, existing))
    # Validate the whole selection before changing any destination file.
    if not check:
        for destination, data, existing in pending:
            destination.parent.mkdir(parents=True, exist_ok=True)
            if existing != data:
                destination.write_bytes(data)
        record_path.parent.mkdir(parents=True, exist_ok=True)
        record_path.write_text(json.dumps(records, indent=2, sort_keys=True) + '\n')
    return len(pending)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--vendor', type=Path, required=True)
    parser.add_argument('--record', type=Path, help='adapter fingerprints; default: VENDOR/kolibrios-compat-record.json')
    parser.add_argument('--source', type=Path, default=DEFAULT_SOURCE)
    parser.add_argument('--module', action='append', help='apply only this module; may be repeated')
    parser.add_argument('--check', action='store_true', help='verify staged files without writing')
    args = parser.parse_args()
    record = args.record or args.vendor / 'kolibrios-compat-record.json'
    try:
        count = stage(args.vendor, record, args.source, args.module, args.check)
    except (RuntimeError, OSError, ValueError) as error:
        parser.exit(1, 'KolibriOS compatibility: ' + str(error) + '\n')
    print(f'{count} shared KolibriOS compatibility files ' + ('verified' if args.check else 'staged'))


if __name__ == '__main__':
    main()
