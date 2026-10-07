/* Native console adapter around unchanged libvterm 0.3.3. */
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <stdlib.h>
#include "vterm.h"
#include "vterm-font-tables.h"
#include "utf8.h"

/* Original libvterm encoder; the native keyboard supplies CP866 bytes. */
int sdk_vterm_encode_cp866(unsigned int code, char *output)
{
    if (!output || code>255) return 0;
    return fill_utf8(code<128?code:sdk_cp866_unicode[code-128],output);
}

static VTerm *terminal;
static VTermScreen *screen;
static VTermState *state;
static int cursor_visible = 1;
static int alternate_screen;
static int mouse_mode;
static unsigned int mouse_buttons;
enum { HISTORY_LIMIT = 1000 };
struct HistoryLine { int columns; VTermScreenCell *cells; };
static struct HistoryLine history[HISTORY_LIMIT];
static int history_count, history_first;
static const unsigned char *unicode_font;
static uint32_t unicode_count;

static uint32_t font_word(const unsigned char *p)
{ return (uint32_t)p[0] | (uint32_t)p[1]<<8 | (uint32_t)p[2]<<16 | (uint32_t)p[3]<<24; }

static int vterm_impl_set_font(const unsigned char *data, size_t length)
{
    if (!data || length<8 || memcmp(data,"KBF1",4)) return 0;
    uint32_t count=font_word(data+4);
    if (count>(length-8)/12) return 0;
    size_t first=8+(size_t)count*12;
    uint32_t previous=0;
    for (uint32_t i=0; i<count; ++i) {
        const unsigned char *entry=data+8+(size_t)i*12;
        uint32_t code=font_word(entry), offset=font_word(entry+4), width=font_word(entry+8);
        if (code>0x10ffff || (i && code<=previous) || (width!=1 && width!=2) ||
            offset<first || offset>length || width*16>length-offset) return 0;
        previous=code;
    }
    unicode_font=data; unicode_count=count;
    return 1;
}

static const unsigned char *unicode_glyph(uint32_t code, int *width)
{
    uint32_t low=0, high=unicode_count;
    while (low<high) {
        uint32_t middle=low+(high-low)/2;
        const unsigned char *entry=unicode_font+8+(size_t)middle*12;
        uint32_t found=font_word(entry);
        if (found<code) low=middle+1;
        else if (found>code) high=middle;
        else { *width=font_word(entry+8)*8; return unicode_font+font_word(entry+4); }
    }
    return NULL;
}

static int clear_history(void *user)
{
    (void)user;
    for (int i=0; i<history_count; ++i)
        free(history[(history_first+i)%HISTORY_LIMIT].cells);
    memset(history,0,sizeof(history));
    history_count=history_first=0;
    return 1;
}

static int push_history(int columns, const VTermScreenCell *cells, void *user)
{
    (void)user;
    VTermScreenCell *copy=malloc((size_t)columns*sizeof(*copy));
    if (!copy) return 0;
    memcpy(copy,cells,(size_t)columns*sizeof(*copy));
    if (history_count==HISTORY_LIMIT) {
        free(history[history_first].cells);
        history_first=(history_first+1)%HISTORY_LIMIT;
        --history_count;
    }
    history[(history_first+history_count++)%HISTORY_LIMIT]=(struct HistoryLine){columns,copy};
    return 1;
}

static int pop_history(int columns, VTermScreenCell *cells, void *user)
{
    (void)user;
    if (!history_count) return 0;
    struct HistoryLine *line=&history[(history_first+--history_count)%HISTORY_LIMIT];
    memset(cells,0,(size_t)columns*sizeof(*cells));
    int copy=columns<line->columns?columns:line->columns;
    memcpy(cells,line->cells,(size_t)copy*sizeof(*cells));
    for (int i=copy; i<columns; ++i) { cells[i].chars[0]=' '; cells[i].width=1; }
    free(line->cells); line->cells=NULL;
    return 1;
}

static int term_property(VTermProp prop, VTermValue *value, void *user)
{
    (void)user;
    if (prop == VTERM_PROP_CURSORVISIBLE)
        cursor_visible = value->boolean;
    if (prop == VTERM_PROP_ALTSCREEN)
        alternate_screen = value->boolean;
    if (prop == VTERM_PROP_MOUSE)
        mouse_mode = value->number;
    return 1;
}

static const VTermScreenCallbacks callbacks = {
    .settermprop = term_property, .sb_pushline=push_history,
    .sb_popline=pop_history, .sb_clear=clear_history
};

