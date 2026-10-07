#!/usr/bin/env python3
"""Keep original Go sources intact while lowering supported generics for GCC."""

import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('resolver', ROOT / 'tooling/resolve-packages.py')
resolver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(resolver)


def write_embed_dependencies(args, sources):
    if '-c' not in args or '-o' not in args:
        return
    target = args[args.index('-o') + 1]
    output = Path(target).resolve()
    config_arg = next((arg for arg in args if arg.startswith('-fgo-embedcfg=')), None)
    dependency_file = Path(str(output) + '.embed.d')
    if config_arg is None:
        dependency_file.unlink(missing_ok=True)
        return
    config = json.loads(Path(config_arg.split('=', 1)[1]).read_text())
    resources = {Path(path).resolve() for path in config['Files'].values()}
    directories = set()
    roots = {source.parent for source in sources}
    for resource in resources:
        directory = resource.parent
        while True:
            directories.add(directory)
            if directory in roots or directory.parent == directory:
                break
            directory = directory.parent
    if resources:
        # Also watch currently empty directories: adding their first matching
        # file changes that directory, but not its ancestors' timestamps.
        for root in roots:
            for directory, children, _ in os.walk(root, followlinks=False):
                children[:] = [name for name in children if name not in ('.git', '.hg', '.svn', '.bzr')]
                directories.add(Path(directory))
    dependencies = sorted(map(str, resources | directories))
    def quote(path):
        return str(path).replace('$', '$$').replace('\\', '\\\\').replace('#', '\\#').replace(' ', '\\ ').replace(':', '\\:')
    # Empty rules let a removed resource trigger a fresh compiler error, rather
    # than make failing first with "No rule to make target".
    data = quote(target) + ': ' + ' '.join(map(quote, dependencies)) + '\n'
    data += ''.join(quote(path) + ':\n' for path in dependencies)
    dependency_file.write_text(data)


def main():
    args = sys.argv[1:]
    source_indices = [i for i, arg in enumerate(args) if arg.endswith('.go') and Path(arg).is_file()]
    sources = [Path(args[i]).resolve() for i in source_indices]
    includes = [Path(arg[2:]).resolve() for arg in args if arg.startswith('-I') and len(arg) > 2]
    for i, arg in enumerate(args[:-1]):
        if arg == '-I':
            includes.append(Path(args[i + 1]).resolve())
    package = next((arg.split('=', 1)[1] for arg in args if arg.startswith('-fgo-pkgpath=')), 'main')
    imports = set()
    generic = False
    embeds = False
    for source in sources:
        text = source.read_text(encoding='utf-8')
        embeds |= '//go:embed' in text
        # The parser/type checker validates the actual syntax before lowering.
        # Mask comments and string literals to avoid generic examples in docs.
        code = re.sub(r'/\*.*?\*/|//[^\n]*|`[^`]*`|"(?:\\.|[^"\\])*"', ' ', text, flags=re.S)
        generic |= bool(re.search(r'\b(?:func|type)\s+\w+\s*\[', code))
        imports.update(resolver.parse_imports(str(source)))
    generic |= any((root / (path + '.generics.json')).exists() for root in includes for path in imports)
    build = Path(os.environ['GCCGO_BUILD_DIR']).resolve()
    command = [os.environ.get('GCCGO_BASE', 'gccgo-15'), '-B' + str(build / 'gcc') + '/']
    helper = Path(os.environ.get('GCCGO_COMPAT', str(build / 'compat/go2go')))
    if embeds and not any(arg.startswith('-fgo-embedcfg=') for arg in args):
        if not helper.is_file():
            raise RuntimeError('embed compiler helper is missing; run tooling/gccgo/build-compat.py --host-go /path/to/go')
        embed_key = hashlib.sha256('\n'.join(map(str, sources)).encode()).hexdigest()
        embed_config = build / 'embedcfg' / (embed_key + '.json')
        subprocess.run([str(helper), '-embedcfg', str(embed_config)] + list(map(str, sources)), check=True)
        args.append('-fgo-embedcfg=' + str(embed_config))
    if generic and helper.is_file():
        declarations = subprocess.run([str(helper), '-scan-generics'] + list(map(str, sources)),
                                      check=True, capture_output=True, text=True).stdout.strip() == 'true'
        generic = declarations or any((root / (path + '.generics.json')).exists()
                                      for root in includes for path in imports)
    if not generic:
        result = subprocess.run(command + args)
        if result.returncode == 0:
            write_embed_dependencies(args, sources)
        if result.returncode == 0 and '-c' in args and '-o' in args:
            output = Path(args[args.index('-o') + 1])
            if output.name.endswith('.gccgo.go.o'):
                output.with_name(output.name[:-len('.gccgo.go.o')] + '.generics.json').unlink(missing_ok=True)
        return result.returncode
    if not helper.is_file():
        raise RuntimeError('generic compiler helper is missing; run tooling/gccgo/build-compat.py --host-go /path/to/go')
    key = hashlib.sha256()
    for source in sources:
        key.update(str(source).encode())
        key.update(source.read_bytes())
    key.update(str(includes).encode())
    directory = build / 'lowered' / key.hexdigest()
    lower = [str(helper), '-out', str(directory), '-package', package]
    for root in includes:
        lower.extend(['-gccgo-import-root', str(root)])
    lower.extend(map(str, sources))
    subprocess.run(lower, check=True)
    lowered_sources = json.loads((directory / 'compiler-inputs.json').read_text())
    for index, source in zip(source_indices, sources):
        args[index] = lowered_sources[str(source)]
    result = subprocess.run(command + args)
    if result.returncode == 0:
        write_embed_dependencies(args, sources)
    if result.returncode == 0 and '-c' in args and '-o' in args:
        output = Path(args[args.index('-o') + 1])
        if output.name.endswith('.gccgo.go.o'):
            metadata = output.with_name(output.name[:-len('.gccgo.go.o')] + '.generics.json')
            record = {'version': 1, 'package': package, 'sources': list(map(str, sources)),
                      'source_sha256': {str(path): hashlib.sha256(path.read_bytes()).hexdigest() for path in sources}}
            metadata.write_text(json.dumps(record, indent=2) + '\n')
    return result.returncode


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (RuntimeError, subprocess.CalledProcessError) as error:
        print('gccgo compatibility: ' + str(error), file=sys.stderr)
        sys.exit(1)
