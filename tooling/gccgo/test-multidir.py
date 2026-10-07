#!/usr/bin/env python3
"""Compile a package with repeated filenames and distinct function-local types."""
import argparse
import hashlib
from pathlib import Path
import subprocess
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--compiler', required=True)
args = parser.parse_args()
with tempfile.TemporaryDirectory(prefix='gccgo-multidir-') as temporary:
    directory = Path(temporary)
    bodies = {
        'main.go': '''package main
// Keep a heap root for the hosted libgo regression executable.
var Root = new(int)
func Id[T any](value T) T { return value }
func main() {
    *Root = 1
    if (&LeftNode{}).Value() != 11 || (&RightNode{}).Value() != 29 { panic("lost method") }
    if Left() == Rght() { panic("merged function-local type identities") }
}
''',
        'left/types.go': '''package main
type LeftNode struct{}
func (*LeftNode) Value() int { return 11 }
func Left() interface{} { type item struct { N int }; return Id(item{1}) }
''',
        'right/types.go': '''package main
type RightNode struct{}
func (*RightNode) Value() int { return 29 }
func Rght() interface{} { type item struct { N int }; return Id(item{1}) }
''',
    }
    # Place the local declarations at the same byte offset to catch basename-
    # based identity collisions independently of the lowered-file collision.
    offset = max(body.index('type item') for body in list(bodies.values())[1:])
    for name in ('left/types.go', 'right/types.go'):
        body = bodies[name]
        bodies[name] = body.replace('type item', ' ' * (offset - body.index('type item')) + 'type item')
    sources = []
    for name, body in bodies.items():
        path = directory / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(body)
        sources.append(path)
    hashes = {path: hashlib.sha256(path.read_bytes()).hexdigest() for path in sources}
    for optimization in ([], ['-O2']):
        executable = directory / 'test'
        subprocess.run([args.compiler, *optimization, '-fno-split-stack', '-fno-pie', '-no-pie',
                        *map(str, sources), '-o', str(executable)], check=True)
        subprocess.run([str(executable)], check=True)
    subprocess.run([args.compiler, '-m32', '-c', '-fno-split-stack',
                    *map(str, sources), '-o', str(directory / 'test-386.o')], check=True)
    assert hashes == {path: hashlib.sha256(path.read_bytes()).hexdigest() for path in sources}
    print('PASS: same-named files retain distinct methods and local types; host execution and 386 compilation')