int vterm_impl_start(int rows, int columns)
{
    if (rows < 1 || columns < 1)
        return 0;
    if (terminal)
        vterm_free(terminal);
    clear_history(NULL);
    terminal = vterm_new(rows, columns);
    if (!terminal)
        return 0;
    vterm_set_utf8(terminal, 1);
    screen = vterm_obtain_screen(terminal);
    state = vterm_obtain_state(terminal);
    vterm_screen_enable_altscreen(screen, 1);
    vterm_screen_set_callbacks(screen, &callbacks, NULL);
    VTermColor fg, bg;
    vterm_color_rgb(&fg, 192, 192, 192);
    vterm_color_rgb(&bg, 0, 0, 0);
    vterm_state_set_default_colors(state, &fg, &bg);
    for (int i = 0; i < 16; ++i) {
        uint32_t rgb = sdk_ansi_rgb[i];
        VTermColor color;
        vterm_color_rgb(&color, rgb >> 16, rgb >> 8, rgb);
        vterm_state_set_palette_color(state, i, &color);
    }
    cursor_visible = 1;
    alternate_screen = 0;
    mouse_mode = VTERM_PROP_MOUSE_NONE;
    mouse_buttons = 0;
    vterm_screen_reset(screen, 1);
    return 1;
}

void vterm_impl_stop(void)
{
    if (terminal)
        vterm_free(terminal);
    terminal = NULL;
    screen = NULL;
    state = NULL;
    clear_history(NULL);
}

size_t vterm_impl_write(const char *data, size_t length)
{
    if (!terminal)
        return 0;
    size_t used = vterm_input_write(terminal, data, length);
    vterm_screen_flush_damage(screen);
    return used;
}

void vterm_impl_cursor(uint32_t *x, uint32_t *y)
{
    VTermPos pos = {0, 0};
    if (state)
        vterm_state_get_cursorpos(state, &pos);
    if (x) *x = pos.col;
    if (y) *y = pos.row;
}

void vterm_impl_layout(uint32_t *height, uint32_t *offset, uint32_t *cursor_y)
{
    int rows=0, columns=0;
    if (terminal) vterm_get_size(terminal,&rows,&columns);
    uint32_t history_rows=alternate_screen?0:history_count;
    if (height) *height=rows+history_rows;
    if (offset) *offset=history_rows;
    if (cursor_y) {
        VTermPos pos={0,0};
        if (state) vterm_state_get_cursorpos(state,&pos);
        *cursor_y=history_rows+pos.row;
    }
}

static uint32_t color_rgb(const VTermColor *color)
{
    if (VTERM_COLOR_IS_INDEXED(color))
        return sdk_ansi_rgb[color->indexed.idx];
    return ((uint32_t)color->rgb.red << 16) |
           ((uint32_t)color->rgb.green << 8) | color->rgb.blue;
}

int vterm_impl_cell(int x, int y, uint32_t *data)
{
    if (!screen || !data)
        return 0;
    VTermScreenCell cell;
    if (!vterm_screen_get_cell(screen, (VTermPos){y, x}, &cell))
        return 0;
    data[0] = cell.chars[0];
    data[1] = color_rgb(&cell.fg);
    data[2] = color_rgb(&cell.bg);
    if (cell.attrs.reverse) {
        uint32_t swap = data[1]; data[1] = data[2]; data[2] = swap;
    }
    data[3] = cell.width;
    return 1;
}

static unsigned char glyph(uint32_t code)
{
    if (!code || code == UINT32_MAX) return ' ';
    if (code < 128) return code;
    if (code == 0x256d) code = 0x250c;
    if (code == 0x256e) code = 0x2510;
    if (code == 0x256f) code = 0x2518;
    if (code == 0x2570) code = 0x2514;
    for (int i = 0; i < 128; ++i)
        if (sdk_cp866_unicode[i] == code) return i + 128;
    return '?';
}

static unsigned char palette_index(const VTermColor *color, uint32_t *palette)
{
    if (VTERM_COLOR_IS_INDEXED(color)) return color->indexed.idx;
    uint32_t rgb = color_rgb(color);
    unsigned best = 0, distance = UINT32_MAX;
    for (unsigned i = 0; i < 256; ++i) {
        int r = (int)((rgb >> 16) & 255) - (int)((palette[i] >> 16) & 255);
        int g = (int)((rgb >> 8) & 255) - (int)((palette[i] >> 8) & 255);
        int b = (int)(rgb & 255) - (int)(palette[i] & 255);
        unsigned d = r*r + g*g + b*b;
        if (d < distance) { best = i; distance = d; }
        if (!d) break;
    }
    return best;
}

