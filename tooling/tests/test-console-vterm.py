from pathlib import Path
import subprocess,ctypes,struct,gzip
root=Path(__file__).resolve().parents[2]
vterm=root/'third_party/libvterm/libvterm-0.3.3'
output=root/'.build-cache/console-vterm-host.so'
command=['gcc','-shared','-fPIC','-O2','-std=c99','-Wall','-I'+str(vterm/'include'),'-I'+str(vterm/'src'),'-I'+str(root/'tooling/console'),str(root/'tooling/console/vterm_backend.c')]+[str(p) for p in sorted((vterm/'src').glob('*.c'))]+['-o',str(output)]
subprocess.run(command,check=True)
lib=ctypes.CDLL(str(output))
lib.sdk_vterm_encode_cp866.argtypes=[ctypes.c_uint,ctypes.c_void_p]
for code in range(256):
    encoded=ctypes.create_string_buffer(6)
    length=lib.sdk_vterm_encode_cp866(code,encoded)
    assert encoded.raw[:length]==bytes([code]).decode('cp866').encode('utf-8')
lib.sdk_vterm_start.argtypes=[ctypes.c_int,ctypes.c_int];lib.sdk_vterm_start.restype=ctypes.c_int
lib.sdk_vterm_write.argtypes=[ctypes.c_char_p,ctypes.c_size_t];lib.sdk_vterm_write.restype=ctypes.c_size_t
lib.sdk_vterm_cell.argtypes=[ctypes.c_int,ctypes.c_int,ctypes.POINTER(ctypes.c_uint32)]
lib.sdk_vterm_cursor.argtypes=[ctypes.POINTER(ctypes.c_uint32),ctypes.POINTER(ctypes.c_uint32)]
lib.sdk_vterm_read_response.argtypes=[ctypes.c_char_p,ctypes.c_size_t];lib.sdk_vterm_read_response.restype=ctypes.c_size_t
assert lib.sdk_vterm_start(4,12)==1
write=lambda data: lib.sdk_vterm_write(data,len(data))
def cell(x,y):
    out=(ctypes.c_uint32*4)();assert lib.sdk_vterm_cell(x,y,out)==1;return list(out)
def cursor():
    x=ctypes.c_uint32();y=ctypes.c_uint32();lib.sdk_vterm_cursor(ctypes.byref(x),ctypes.byref(y));return(x.value,y.value)
write(b'\xd0');assert cursor()==(0,0)
write(b'\x96');assert cursor()==(1,0) and cell(0,0)[0]==0x416
write(b'\x1b[38;5;196m\x1b[48;5;22mX');assert cell(1,0)[1:3]==[0xff0000,0x005f00]
write(b'\x1b[2;6H\x1b[2D');assert cursor()==(3,1)
write(b'\x1b[6n');response=ctypes.create_string_buffer(128);n=lib.sdk_vterm_read_response(response,128);assert response.raw[:n]==b'\x1b[2;4R'
write(b'\x1b[?1049hALT\x1b[?1049l');assert cell(0,0)[0]==0x416 and cursor()==(3,1)
write('😀'.encode());assert cursor()==(5,1) and cell(3,1)[3]==2
# Scrollback is preserved outside the alternate screen.
lib.sdk_vterm_layout.argtypes=[ctypes.POINTER(ctypes.c_uint32)]*3
lib.sdk_vterm_start(3,12)
write(b'ONE\r\nTWO\r\nTHREE\r\nFOUR')
height=ctypes.c_uint32();offset=ctypes.c_uint32();row=ctypes.c_uint32()
lib.sdk_vterm_layout(ctypes.byref(height),ctypes.byref(offset),ctypes.byref(row))
assert (height.value,offset.value,row.value)==(4,1,3)
write(b'\x1b[?1049h')
lib.sdk_vterm_layout(ctypes.byref(height),ctypes.byref(offset),ctypes.byref(row))
assert (height.value,offset.value,row.value)==(3,0,2)
write(b'\x1b[?1049l')
lib.sdk_vterm_layout(ctypes.byref(height),ctypes.byref(offset),ctypes.byref(row))
assert (height.value,offset.value,row.value)==(4,1,3)
lib.sdk_vterm_stop()
font_path=root/'.build-cache/console-unifont.kbf'
subprocess.run(['python3',str(root/'tooling/build-unifont.py'),str(font_path)],check=True)
font_data=font_path.read_bytes()
font_buffer=ctypes.create_string_buffer(font_data)
lib.sdk_vterm_set_font.argtypes=[ctypes.c_void_p,ctypes.c_size_t]
lib.sdk_vterm_glyph_width.argtypes=[ctypes.c_uint32]
assert lib.sdk_vterm_set_font(font_buffer,len(font_data))==1
assert lib.sdk_vterm_set_font(font_buffer,7)==0
bad=ctypes.create_string_buffer(b'KBF1'+struct.pack('<I',0xffffffff))
assert lib.sdk_vterm_set_font(bad,8)==0
assert lib.sdk_vterm_glyph_width(0x1f600)==16
assert lib.sdk_vterm_glyph_width(0x4e2d)==16
assert lib.sdk_vterm_glyph_width(0x256d)==8
# Compare the actual framebuffer to the unchanged upstream rows, including
# the second cell of wide glyphs and an overlaid combining accent.
source={int(line.split(':')[0],16):bytes.fromhex(line.split(':')[1])
        for line in gzip.decompress((root/'third_party/unifont/unifont_all-16.0.04.hex.gz').read_bytes()).decode().splitlines()}
lib.sdk_vterm_render.argtypes=[ctypes.c_void_p,ctypes.c_int,ctypes.c_int,ctypes.c_void_p,ctypes.c_int,ctypes.c_int,ctypes.c_void_p,ctypes.c_uint]
lib.sdk_vterm_start(1,8)
write('\x1b[?25l\x1b[37m\x1b[40m😀中╭e\u0301'.encode())
pixels=(ctypes.c_ubyte*(8*8*16))();palette=(ctypes.c_uint32*256)();legacy=(ctypes.c_ubyte*(256*16))()
lib.sdk_vterm_render(pixels,8,1,legacy,8,16,palette,0)
expected=bytearray(8*8*16)
for code,column,width in ((0x1f600,0,16),(0x4e2d,2,16),(0x256d,4,8),(ord('e'),5,8),(0x301,5,8)):
    bitmap=source[code]
    for y in range(16):
        for x in range(width):
            if bitmap[y*(width//8)+x//8] & (0x80>>(x%8)):
                expected[y*64+column*8+x]=7
assert bytes(pixels)==expected, ('Unicode framebuffer differs from upstream glyph pixels', [(i,a,b) for i,(a,b) in enumerate(zip(bytes(pixels),expected)) if a!=b][:16], [cell(x,0) for x in range(8)])
lib.sdk_vterm_start(2,4)
write('\x1b[?25l\x1b[37m\x1b[40me\u0301\r\n\r\n\r\n'.encode())
blank=(ctypes.c_ubyte*(4*8*2*16))()
lib.sdk_vterm_render(blank,4,2,legacy,8,16,palette,1)
assert not any(blank), 'Blank scrollback cell retained a previous combining glyph'
lib.sdk_vterm_stop()
print('Upstream terminal host integration PASS: split UTF-8, exact ANSI256 color, cursor motion/reply, alternate-screen restore, wide Unicode geometry')
print('Unchanged Unifont framebuffer PASS: emoji, CJK, rounded border, combining accent, wide continuation, malformed font rejection')
