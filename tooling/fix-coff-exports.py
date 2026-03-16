#!/usr/bin/env python3

import argparse
import struct
import sys


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Fix COFF EXPORTS symbol section.")
    parser.add_argument("--file", required=True, help="Path to COFF object to patch.")
    parser.add_argument("--symbol", default="EXPORTS", help="Symbol name to fix.")
    parser.add_argument("--section", default=".data", help="Section name for the symbol.")
    parser.add_argument(
        "--no-reloc-fix",
        action="store_false",
        dest="fix_relocs",
        help="Skip COFF relocation type remapping.",
    )
    return parser.parse_args()


def read_cstring(blob: bytes, start: int) -> str:
    end = blob.find(b"\x00", start)
    if end == -1:
        end = len(blob)
    return blob[start:end].decode("latin1")


def read_section_name(entry: bytes, strtab: bytes) -> str:
    raw = entry[:8]
    if raw.startswith(b"/"):
        try:
            offset = int(raw[1:].rstrip(b"\x00").decode("ascii"))
        except ValueError:
            return raw.rstrip(b"\x00").decode("latin1")
        if offset < 4 or offset >= len(strtab):
            return ""
        return read_cstring(strtab, offset)
    return raw.rstrip(b"\x00").decode("latin1")


def read_symbol_name(entry: bytes, strtab: bytes) -> str:
    if entry[:4] == b"\x00\x00\x00\x00":
        offset = struct.unpack_from("<L", entry, 4)[0]
        if offset < 4 or offset >= len(strtab):
            return ""
        return read_cstring(strtab, offset)
    return entry[:8].rstrip(b"\x00").decode("latin1")


def main() -> int:
    args = parse_args()
    with open(args.file, "rb") as handle:
        data = bytearray(handle.read())

    if len(data) < 20:
        print("error: file too small to be COFF", file=sys.stderr)
        return 1

    machine, nscns, _time, symptr, nsyms, opthdr, _flags = struct.unpack_from(
        "<HHLLLHH", data, 0
    )
    if symptr == 0 or nsyms == 0:
        print("error: COFF symbol table not found", file=sys.stderr)
        return 1

    sect_off = 20 + opthdr
    sect_size = 40 * nscns
    if sect_off + sect_size > len(data):
        print("error: section headers truncated", file=sys.stderr)
        return 1

    strtab_off = symptr + nsyms * 18
    if strtab_off + 4 > len(data):
        print("error: string table missing", file=sys.stderr)
        return 1
    strtab_len = struct.unpack_from("<L", data, strtab_off)[0]
    strtab = bytes(data[strtab_off : strtab_off + strtab_len])

    target_section_index = None
    for idx in range(nscns):
        entry = bytes(data[sect_off + idx * 40 : sect_off + (idx + 1) * 40])
        name = read_section_name(entry, strtab)
        if name == args.section:
            target_section_index = idx + 1
            break

    if target_section_index is None:
        print(f"error: section {args.section} not found", file=sys.stderr)
        return 1

    patched = False
    sym_off = symptr
    sym_index = 0
    while sym_index < nsyms:
        entry = bytes(data[sym_off : sym_off + 18])
        name = read_symbol_name(entry, strtab)
        aux_count = entry[17]
        if name == args.symbol:
            struct.pack_into("<h", data, sym_off + 12, target_section_index)
            patched = True
        sym_index += 1 + aux_count
        sym_off += 18 * (1 + aux_count)

    if not patched:
        print(f"error: symbol {args.symbol} not found", file=sys.stderr)
        return 1

    if args.fix_relocs:
        reloc_map = {1: 6, 2: 20}
        for idx in range(nscns):
            entry_off = sect_off + idx * 40
            entry = data[entry_off : entry_off + 40]
            if len(entry) < 40:
                print("error: section headers truncated", file=sys.stderr)
                return 1
            _name = entry[:8]
            _vsz, _vaddr, _size, _ptr_raw, ptr_reloc, _ptr_line, nreloc, _nline, _ch = (
                struct.unpack_from("<IIIIIIHHI", entry, 8)
            )
            if nreloc == 0 or ptr_reloc == 0:
                continue
            end_reloc = ptr_reloc + nreloc * 10
            if end_reloc > len(data):
                print("error: relocation table truncated", file=sys.stderr)
                return 1
            for rel_idx in range(nreloc):
                rel_off = ptr_reloc + rel_idx * 10
                rtype = struct.unpack_from("<H", data, rel_off + 8)[0]
                new_type = reloc_map.get(rtype)
                if new_type is not None and new_type != rtype:
                    struct.pack_into("<H", data, rel_off + 8, new_type)

    with open(args.file, "wb") as handle:
        handle.write(data)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