/* The original CP866 bitmap font remains the fallback. Imported Unifont
 * pixels are MSB first, with unchanged eight- or sixteen-pixel rows. */
void vterm_impl_render(unsigned char *pixels, int columns, int rows,
                      const unsigned char *font, int fw, int fh, uint32_t *palette,
                      unsigned int viewport)
{
    if (!screen || !pixels || !font || !palette) return;
    int term_rows, term_cols;
    vterm_get_size(terminal, &term_rows, &term_cols);
    memcpy(palette, sdk_ansi_rgb, sizeof(sdk_ansi_rgb));
    memset(pixels, 0, (size_t)columns * rows * fw * fh);
    VTermPos cursor;
    vterm_state_get_cursorpos(state, &cursor);
    int history_rows=alternate_screen?0:history_count;
    if (alternate_screen) viewport=0;
    if (viewport>(unsigned int)history_rows) viewport=history_rows;
    for (int pass=0; pass<2; ++pass)
    for (int y = 0; y < rows && y < term_rows; ++y) {
        for (int x = 0; x < columns && x < term_cols; ++x) {
            VTermScreenCell cell = {0};
            int line=y+viewport;
            if (line<history_rows) {
                struct HistoryLine *saved=&history[(history_first+line)%HISTORY_LIMIT];
                if (x>=saved->columns) continue;
                cell=saved->cells[x];
            } else if (!vterm_screen_get_cell(screen, (VTermPos){line-history_rows, x}, &cell)) continue;
            unsigned char fg = palette_index(&cell.fg, palette);
            unsigned char bg = palette_index(&cell.bg, palette);
            if (cell.attrs.reverse) { unsigned char swap = fg; fg = bg; bg = swap; }
            if (!pass) {
                for (int row=0; row<fh; ++row)
                    memset(pixels+((y*fh+row)*columns+x)*fw,bg,fw);
                continue;
            }
            if (cell.chars[0]==UINT32_MAX) continue;
            unsigned char ch = cell.attrs.conceal ? ' ' : glyph(cell.chars[0]);
            int span=cell.width==2?2:1;
            if (span>columns-x) span=columns-x;
            for (int row = 0; row < fh; ++row) {
                unsigned bits = fw > 8
                    ? (unsigned)font[row*512+ch*2] | ((unsigned)font[row*512+ch*2+1] << 8)
                    : font[row*256+ch];
                unsigned char *out = pixels + ((y*fh+row)*columns+x)*fw;
                int bitmap_width=0;
                const unsigned char *bitmap=cell.attrs.conceal?NULL:unicode_glyph(cell.chars[0]?cell.chars[0]:' ',&bitmap_width);
                for (int column=0; column<fw*span; ++column) {
                    int ink;
                    if (bitmap) {
                        int sx=column*bitmap_width/(fw*(cell.width==2?2:1));
                        const unsigned char *scan=bitmap+(row*16/fh)*(bitmap_width/8);
                        ink=(scan[sx/8]>>(7-sx%8))&1;
                        if (cell.attrs.bold && sx>0) ink|=(scan[(sx-1)/8]>>(7-(sx-1)%8))&1;
                    } else {
                        unsigned bold=cell.attrs.bold?bits|(bits<<1):bits;
                        ink=column<fw && ((bold>>column)&1);
                    }
                    for (int mark=1; !cell.attrs.conceal && cell.chars[0] && mark<VTERM_MAX_CHARS_PER_CELL && cell.chars[mark]; ++mark) {
                        int mw=0; const unsigned char *combining=unicode_glyph(cell.chars[mark],&mw);
                        if (combining) {
                            int sx=column*mw/(fw*(cell.width==2?2:1));
                            ink|=(combining[(row*16/fh)*(mw/8)+sx/8]>>(7-sx%8))&1;
                        }
                    }
                    if ((cell.attrs.underline && row==fh-2) || (cell.attrs.strike && row==fh/2)) ink=1;
                    if (ink) out[column]=fg;
                }
            }
            if (cursor_visible && x == cursor.col && line == cursor.row+history_rows) {
                unsigned char *out = pixels + ((y*fh+fh-1)*columns+x)*fw;
                for (int column = 0; column < fw; ++column) out[column] = fg;
            }
        }
    }
}

int vterm_impl_output_pending(void)
{
    return terminal ? (int)vterm_output_get_buffer_current(terminal) : 0;
}

size_t vterm_impl_read_response(char *data, size_t length)
{
    return terminal ? vterm_output_read(terminal, data, length) : 0;
}

