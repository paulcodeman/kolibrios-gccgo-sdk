"""Encode relocatable ELF32 into the plain COFF subset used by KolibriOS."""
import struct


def convert(source, target):
    elf = source.read_bytes()
    if elf[:7] != b'\x7fELF\x01\x01\x01':
        raise ValueError('expected little-endian ELF32')
    if struct.unpack_from('<HH', elf, 16) != (1,3):
        raise ValueError('expected relocatable i386 ELF')
    headers = struct.unpack_from('<I', elf, 32)[0]
    stride, count, name_table = struct.unpack_from('<HHH', elf, 46)
    sections = [struct.unpack_from('<10I', elf, headers+i*stride) for i in range(count)]
    names = sections[name_table]
    z = lambda data, p: data[p:data.index(0,p)].decode()
    strings = bytearray(b'\0\0\0\0')
    def coff_name(name):
        encoded = name.encode()
        if len(encoded) <= 8: return encoded.ljust(8,b'\0')
        offset = len(strings); strings.extend(encoded+b'\0')
        return struct.pack('<II',0,offset)
    selected = [i for i,s in enumerate(sections) if s[2]&2]
    section_map = {index:position+1 for position,index in enumerate(selected)}
    symbols = []
    symbol_map = {}
    symtab_index = next(i for i,s in enumerate(sections) if s[1]==2)
    symtab = sections[symtab_index]
    symbol_names = sections[symtab[6]]
    symbol_names = elf[symbol_names[4]:symbol_names[4]+symbol_names[5]]
    for index, position in enumerate(range(symtab[4],symtab[4]+symtab[5],16)):
        name,value,size,info,other,section = struct.unpack_from('<IIIBBH',elf,position)
        if section not in section_map and section!=0xfff1: continue
        symbol_map[index]=len(symbols)
        text=z(symbol_names,name) if name else '.section%d'%section
        symbols.append(coff_name(text)+struct.pack('<IhHBB',value,
            -1 if section==0xfff1 else section_map[section],0,2 if info>>4 else 3,0))
    output = bytearray(20+40*len(selected))
    for index in selected:
        s=sections[index]
        raw = bytearray(elf[s[4]:s[4]+s[5]]) if s[1]!=8 else bytearray()
        relocs=[]
        for relocation in sections:
            if relocation[1]!=9 or relocation[7]!=index: continue
            for p in range(relocation[4],relocation[4]+relocation[5],8):
                address,info=struct.unpack_from('<II',elf,p)
                kind=info&255
                if kind not in (1,2): raise ValueError('unsupported relocation %d'%kind)
                symbol=symbol_map[info>>8]
                if kind==2:
                    value=struct.unpack_from('<I',raw,address)[0]
                    struct.pack_into('<I',raw,address,(value+4)&0xffffffff)
                relocs.append(struct.pack('<IIH',address,symbol,6 if kind==1 else 20))
        if len(relocs)>65535: raise ValueError('too many relocations in COFF section')
        raw_offset=len(output) if raw else 0; output.extend(raw)
        reloc_offset=len(output) if relocs else 0; output.extend(b''.join(relocs))
        alignment=s[8] or 1
        if alignment&(alignment-1) or alignment>4096: raise ValueError('unsupported alignment')
        flags=(alignment.bit_length()<<20)|(0x20 if s[2]&4 else 0x80 if s[1]==8 else 0x40)
        flags|=0x40000000|(0x80000000 if s[2]&1 else 0)
        # Section names are diagnostic only; keep names short and unique.
        name=z(elf,names[4]+s[0]); name=name if len(name)<=8 else '.c%d'%index
        header=name.encode().ljust(8,b'\0')+struct.pack('<IIIIIIHHI',0,0,s[5],
            raw_offset,reloc_offset,0,len(relocs),0,flags)
        position=20+40*(section_map[index]-1);output[position:position+40]=header
    symbol_offset=len(output);output.extend(b''.join(symbols))
    struct.pack_into('<I',strings,0,len(strings));output.extend(strings)
    struct.pack_into('<HHIIIHH',output,0,0x14c,len(selected),0,symbol_offset,len(symbols),0,0x104)
    target.write_bytes(output)
