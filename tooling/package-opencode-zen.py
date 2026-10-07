#!/usr/bin/env python3
"""Create a public-Zen portable package only after full native CLI verification."""
from pathlib import Path
import argparse
import hashlib
import json
import shutil
import zipfile

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base', type=Path, required=True, help='qualified portable installation to extend')
    parser.add_argument('--desktop', action='store_true', help='also copy the archive to the Desktop')
    parser.add_argument('--resume', action='store_true', help='resume packaging the same verified binary without deleting existing files')
    args = parser.parse_args()
    baseline = json.loads((args.base / 'manifest.json').read_text(encoding='utf-8'))
    for name, expected in baseline['files'].items():
        assert digest(args.base / name) == expected, ('base package changed', name)
    binary = ROOT / 'apps/opencode/opencode.kex'
    binary_hash = digest(binary)
    evidence_path = ROOT / '.build-cache/opencode/zen-public/native/validation.json'
    evidence = json.loads(evidence_path.read_text(encoding='utf-8'))
    assert evidence['binary_sha256'] == binary_hash
    assert evidence['native_Enter_remote_reply'] and evidence['clean_exit']
    destination = ROOT / '.build-cache/opencode/release' / ('opencode-zen-portable-' + binary_hash[:12])
    if destination.exists():
        if not args.resume or digest(destination / 'opencode/opencode.kex') != binary_hash:
            raise RuntimeError('release already exists or contains a different binary: ' + str(destination))
    shutil.copytree(args.base, destination, dirs_exist_ok=args.resume)
    shutil.copyfile(binary, destination / 'opencode/opencode.kex')
    resources = ROOT / 'apps/opencode/assets'
    resource_manifest = json.loads((resources / 'manifest.json').read_text(encoding='utf-8'))
    for name, expected in resource_manifest['files'].items():
        assert digest(resources / name) == expected, ('build assets changed', name)
    shutil.copytree(resources, destination / 'opencode/assets', dirs_exist_ok=args.resume)
    shutil.copyfile(ROOT / 'apps/opencode/opencode.kex.env', destination / 'opencode/opencode.kex.env')
    # This is a new release copy. The qualified baseline remains untouched.
    # Remove its superseded resource copies at their explicit legacy paths.
    for name in ('opencode/gosh.kex', 'opencode/certs/ca-bundle.crt'):
        legacy = destination / name
        assert legacy.resolve().is_relative_to(destination.resolve())
        legacy.unlink(missing_ok=True)
    legacy_certificates = destination / 'opencode/certs'
    if legacy_certificates.is_dir() and not any(legacy_certificates.iterdir()):
        legacy_certificates.rmdir()
    (destination / 'README.ru.txt').write_text(
        'Оригинальный Go OpenCode для KolibriOS: бесплатные удалённые модели Zen\n\n'
        'Скопируйте всю папку opencode с ISO на записываемый диск, например /tmp0/1/.\n'
        'Запустите opencode/opencode.kex вместе с его opencode.kex.env.\n'
        'Каталог assets рядом с программой содержит сертификаты и gosh.kex.\n'
        'Локальная модель и Windows-сервер не нужны. Нужны интернет и верная дата/время.\n'
        'На RAM-диске история и настройки исчезнут при перезагрузке.\n\n'
        'При запуске загружаются актуальные списки Zen и models.dev. Включаются только\n'
        'бесплатные, не устаревшие модели с инструментами и Chat Completions.\n'
        'По умолчанию выбрана Space Bunny Free, проверенная без личного API-ключа.\n'
        'Ctrl+O — модели (Zen: ...), стрелки — выбор, Enter — подтвердить, Esc — закрыть.\n'
        'Ctrl+K — команды; Ctrl+N — новая сессия. Команд через / в старой Go-версии нет.\n'
        'Заголовок Select Local Model относится к штатному совместимому API-клиенту:\n'
        'модели с префиксом Zen: работают удалённо, непосредственно через HTTPS.\n\n'
        'Используется публичный режим public. Некоторые бесплатные модели сервер\n'
        'Zen может отклонять (FreeTierError), ограничивать или удалять. Наличие в\n'
        'каталоге не гарантирует разрешение запроса; отказ сервера не обходится.\n'
        'Модели с платным вводом/выводом исключены. Протокол Responses пока не включён.\n'
        'Бесплатный удалённый сервис получает отправленные сообщения и данные,\n'
        'которые OpenCode использует при работе с проектом.\n\n'
        '139 оригинальных Go-файлов OpenCode сохранены без изменений. В status.go\n'
        'исправлена только ширина нижней панели: уведомления не растягивают её\n'
        'по вертикали. Проверяемая правка находится в sources/opencode-patches/.\n'
        'Адаптеры запуска — в sources/opencode-bootstrap/. Контекст взят из каталога.\n'
        'Для возврата к локальному серверу уберите OPENCODE_ZEN_PUBLIC=1 и добавьте\n'
        'LOCAL_ENDPOINT=http://10.0.2.2:18081/v1 в opencode.kex.env.\n'
        'Системные требования прежние: исправленные ядро и консоль, CPU с RDRAND\n'
        'или RDSEED. Дополнительные изменения ядра/консоли для Zen не требуются.\n\n'
        'qualification.json — проверка этой сборки в настоящей KolibriOS;\n'
        'manifest.json — контрольные суммы. Справка CLI: --help.\n', encoding='utf-8')
    source_directory = destination / 'sources/opencode-bootstrap'
    source_directory.mkdir(parents=True, exist_ok=True)
    for source in (ROOT / 'apps/opencode/bootstrap').glob('*.go'):
        shutil.copyfile(source, source_directory / source.name)
    patch_directory = destination / 'sources/opencode-patches'
    patch_directory.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(ROOT / 'apps/opencode/patches/status-width.patch', patch_directory / 'status-width.patch')
    shutil.copyfile(ROOT / '.build-cache/opencode/application-source-compat.json', patch_directory / 'audit.json')
    (destination / 'docs').mkdir(exist_ok=True)
    shutil.copyfile(ROOT / 'docs/OPENCODE_PORT.md', destination / 'docs/OPENCODE_PORT.md')
    (destination / 'qualification.json').write_text(json.dumps(evidence, ensure_ascii=False, indent=2), encoding='utf-8')
    manifest = {'opencode_revision': baseline['opencode_revision'], 'unchanged_application_files': 139,
                'application_patches': ['internal/tui/components/core/status.go'],
                'binary_sha256': binary_hash, 'public_provider': 'OpenCode Zen',
                'base_binary_sha256': baseline['binary_sha256'], 'files': {}}
    for path in sorted(destination.rglob('*')):
        if path.is_file() and path.name != 'manifest.json':
            manifest['files'][path.relative_to(destination).as_posix()] = digest(path)
    (destination / 'manifest.json').write_text(json.dumps(manifest, indent=2), encoding='utf-8')
    archive_path = destination.with_suffix('.zip')
    with zipfile.ZipFile(archive_path, 'w', zipfile.ZIP_DEFLATED) as archive:
        for path in sorted(destination.rglob('*')):
            if path.is_file():
                archive.write(path, path.relative_to(destination.parent))
    with zipfile.ZipFile(archive_path) as archive:
        assert archive.testzip() is None
        for name in manifest['files']:
            assert hashlib.sha256(archive.read(destination.name + '/' + name)).hexdigest() == manifest['files'][name]
    if args.desktop:
        shutil.copyfile(archive_path, Path.home() / 'Desktop' / archive_path.name)
    print('Verified portable archive: ' + str(archive_path))


if __name__ == '__main__':
    main()
