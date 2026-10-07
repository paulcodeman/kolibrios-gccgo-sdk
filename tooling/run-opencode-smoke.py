#!/usr/bin/env python3
"""Boot a temporary KolibriOS image and run a test from a temporary FAT disk."""

import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import shutil
import socket
import struct
import subprocess
import tempfile
import time


ROOT = Path(__file__).resolve().parent.parent


def executable_container(path):
    with path.open('rb') as executable:
        header = executable.read(12)
    if len(header) != 12:
        return False
    if header[:8] == b'MENUET01':
        return path.stat().st_size >= 36
    if header[:4] == b'KPCK':
        size, method = struct.unpack('<II', header[4:])
        # Upstream kpack: LZMA, optionally CALL/JMP transform 1 or 2.
        # The kernel validates the unpacked application header when loading.
        return 36 <= size < 0x80000000 and method in (1, 0x41, 0x81) and path.stat().st_size > 12
    return False


def main():
    # QEMU's RTC defaults to UTC. FAT has no timezone field; stage fixture
    # timestamps in that same clock domain for original edit conflict checks.
    os.environ['TZ'] = 'UTC'
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--image', type=Path, required=True, help='original KolibriOS floppy image (read only)')
    parser.add_argument('--boot-disk-image', type=Path,
                        help='boot a temporary copy of a complete FAT32 disk image, staging the test on its first partition')
    parser.add_argument('--cdrom-image', type=Path,
                        help='attach a read-only ISO on the secondary IDE channel')
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--binary-relative-path', default='SMOKE.KEX',
                        help='native installation path inside the temporary FAT volume')
    parser.add_argument('--arguments', default='', help='loader command line passed to the executable')
    parser.add_argument('--environment-file', type=Path,
                        help='literal KEY=VALUE sidecar staged as SMOKE.KEX.env before application initialization')
    parser.add_argument('--service-binary', type=Path,
                        help='optional native test service started before the application; must create SERVICE.READY')
    parser.add_argument('--collect-file', action='append', default=[],
                        help='relative FAT path copied to PREFIX-files after QEMU exits; may be repeated')
    parser.add_argument('--stage-file', action='append', default=[], metavar='SOURCE=TARGET',
                        help='copy a fixture file to a relative path on the temporary FAT volume')
    parser.add_argument('--launcher', type=Path, default=ROOT / 'apps/tests/opencode/test-launcher/opencode-test-launcher.kex')
    parser.add_argument('--prefix', default='http-smoke')
    parser.add_argument('--debug-board', action='store_true')
    parser.add_argument('--system-tmp', action='store_true',
                        help='start the original TMPDISK utility with a 32 MiB /tmp0/1 RAM disk')
    parser.add_argument('--expect-marker',
                        help='require a marker in the debug-port log; use a kernel built with --debug-output')
    parser.add_argument('--kernel', type=Path, help='optional kernel for the temporary boot image')
    parser.add_argument('--console-object', type=Path, help='optional native CONSOLE.OBJ for the temporary boot image')
    parser.add_argument('--input-script', type=Path,
                        help='JSON timed events with keys or QMP input-send-event mouse events')
    parser.add_argument('--timeout', type=int, default=36, help='capture duration in seconds (default: 36)')
    parser.add_argument('--memory', type=int, default=128, help='guest RAM in MiB (default: 128)')
    parser.add_argument('--cpu', default='max', help='QEMU CPU model; max exposes native CPU random instructions')
    parser.add_argument('--disk-size-mib', type=int, default=64, help='temporary FAT16 disk size in MiB (default: 64)')
    parser.add_argument('--user-network', action='store_true',
                        help='attach a temporary QEMU RTL8139 with user networking; launcher sets guest 10.0.2.15/24')
    parser.add_argument('--network-capture', action='store_true',
                        help='save the temporary guest network as PREFIX-network.pcap')
    args = parser.parse_args()
    binary_path = PurePosixPath(args.binary_relative_path)
    if (not args.binary_relative_path or '\0' in args.binary_relative_path or
            '\\' in args.binary_relative_path or ':' in args.binary_relative_path or
            binary_path.is_absolute() or '..' in binary_path.parts or str(binary_path) == '.'):
        parser.error('binary-relative-path must be a relative FAT filename')
    if args.network_capture and not args.user_network:
        parser.error('--network-capture requires --user-network')
    if not args.prefix or Path(args.prefix).name != args.prefix or args.prefix in ('.', '..'):
        parser.error('prefix must be a single artifact filename component')
    if args.timeout <= 0:
        parser.error('timeout must be positive')
    if args.memory <= 0 or not 16 <= args.disk_size_mib <= 2048:
        parser.error('memory must be positive and FAT16 disk size must be 16..2048 MiB')
    partition_sectors = args.disk_size_mib * 2048
    if '\0' in args.arguments or len(args.arguments.encode('utf-8')) >= 1024:
        parser.error('arguments must fit the 1024-byte NUL-terminated loader buffer')
    input_events = json.loads(args.input_script.read_text()) if args.input_script else []
    staged_files=[]
    for item in args.stage_file:
        source, separator, target=item.partition('=')
        path=PurePosixPath(target)
        if (not separator or not Path(source).is_file() or not target or '\0' in target or '\\' in target
                or ':' in target or path.is_absolute() or '..' in path.parts or str(path)=='.'):
            parser.error('staged files require SOURCE=relative/FAT/path')
        staged_files.append((Path(source),path))
    for name in args.collect_file:
        path = PurePosixPath(name)
        if not name or '\0' in name or '\\' in name or path.is_absolute() or '..' in path.parts or str(path) == '.':
            parser.error('collected files must be relative paths inside the temporary FAT volume')
    if not isinstance(input_events, list):
        parser.error('input script must be a JSON list')
    for event in input_events:
        if (not isinstance(event, dict) or not isinstance(event.get('at'), (int, float))
                or not 0 <= event['at'] < args.timeout):
            parser.error('input events require a time inside the capture duration')
        keys, mouse = event.get('keys'), event.get('mouse')
        if keys is not None:
            if mouse is not None or not isinstance(keys, list) or not keys or not all(isinstance(key, str) and key for key in keys):
                parser.error('key input requires a nonempty key list and no mouse events')
        elif not isinstance(mouse, list) or not mouse:
            parser.error('input events require keys or a nonempty QMP mouse event list')
        else:
            for item in mouse:
                if not isinstance(item, dict) or item.get('type') not in ('rel', 'btn') or not isinstance(item.get('data'), dict):
                    parser.error('mouse input requires QMP rel/btn data')
                data = item['data']
                if item['type'] == 'rel':
                    if data.get('axis') not in ('x', 'y') or not isinstance(data.get('value'), int):
                        parser.error('relative mouse event requires axis x/y and integer value')
                elif data.get('button') not in ('left', 'right', 'middle', 'wheel-up', 'wheel-down', 'wheel-left', 'wheel-right') or not isinstance(data.get('down'), bool):
                    parser.error('mouse button event requires a supported button and boolean down')
    input_events.sort(key=lambda event: event['at'])
    if not executable_container(args.binary):
        parser.error('binary is not a finished KolibriOS executable or KPACK container; wait for its build to complete')
    if args.service_binary:
        if not executable_container(args.service_binary):
            parser.error('service binary is not a finished KolibriOS executable or KPACK container')
    artifacts = ROOT / '.build-cache/opencode'
    artifacts.mkdir(parents=True, exist_ok=True)
    run_record = {
        'binary_sha256': hashlib.sha256(args.binary.read_bytes()).hexdigest(),
        'image_sha256': hashlib.sha256(args.image.read_bytes()).hexdigest(),
        'boot_disk_sha256': hashlib.sha256(args.boot_disk_image.read_bytes()).hexdigest() if args.boot_disk_image else None,
        'cdrom_sha256': hashlib.sha256(args.cdrom_image.read_bytes()).hexdigest() if args.cdrom_image else None,
        'kernel_sha256': hashlib.sha256(args.kernel.read_bytes()).hexdigest() if args.kernel else None,
        'console_sha256': hashlib.sha256(args.console_object.read_bytes()).hexdigest() if args.console_object else None,
        'expected_marker': args.expect_marker, 'memory_mib': args.memory,
        'status': 'started',
    }
    run_record_path = artifacts / (args.prefix + '-run.json')
    run_record_path.write_text(json.dumps(run_record, indent=2) + '\n')
    with tempfile.TemporaryDirectory(prefix='opencode-qemu-') as temporary:
        directory = Path(temporary)
        floppy = directory / 'boot.img'
        shutil.copyfile(args.image, floppy)
        # Prune only the temporary copy; retain boot files, libraries and drivers.
        for folder in ('3D', 'DEMOS', 'GAMES', 'MEDIA'):
            subprocess.run(['mdeltree', '-i', str(floppy), '::/' + folder],
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if args.kernel:
            subprocess.run(['mcopy', '-o', '-i', str(floppy), str(args.kernel), '::/KERNEL.MNT'], check=True)
        if args.console_object:
            subprocess.run(['mcopy', '-o', '-i', str(floppy), str(args.console_object), '::/LIB/CONSOLE.OBJ'], check=True)
        # Qualify the actual image components even when the caller supplies no
        # replacement kernel or console. The source image remains read only.
        for field, filename in (('kernel_sha256', 'KERNEL.MNT'),
                                ('console_sha256', 'LIB/CONSOLE.OBJ')):
            component = directory / Path(filename).name
            subprocess.run(['mcopy', '-i', str(floppy), '::/' + filename,
                            str(component)], check=True)
            run_record[field] = hashlib.sha256(component.read_bytes()).hexdigest()
        run_record_path.write_text(json.dumps(run_record, indent=2) + '\n')
        subprocess.run(['mcopy', '-o', '-i', str(floppy), str(args.launcher), '::/@HA'], check=True)
        autorun = directory / 'autorun.dat'
        startup = '/SYS/ESKIN "" 0\n/SYS/@TASKBAR "" 0\n'
        if args.debug_board:
            startup += '/SYS/DEVELOP/BOARD "" 0\n'
        if args.system_tmp:
            startup += '/SYS/TMPDISK A0S32 -1\n'
        if args.user_network:
            # Original NETCFG's A mode detects PCI NICs, loads and attaches
            # their driver, then exits. The launcher configures guest IPv4.
            startup += '/SYS/NETWORK/NETCFG "A" 0\n'
        startup += '/SYS/@HA "" 0\n'
        autorun.write_text(startup)
        subprocess.run(['mcopy', '-o', '-i', str(floppy), str(autorun), '::/SETTINGS/AUTORUN.DAT'], check=True)
        disk = directory / 'disk.img'
        if args.boot_disk_image:
            if args.kernel or args.console_object:
                parser.error('boot-disk-image must qualify the original image components without overrides')
            shutil.copyfile(args.boot_disk_image, disk)
            partition = str(disk) + '@@1M'
            # The original HDD loader loads its ramdisk from this file. Only
            # the temporary ramdisk receives the test launcher and autorun.
            subprocess.run(['mcopy', '-o', '-i', partition, str(floppy), '::/KOLIBRI.IMG'], check=True)
            component = directory / 'disk-kernel.mnt'
            subprocess.run(['mcopy', '-i', partition, '::/KERNEL.MNT', str(component)], check=True)
            run_record['kernel_sha256'] = hashlib.sha256(component.read_bytes()).hexdigest()
            run_record_path.write_text(json.dumps(run_record, indent=2) + '\n')
        else:
            partition = directory / 'partition.img'
            subprocess.run(['mformat', '-C', '-i', str(partition), '-T', str(partition_sectors),
                            '-h', '16', '-s', '63', '-H', '2048', '-v', 'SDKTEST', '::'], check=True)
        created=set()
        def create_directory(parent):
            if str(parent) == '.' or parent in created:
                return
            target = '::/' + str(parent)
            # mmd can prompt when the complete image already has this folder.
            # Reuse only an existing directory; create missing ones explicitly.
            existing = subprocess.run(['mdir', '-i', str(partition), target + '/'],
                                      stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL,
                                      stderr=subprocess.DEVNULL)
            if existing.returncode:
                subprocess.run(['mmd', '-i', str(partition), target],
                               stdin=subprocess.DEVNULL, check=True)
            created.add(parent)
        for parent in reversed(binary_path.parents):
            create_directory(parent)
        subprocess.run(['mcopy', '-o', '-i', str(partition), str(args.binary), '::/' + str(binary_path)], check=True)
        if str(binary_path) != 'SMOKE.KEX':
            executable_path = directory / 'SMOKE.PATH'
            executable_path.write_text('/hd0/1/' + str(binary_path))
            subprocess.run(['mcopy', '-i', str(partition), str(executable_path), '::/SMOKE.PATH'], check=True)
        for source, target in staged_files:
            for parent in reversed(target.parents):
                create_directory(parent)
            subprocess.run(['mcopy','-o','-i',str(partition),str(source),'::/'+str(target)],check=True)
        if args.user_network:
            network_marker = directory / 'NET.USER'
            network_marker.write_text('QEMU user networking\n')
            subprocess.run(['mcopy', '-i', str(partition), str(network_marker), '::/NET.USER'], check=True)
        if args.environment_file:
            subprocess.run(['mcopy', '-o', '-i', str(partition), str(args.environment_file), '::/' + str(binary_path) + '.env'], check=True)
        if args.service_binary:
            subprocess.run(['mcopy', '-i', str(partition), str(args.service_binary), '::/SERVICE.KEX'], check=True)
        arguments = directory / 'arguments.txt'
        arguments.write_bytes(args.arguments.encode('utf-8'))
        subprocess.run(['mcopy', '-i', str(partition), str(arguments), '::/SMOKE.ARGS'], check=True)
        mbr = bytearray(512)
        # Conventional FAT16 MBR partition at LBA 2048.
        mbr[446:462] = struct.pack('<B3sB3sII', 0, b'\xfe\xff\xff', 6,
                                  b'\xfe\xff\xff', 2048, partition_sectors)
        mbr[510:512] = b'\x55\xaa'
        if not args.boot_disk_image:
            with disk.open('wb') as output, partition.open('rb') as volume:
                output.write(mbr)
                output.seek(2048 * 512)
                shutil.copyfileobj(volume, output)
        monitor = directory / 'monitor.sock'
        with (artifacts / (args.prefix + '-qemu.log')).open('w') as log:
            network_options = (['-netdev', 'user,id=sdknet', '-device', 'rtl8139,netdev=sdknet']
                               if args.user_network else ['-nic', 'none'])
            if args.network_capture:
                network_options += ['-object', 'filter-dump,id=sdkcapture,netdev=sdknet,file=' +
                                    str(artifacts / (args.prefix + '-network.pcap'))]
            boot_options = (['-boot', 'c'] if args.boot_disk_image else
                            ['-drive', 'file=' + str(floppy) + ',format=raw,if=floppy,index=0', '-boot', 'a'])
            if args.cdrom_image:
                boot_options += ['-drive', 'file=' + str(args.cdrom_image) + ',format=raw,if=ide,index=2,media=cdrom,readonly=on']
            process = subprocess.Popen(['qemu-system-i386', '-display', 'none'] + network_options + boot_options + [
                         '-debugcon', 'file:' + str(artifacts / (args.prefix + '-kernel.log')),
                         '-m', str(args.memory), '-cpu', args.cpu, '-qmp', 'unix:' + str(monitor) + ',server=on,wait=off',
                         '-drive', 'file=' + str(disk) + ',format=raw,if=ide,index=0'],
                         stdout=log, stderr=subprocess.STDOUT)
            try:
                for _ in range(100):
                    if monitor.exists():
                        break
                    if process.poll() is not None:
                        raise RuntimeError('QEMU exited before opening monitor')
                    time.sleep(0.1)
                with socket.socket(socket.AF_UNIX) as connection:
                    connection.settimeout(10)
                    connection.connect(str(monitor))
                    stream = connection.makefile('rwb')
                    stream.readline()

                    def execute(command, arguments=None):
                        request = {'execute': command}
                        if arguments:
                            request['arguments'] = arguments
                        stream.write((json.dumps(request) + '\n').encode())
                        stream.flush()
                        while True:
                            response = json.loads(stream.readline())
                            if 'error' in response:
                                raise RuntimeError(str(response))
                            if 'return' in response:
                                return

                    execute('qmp_capabilities')
                    elapsed = 0
                    captures = list(range(12, args.timeout, 12)) + [args.timeout]
                    timeline = sorted(set(captures + [event['at'] for event in input_events]))
                    for instant in timeline:
                        delay = instant - elapsed
                        time.sleep(delay)
                        elapsed = instant
                        for event in input_events:
                            if event['at'] == instant:
                                if 'keys' in event:
                                    execute('send-key', {'keys': [{'type': 'qcode', 'data': key}
                                                                  for key in event['keys']], 'hold-time': 80})
                                else:
                                    execute('input-send-event', {'events': event['mouse']})
                        if instant in captures:
                            screenshot = artifacts / (args.prefix + '-' + str(elapsed) + 's.ppm')
                            execute('screendump', {'filename': str(screenshot)})
                            print('Captured ' + str(screenshot), flush=True)
                    execute('quit')
                    process.wait(timeout=10)
            finally:
                if process.poll() is None:
                    process.terminate()
                    process.wait(timeout=10)
        for name in args.collect_file:
            destination = artifacts / (args.prefix + '-files') / name
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.unlink(missing_ok=True)
            result = subprocess.run(['mcopy', '-o', '-i', str(disk) + '@@1048576',
                                     '::/' + name, str(destination)],
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            if result.returncode:
                print('Not collected ' + name + ': ' + result.stderr.strip(), flush=True)
            else:
                print('Collected ' + str(destination), flush=True)
    messages = (artifacts / (args.prefix + '-kernel.log')).read_text(errors='replace')
    run_record['status'] = 'captured'
    run_record_path.write_text(json.dumps(run_record, indent=2) + '\n')
    for failure in ('runtime panic:', 'K : Page fault', 'K : General protection',
                    'K : Undefined Exception', 'CONSOLE_VTERM_FATAL'):
        if failure in messages:
            raise RuntimeError('native execution failed: ' + failure +
                               '; inspect the debug log and screenshots')
    if args.expect_marker:
        if args.expect_marker not in messages:
            raise RuntimeError('expected marker not found: ' + args.expect_marker +
                               '; inspect the debug log and screenshots')
        print('Verified target marker: ' + args.expect_marker)
        run_record['status'] = 'native_marker_verified'
        run_record_path.write_text(json.dumps(run_record, indent=2) + '\n')
    else:
        print('Capture completed. Inspect the screen for the test PASS or FAIL marker.')


if __name__ == '__main__':
    main()