static volatile unsigned int engine_busy;
#ifdef KOLIBRIOS
extern void sdk_console_yield(void);
#endif
static void engine_enter(void)
{
    while (__atomic_exchange_n(&engine_busy, 1, __ATOMIC_ACQUIRE)) {
#ifdef KOLIBRIOS
        sdk_console_yield();
#else
        __asm__ volatile("pause");
#endif
    }
}
static void engine_leave(void) { __atomic_store_n(&engine_busy, 0, __ATOMIC_RELEASE); }

int sdk_vterm_mouse_active(void)
{ engine_enter(); int active = terminal && mouse_mode != VTERM_PROP_MOUSE_NONE; engine_leave(); return active; }

/* Native values are read by the upstream console window thread: sysfuncs.txt
 * 37/1,2,7 and 66/3. Upstream libvterm owns all mouse protocol encoding. */
void sdk_vterm_mouse(int x, int y, unsigned int buttons, unsigned int scroll,
                     unsigned int controls, int fw, int fh)
{
    engine_enter();
    if (!terminal || mouse_mode == VTERM_PROP_MOUSE_NONE || fw <= 0 || fh <= 0) {
        engine_leave(); return;
    }
    int rows, columns;
    vterm_get_size(terminal, &rows, &columns);
    int inside = x >= 0 && y >= 0 && x < columns*fw && y < rows*fh;
    if (!inside && !mouse_buttons) { engine_leave(); return; }
    int row = y < 0 ? 0 : y/fh, column = x < 0 ? 0 : x/fw;
    if (row >= rows) row = rows-1;
    if (column >= columns) column = columns-1;
    VTermModifier modifiers = VTERM_MOD_NONE;
    if (controls & 3) modifiers |= VTERM_MOD_SHIFT;
    if (controls & 12) modifiers |= VTERM_MOD_CTRL;
    if (controls & 48) modifiers |= VTERM_MOD_ALT;
    vterm_mouse_move(terminal, row, column, modifiers);
    const int native_to_terminal[3] = {1, 3, 2};
    for (int bit = 0; bit < 3; ++bit) {
        if ((buttons ^ mouse_buttons) & (1u << bit))
            vterm_mouse_button(terminal, native_to_terminal[bit], !!(buttons & (1u << bit)), modifiers);
    }
    mouse_buttons = buttons & 7;
    int vertical = (int16_t)scroll, horizontal = (int16_t)(scroll >> 16);
    if (inside) {
        while (vertical != 0) {
            vterm_mouse_button(terminal, vertical < 0 ? 4 : 5, 1, modifiers);
            vertical += vertical < 0 ? 1 : -1;
        }
        while (horizontal != 0) {
            vterm_mouse_button(terminal, horizontal < 0 ? 6 : 7, 1, modifiers);
            horizontal += horizontal < 0 ? 1 : -1;
        }
    }
    engine_leave();
}
int sdk_vterm_start(int r, int c)
{ engine_enter(); int result = vterm_impl_start(r,c); engine_leave(); return result; }
int sdk_vterm_set_font(const unsigned char *p, size_t n)
{ engine_enter(); int result=vterm_impl_set_font(p,n); engine_leave(); return result; }
int sdk_vterm_glyph_width(uint32_t code)
{ engine_enter(); int width=0; unicode_glyph(code,&width); engine_leave(); return width; }
void sdk_vterm_stop(void)
{ engine_enter(); vterm_impl_stop(); engine_leave(); }
size_t sdk_vterm_write(const char *p, size_t n)
{ engine_enter(); size_t result = vterm_impl_write(p,n); engine_leave(); return result; }
void sdk_vterm_cursor(uint32_t *x, uint32_t *y)
{ engine_enter(); vterm_impl_cursor(x,y); engine_leave(); }
int sdk_vterm_cell(int x, int y, uint32_t *p)
{ engine_enter(); int result = vterm_impl_cell(x,y,p); engine_leave(); return result; }
void sdk_vterm_render(unsigned char *p,int c,int r,const unsigned char *f,int w,int h,uint32_t *pal,unsigned int viewport)
{ engine_enter(); vterm_impl_render(p,c,r,f,w,h,pal,viewport); engine_leave(); }
void sdk_vterm_layout(uint32_t *h,uint32_t *o,uint32_t *y)
{ engine_enter(); vterm_impl_layout(h,o,y); engine_leave(); }
int sdk_vterm_output_pending(void)
{ engine_enter(); int result = vterm_impl_output_pending(); engine_leave(); return result; }
size_t sdk_vterm_read_response(char *p, size_t n)
{ engine_enter(); size_t result = vterm_impl_read_response(p,n); engine_leave(); return result; }
