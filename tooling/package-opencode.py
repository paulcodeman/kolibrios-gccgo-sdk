#!/usr/bin/env python3
"""Bundle a qualified original Go OpenCode CLI and its native dependencies."""
from pathlib import Path
import argparse
import hashlib
import json
import shutil
import subprocess
import zipfile

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / '.build-cache/opencode'
REVISION = '73ee493265acf15fcd8caab2bc8cd3bd375b63cb'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--local-endpoint', help='include a ready local-model endpoint in the portable sidecar')
    args = parser.parse_args()
    if args.local_endpoint and any(character in args.local_endpoint for character in '\r\n\0'):
        parser.error('local-endpoint must be a single environment value')
    for script in ('verify-stdlib-upstream.py', 'verify-native-dependencies.py'):
        subprocess.run(['python3', str(ROOT / 'tooling' / script)], check=True)
    subprocess.run(['python3', str(ROOT / 'tooling/opencode-port.py'), 'audit'], check=True)
    binary = ROOT / 'apps/opencode/opencode.kex'
    kernel = CACHE / 'kernel-fat-case-packed.mnt'
    console = CACHE / 'console-vterm.obj'
    expected = {'binary_sha256': digest(binary), 'kernel_sha256': digest(kernel),
                'console_sha256': digest(console)}
    prefixes = ('original-cli-real-ai-verified-tls-native',
                'original-cli-tui-verified-tls-native', 'original-cli-native-tools',
                'original-cli-native-mcp-env', 'original-cli-native-shell-search-case',
                'original-cli-native-lsp-ready')
    for additional in ('original-cli-installed-package-tls', 'original-cli-tui-verified-tls-repeat-qualified'):
        if (CACHE / (additional + '-files/validation.json')).is_file():
            prefixes += (additional,)
    evidence = {}
    for prefix in prefixes:
        report = json.loads((CACHE / (prefix + '-files/validation.json')).read_text())
        assert report['original_CLI_completed'], prefix
        for key, value in expected.items():
            assert report['execution'][key] == value, (prefix, key, 'requalify current build')
        evidence[prefix] = report
    certificates = ROOT / 'third_party/ca-certificates'
    provenance = json.loads((certificates / 'UPSTREAM.json').read_text())
    assert digest(certificates / 'ca-bundle.crt') == provenance['files']['ca-bundle.crt']['sha256']
    destination = CACHE / 'release' / ('opencode-kolibrios-' + expected['binary_sha256'][:12])
    destination.mkdir(parents=True, exist_ok=True)

    def copy(source, relative):
        target = destination / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source, target)

    copy(binary, 'opencode/opencode.kex')
    copy(ROOT / 'apps/tools/gosh/gosh.kex', 'opencode/gosh.kex')
    copy(certificates / 'ca-bundle.crt', 'opencode/certs/ca-bundle.crt')
    copy(kernel, 'system/KERNEL.MNT')
    copy(console, 'system/LIB/CONSOLE.OBJ')
    for directory in ('opencode/.config/opencode',):
        (destination / directory).mkdir(parents=True, exist_ok=True)
    environment = (
        'KOLIBRI_WORKING_DIRECTORY=.\nHOME=.\nUSER=kolibri\nXDG_CONFIG_HOME=./.config\n'
        'SHELL=./gosh.kex\n'
        'PATH=.:/sys:/sys/develop\n'
        'SSL_CERT_FILE=./certs/ca-bundle.crt\n')
    if args.local_endpoint:
        environment += 'LOCAL_ENDPOINT=' + args.local_endpoint + '\n'
    (destination / 'opencode/opencode.kex.env').write_text(environment)
    copy(ROOT / 'tooling/start-opencode-local-ai.ps1', 'Start-LocalAI.ps1')
    provider_instructions = (
        '3. LOCAL_ENDPOINT уже задан: ' + args.local_endpoint + '\n'
        '   Для профиля QEMU 10.0.2.2:18081 запустите Start-LocalAI.ps1 в Windows PowerShell.\n'
        '   Скрипт использует Qwen и llama-server из кэша установленного SDK через WSL.\n'
        '   Если SDK перемещён, передайте -SdkRoot с его новым путём.\n'
        '   Сама модель работает на хосте и в архив не включена.\n'
        if args.local_endpoint else
        '3. Настройте провайдера в opencode.kex.env или штатном .opencode.json.\n'
        '   Для собственного сервера см. local-ai.env.example. Ключи не входят в пакет.\n'
    )
    (destination / 'opencode/local-ai.env.example').write_text(
        '# Add your own accessible OpenAI-compatible endpoint to opencode.kex.env.\n'
        '# This example does not configure a running service or supply credentials.\n'
        'LOCAL_ENDPOINT=https://your-model-server.example/v1\n')
    (destination / 'README.ru.txt').write_text(
        'Оригинальный Go OpenCode CLI для KolibriOS\n\n'
        'Версия исходников: ' + REVISION + '\n'
        'Все 140 Go-файлов OpenCode сохранены без изменений.\n\n'
        'Установка:\n'
        '1. Используйте совместимые KERNEL.MNT и LIB/CONSOLE.OBJ из system/.\n'
        '   Перед заменой сохраните копии своих файлов и перезагрузите KolibriOS.\n'
        '2. Скопируйте каталог opencode целиком в любую папку записываемого диска.\n'
        '   KOLIBRI_WORKING_DIRECTORY=. привязывает относительные пути к папке программы.\n'
        '   Временные файлы используют системный RAM-диск /tmp0/1.\n'
        + provider_instructions +
        '4. Запустите /hd0/1/opencode/opencode.kex с параметрами -c /hd0/1/ВАШ_ПРОЕКТ.\n'
        '   Разовый запрос: -c /hd0/1/ВАШ_ПРОЕКТ -p "Привет" -f json -q.\n'
        '   Справка: --help. Без параметров откроется штатный интерфейс настроенного провайдера.\n'
        '   ISO доступен только для чтения: для истории и SQLite нужен записываемый диск.\n\n'
        'Проверено в QEMU с 512 МБ RAM. Для HTTPS нужны сеть и верные дата/время.\n'
        'Криптографическая случайность требует CPU с RDSEED или RDRAND.\n'
        'Пакет содержит публичные корневые сертификаты Mozilla; тестовые CA отсутствуют.\n'
        'gosh использует оригинальный mvdan/sh и часть синтаксиса Bash; это не GNU Bash.\n'
        'Git, rg, внешние редакторы и LSP/MCP-серверы устанавливаются отдельно и должны\n'
        'иметь нативные версии. Наблюдение за файлами использует опрос раз в секунду.\n'
        'Проверка ИИ использовала настоящую модель Qwen на хосте: сама модель в пакет\n'
        'не включена. Платные облачные провайдеры требуют ваших ключей и отдельно\n'
        'проверяются с вашим аккаунтом.\n\n'
        'qualification.json содержит результаты нативных проверок этой сборки.\n'
        'manifest.json содержит хэши файлов; licenses/ — лицензии зависимостей.\n'
        'Исходники совместимого ядра и воспроизводимые адаптеры находятся в sources/.\n',
        encoding='utf-8')
    license_names = ('LICENSE', 'COPYING', 'COPYRIGHT', 'NOTICE', 'OFL', 'MPL')
    roots = {'vendor': CACHE / 'vendor', 'opencode': CACHE / 'third_party/github.com/opencode-ai/opencode',
             'sdk-third-party': ROOT / 'third_party', 'stdlib': ROOT / 'stdlib'}
    for label, source in roots.items():
        for path in source.rglob('*'):
            if path.is_file() and any(path.name.upper().startswith(name) for name in license_names):
                copy(path, Path('licenses') / label / path.relative_to(source))
    copy(ROOT / 'LICENSE', 'licenses/SDK-MIT.txt')
    copy(ROOT / 'THIRD_PARTY_LICENSES.md', 'licenses/THIRD_PARTY_LICENSES.md')
    for name in ('OPENCODE_PORT.md', 'OPENCODE_STDLIB.md', 'BUILD.md'):
        copy(ROOT / 'docs' / name, Path('docs') / name)
    gcc_major = subprocess.check_output(['gcc', '-dumpfullversion'], text=True).strip().split('.')[0]
    copy(Path('/usr/share/doc') / ('gcc-' + gcc_major + '-base/copyright'), 'licenses/GCC-copyright-and-runtime-exception.txt')
    copy(Path('/usr/share/common-licenses/GPL-3'), 'licenses/GPL-3.txt')
    kernel_source = ROOT.parent / 'kernel/trunk'
    shutil.copytree(kernel_source, destination / 'sources/kernel/trunk', dirs_exist_ok=True,
                    ignore=shutil.ignore_patterns('*.mnt', '*.fas', '*.o', '*.obj', '.git'))
    console_source = ROOT.parent / 'programs/develop/libraries/console_coff'
    shutil.copytree(console_source, destination / 'sources/programs/develop/libraries/console_coff',
                    dirs_exist_ok=True, ignore=shutil.ignore_patterns('*.obj', '*.o', '.git'))
    copy(ROOT.parent / 'programs/struct.inc', 'sources/programs/struct.inc')
    for directory in ('libvterm', 'kolibri-printf', 'kolibri-newlib-string'):
        shutil.copytree(ROOT / 'third_party' / directory,
                        destination / 'sources/third_party' / directory, dirs_exist_ok=True)
    shutil.copytree(ROOT / 'tooling/console', destination / 'sources/tooling/console', dirs_exist_ok=True)
    for name in ('build-kolibri-socket-kernel.py', 'build-kolibri-console.py', 'build_console_vterm.py',
                 'elf_to_kolibri_coff.py'):
        copy(ROOT / 'tooling' / name, Path('sources/tooling') / name)
    (destination / 'sources/BUILDING.txt').write_text(
        'From sources/, with FASM, Python 3, GCC i686 and binutils installed:\n'
        'python3 tooling/build-kolibri-socket-kernel.py --source kernel/trunk --output kernel.mnt '
        '--exclusive-create --local-streams --fat-name-case --debug-output --symbols\n'
        'Compress kernel.mnt with the original KolibriOS kerpack utility.\n'
        'python3 tooling/build-kolibri-console.py --source programs/develop/libraries/console_coff '
        '--output CONSOLE.OBJ --signal-events --utf8-output --vterm\n'
        'The original OpenCode/compiler/SDK build is documented in the SDK repository\n'
        'docs/OPENCODE_PORT.md; tooling/opencode-port.py preserves the pinned originals.\n')
    (destination / 'qualification.json').write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + '\n')
    manifest = {'opencode_revision': REVISION, 'unchanged_application_files': 140,
                'ca_provenance': provenance, 'files': {}}
    for path in sorted(destination.rglob('*')):
        if path.is_file() and path.name != 'manifest.json':
            manifest['files'][str(path.relative_to(destination)).replace('\\', '/')] = digest(path)
    (destination / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    archive_path = destination.with_suffix('.zip')
    with zipfile.ZipFile(archive_path, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
        for path in sorted(destination.rglob('*')):
            archive.write(path, path.relative_to(destination.parent))
    print('Qualified native package: ' + str(archive_path))


if __name__ == '__main__':
    main()
