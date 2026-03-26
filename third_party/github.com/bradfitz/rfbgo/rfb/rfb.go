/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// toy VNC (RFB) server in Go, just learning the protocol.
//
// Protocol docs:
//    http://www.realvnc.com/docs/rfbproto.pdf
//
// Author: Brad Fitzpatrick <brad@danga.com>
//
// Local changes:
// - export server name through Server.Name
// - prefer zero-rectangle incremental replies over a self-copy hack
// - add a fast path for common little-endian 32-bit true-colour clients
// - use RLock for image reads so capture writers do not serialize readers

package rfb

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
)

const (
	v3 = "RFB 003.003\n"
	v7 = "RFB 003.007\n"
	v8 = "RFB 003.008\n"

	authNone = 1

	statusOK = 0

	encodingRaw      = 0
	encodingCopyRect = 1

	// Client -> Server
	cmdSetPixelFormat           = 0
	cmdSetEncodings             = 2
	cmdFramebufferUpdateRequest = 3
	cmdKeyEvent                 = 4
	cmdPointerEvent             = 5
	cmdClientCutText            = 6

	// Server -> Client
	cmdFramebufferUpdate = 0
)

var EnableDebugLogs bool

func debugf(format string, args ...interface{}) {
	if EnableDebugLogs {
		log.Printf(format, args...)
	}
}

func NewServer(width, height int) *Server {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	conns := make(chan *Conn, 16)
	return &Server{
		width:  width,
		height: height,
		Name:   "rfb-go",
		Conns:  conns,
		conns:  conns,
	}
}

type Server struct {
	width, height int
	conns         chan *Conn

	// Name is reported in the ServerInit message.
	Name string

	// Conns is a channel of incoming connections.
	Conns <-chan *Conn
}

func (s *Server) Serve(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		conn := s.newConn(c)
		select {
		case s.conns <- conn:
		default:
		}
		go conn.serve()
	}
}

func (s *Server) newConn(c net.Conn) *Conn {
	feed := make(chan *FrameUpdate, 1)
	event := make(chan interface{}, 16)
	ready := make(chan struct{})
	done := make(chan struct{})
	conn := &Conn{
		s:      s,
		c:      c,
		br:     bufio.NewReader(c),
		bw:     bufio.NewWriter(c),
		fbupc:  make(chan FrameBufferUpdateRequest, 128),
		closec: done,
		feed:   feed,
		Feed:   feed,
		event:  event,
		Event:  event,
		ready:  ready,
		Ready:  ready,
		Done:   done,
	}
	return conn
}

type LockableImage struct {
	sync.RWMutex
	Img image.Image
}

type FrameUpdate struct {
	Frame *LockableImage
	Rect  image.Rectangle
}

type Conn struct {
	s      *Server
	c      net.Conn
	br     *bufio.Reader
	bw     *bufio.Writer
	fbupc  chan FrameBufferUpdateRequest
	closec chan struct{}

	formatMu sync.RWMutex
	format   PixelFormat

	feed chan *FrameUpdate
	mu   sync.RWMutex
	last *FrameUpdate
	ver  uint32

	buf8 []uint8

	debugRequestCount uint32
	debugPushCount    uint32
	debugEmptyCount   uint32
	debugPublishCount uint32
	debugReplaceCount uint32

	// Feed is the channel used to send new frames.
	// Deprecated: prefer PublishFrame so stale queued frames are replaced.
	Feed chan<- *FrameUpdate

	// Event receives KeyEvent or PointerEvent values and closes on disconnect.
	Event <-chan interface{}

	// Ready closes once the RFB handshake has completed and ServerInit is sent.
	Ready <-chan struct{}

	// Done closes when the connection shuts down.
	Done <-chan struct{}

	event chan interface{}
	ready chan struct{}
}

func (c *Conn) LocalAddr() net.Addr {
	if c == nil || c.c == nil {
		return nil
	}
	return c.c.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	if c == nil || c.c == nil {
		return nil
	}
	return c.c.RemoteAddr()
}

func (c *Conn) dimensions() (w, h int) {
	return c.s.width, c.s.height
}

func (c *Conn) readByte(what string) byte {
	b, err := c.br.ReadByte()
	if err != nil {
		c.failf("reading client byte for %q: %v", what, err)
	}
	return b
}

func (c *Conn) readPadding(what string, size int) {
	for i := 0; i < size; i++ {
		c.readByte(what)
	}
}

func (c *Conn) read(what string, v interface{}) {
	err := binary.Read(c.br, binary.BigEndian, v)
	if err != nil {
		c.failf("reading from client into %T for %q: %v", v, what, err)
	}
}

func (c *Conn) w(v interface{}) {
	if err := binary.Write(c.bw, binary.BigEndian, v); err != nil {
		c.failf("writing to client %T: %v", v, err)
	}
}

func (c *Conn) flush() {
	if err := c.bw.Flush(); err != nil {
		c.failf("flushing client output: %v", err)
	}
}

func (c *Conn) writeBytes(data []byte) {
	if len(data) == 0 {
		return
	}
	const maxChunk = 1024
	for len(data) > 0 {
		chunkLen := len(data)
		if chunkLen > maxChunk {
			chunkLen = maxChunk
		}
		chunk := data[:chunkLen]
		total := len(chunk)
		zeroProgress := 0
		for len(chunk) > 0 {
			n, err := c.c.Write(chunk)
			if n > 0 {
				chunk = chunk[n:]
				zeroProgress = 0
			}
			if err == nil && n > 0 {
				continue
			}
			if err == io.ErrShortWrite {
				zeroProgress++
				if zeroProgress < 8 {
					continue
				}
			}
			if err == nil && n == 0 {
				zeroProgress++
				if zeroProgress < 8 {
					continue
				}
				err = io.ErrShortWrite
			}
			if err == nil {
				continue
			}
			c.failf("writing %d-byte chunk to client: %v", total, err)
		}
		data = data[total:]
	}
}

func (c *Conn) failf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

func (c *Conn) setFormat(pf PixelFormat) {
	c.formatMu.Lock()
	c.format = pf
	c.formatMu.Unlock()
}

func (c *Conn) currentFormat() PixelFormat {
	c.formatMu.RLock()
	pf := c.format
	c.formatMu.RUnlock()
	return pf
}

func (c *Conn) DebugCounters() (requests uint32, pushes uint32, empties uint32, publishes uint32, replaces uint32) {
	return atomic.LoadUint32(&c.debugRequestCount),
		atomic.LoadUint32(&c.debugPushCount),
		atomic.LoadUint32(&c.debugEmptyCount),
		atomic.LoadUint32(&c.debugPublishCount),
		atomic.LoadUint32(&c.debugReplaceCount)
}

func (c *Conn) PublishFrame(update *FrameUpdate) {
	if c == nil || update == nil || update.Frame == nil {
		return
	}
	version := atomic.AddUint32(&c.debugPublishCount, 1)
	c.mu.Lock()
	c.last = update
	c.ver = version
	c.mu.Unlock()
	for {
		select {
		case c.feed <- update:
			return
		default:
		}
		select {
		case <-c.feed:
			atomic.AddUint32(&c.debugReplaceCount, 1)
		default:
		}
	}
}

func (c *Conn) currentPublished() (*FrameUpdate, uint32) {
	c.mu.RLock()
	update := c.last
	version := c.ver
	c.mu.RUnlock()
	return update, version
}

func (c *Conn) serve() {
	defer c.c.Close()
	defer close(c.fbupc)
	defer close(c.closec)
	defer close(c.event)
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("Client disconnect: %v", recovered)
		}
	}()

	_, _ = c.bw.WriteString(v8)
	c.flush()
	sl, err := c.br.ReadSlice('\n')
	if err != nil {
		c.failf("reading client protocol version: %v", err)
	}
	ver := string(sl)
	debugf("RFB version: %q", ver)
	switch ver {
	case v3, v7, v8:
	default:
		c.failf("bogus client-requested security type %q", ver)
	}

	if ver >= v7 {
		_, _ = c.bw.WriteString("\x01\x01")
		c.flush()
		wanted := c.readByte("6.1.2:client requested security-type")
		if wanted != authNone {
			c.failf("client wanted auth type %d, not None", int(wanted))
		}
		debugf("RFB security type: %d", int(wanted))
	} else {
		c.w(uint32(authNone))
		c.flush()
		debugf("RFB security type: legacy-none")
	}

	if ver >= v7 {
		c.w(uint32(statusOK))
		c.flush()
	}

	shared := c.readByte("shared-flag") != 0
	debugf("RFB client init: shared=%t", shared)

	c.setFormat(PixelFormat{
		BPP:        32,
		Depth:      24,
		BigEndian:  0,
		TrueColour: 1,
		RedMax:     0xff,
		GreenMax:   0xff,
		BlueMax:    0xff,
		RedShift:   16,
		GreenShift: 8,
		BlueShift:  0,
	})
	format := c.currentFormat()

	width, height := c.dimensions()
	c.w(uint16(width))
	c.w(uint16(height))
	c.w(format.BPP)
	c.w(format.Depth)
	c.w(format.BigEndian)
	c.w(format.TrueColour)
	c.w(format.RedMax)
	c.w(format.GreenMax)
	c.w(format.BlueMax)
	c.w(format.RedShift)
	c.w(format.GreenShift)
	c.w(format.BlueShift)
	c.w(uint8(0))
	c.w(uint8(0))
	c.w(uint8(0))
	serverName := c.s.Name
	if serverName == "" {
		serverName = "rfb-go"
	}
	c.w(int32(len(serverName)))
	_, _ = c.bw.WriteString(serverName)
	c.flush()
	close(c.ready)
	debugf("RFB server init sent: %dx%d / %s", width, height, serverName)
	go c.pushFramesLoop()

	for {
		cmd := c.readByte("6.4:client-server-packet-type")
		switch cmd {
		case cmdSetPixelFormat:
			c.handleSetPixelFormat()
		case cmdSetEncodings:
			c.handleSetEncodings()
		case cmdFramebufferUpdateRequest:
			c.handleUpdateRequest()
		case cmdPointerEvent:
			c.handlePointerEvent()
		case cmdKeyEvent:
			c.handleKeyEvent()
		case cmdClientCutText:
			c.handleClientCutText()
		default:
			c.failf("unsupported command type %d from client", int(cmd))
		}
	}
}

func (c *Conn) pushFramesLoop() {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("RFB push loop disconnect: %v", recovered)
			_ = c.c.Close()
		}
	}()

	var pending bool
	var havePending bool
	var lastSentVersion uint32
	for {
		select {
		case req, ok := <-c.fbupc:
			if !ok {
				return
			}
			update, version := c.currentPublished()
			if update == nil || update.Frame == nil {
				_ = req
				pending = true
				havePending = true
				continue
			}
			if req.incremental() && version == lastSentVersion {
				c.pushEmptyUpdate()
			} else {
				c.pushFrameUpdate(update, c.currentFormat(), !req.incremental())
				lastSentVersion = version
			}
			havePending = false
		case update, ok := <-c.feed:
			if !ok {
				return
			}
			if update == nil || update.Frame == nil {
				continue
			}
			if havePending && pending {
				current, version := c.currentPublished()
				if current == nil || current.Frame == nil {
					continue
				}
				if version == lastSentVersion {
					c.pushEmptyUpdate()
				} else {
					c.pushFrameUpdate(current, c.currentFormat(), false)
					lastSentVersion = version
				}
				pending = false
				havePending = false
			}
		}
	}
}

func (c *Conn) pushEmptyUpdate() {
	debugSeq := atomic.AddUint32(&c.debugEmptyCount, 1)
	debugEnabled := debugSeq <= 12 || debugSeq%16 == 0
	if debugEnabled {
		debugf("RFB push empty #%d", debugSeq)
	}
	c.w(uint8(cmdFramebufferUpdate))
	c.w(uint8(0))
	c.w(uint16(0))
	c.flush()
	if debugEnabled {
		debugf("RFB push empty #%d: complete", debugSeq)
	}
}

func (c *Conn) pushFrameUpdate(update *FrameUpdate, format PixelFormat, forceFull bool) {
	li := update.Frame
	li.RLock()
	defer li.RUnlock()

	im := li.Img
	b := im.Bounds()
	if b.Min.X != 0 || b.Min.Y != 0 {
		panic("this code is lazy and assumes images with Min bounds at 0,0")
	}
	rect := update.Rect
	if forceFull || rect.Empty() {
		rect = b
	} else {
		rect = rect.Intersect(b)
		if rect.Empty() {
			c.pushEmptyUpdate()
			return
		}
	}
	width, height := rect.Dx(), rect.Dy()
	debugSeq := atomic.AddUint32(&c.debugPushCount, 1)
	debugEnabled := debugSeq <= 12 || debugSeq%16 == 0
	if debugEnabled {
		debugf(
			"RFB push image #%d: x=%d y=%d w=%d h=%d / fmt bpp=%d depth=%d be=%d tc=%d",
			debugSeq, rect.Min.X, rect.Min.Y, width, height,
			format.BPP, format.Depth, format.BigEndian, format.TrueColour,
		)
	}

	c.w(uint8(cmdFramebufferUpdate))
	c.w(uint8(0))
	c.w(uint16(1))
	c.w(uint16(rect.Min.X))
	c.w(uint16(rect.Min.Y))
	c.w(uint16(width))
	c.w(uint16(height))
	c.w(int32(encodingRaw))
	c.flush()

	rgba, isRGBA := im.(*image.RGBA)
	switch {
	case isRGBA && format.isScreensThousands():
		c.pushRGBAScreensThousandsLocked(rgba, rect, format)
	case isRGBA && format.BPP == 8 && format.TrueColour != 0:
		c.pushRGBA8TrueColourLocked(rgba, rect, format)
	case isRGBA && format.isLittleEndian32TrueColour():
		c.pushRGBALittleEndian32Locked(rgba, rect)
	default:
		c.pushGenericLocked(im, rect, format)
	}
	c.flush()
	if debugEnabled {
		debugf("RFB push image #%d: complete", debugSeq)
	}
}

func (c *Conn) pushRGBAScreensThousandsLocked(im *image.RGBA, rect image.Rectangle, format PixelFormat) {
	width := rect.Dx()
	height := rect.Dy()
	needed := width * 2
	if len(c.buf8) < needed {
		c.buf8 = make([]byte, needed)
	}
	out := c.buf8[:needed]
	isBigEndian := format.BigEndian != 0
	for y := rect.Min.Y; y < rect.Min.Y+height; y++ {
		rowStart := y*im.Stride + rect.Min.X*4
		row := im.Pix[rowStart : rowStart+width*4]
		di := 0
		for x := 0; x < width*4; x += 4 {
			u16 := uint16(row[x]&248)<<7 | uint16(row[x+1]&248)<<2 | uint16(row[x+2]>>3)
			if isBigEndian {
				out[di] = uint8(u16 >> 8)
				out[di+1] = uint8(u16)
			} else {
				out[di] = uint8(u16)
				out[di+1] = uint8(u16 >> 8)
			}
			di += 2
		}
		c.writeBytes(out)
	}
}

func (c *Conn) pushRGBALittleEndian32Locked(im *image.RGBA, rect image.Rectangle) {
	width := rect.Dx()
	height := rect.Dy()
	needed := width * 4
	if len(c.buf8) < needed {
		c.buf8 = make([]byte, needed)
	}
	out := c.buf8[:needed]
	for y := rect.Min.Y; y < rect.Min.Y+height; y++ {
		rowStart := y*im.Stride + rect.Min.X*4
		row := im.Pix[rowStart : rowStart+width*4]
		di := 0
		for x := 0; x < width*4; x += 4 {
			out[di] = row[x+2]
			out[di+1] = row[x+1]
			out[di+2] = row[x]
			out[di+3] = 0
			di += 4
		}
		c.writeBytes(out)
	}
}

func (c *Conn) pushRGBA8TrueColourLocked(im *image.RGBA, rect image.Rectangle, format PixelFormat) {
	width := rect.Dx()
	height := rect.Dy()
	if len(c.buf8) < width {
		c.buf8 = make([]byte, width)
	}
	out := c.buf8[:width]
	rmax := uint32(format.RedMax)
	gmax := uint32(format.GreenMax)
	bmax := uint32(format.BlueMax)
	for y := rect.Min.Y; y < rect.Min.Y+height; y++ {
		rowStart := y*im.Stride + rect.Min.X*4
		row := im.Pix[rowStart : rowStart+width*4]
		di := 0
		for x := 0; x < width*4; x += 4 {
			r := scale8ToMax(row[x], rmax)
			g := scale8ToMax(row[x+1], gmax)
			b := scale8ToMax(row[x+2], bmax)
			out[di] = byte((r << format.RedShift) | (g << format.GreenShift) | (b << format.BlueShift))
			di++
		}
		c.writeBytes(out)
	}
}

// pushGenericLocked is the slow path that works for any image.Image concrete
// type and any client-requested pixel format.
func (c *Conn) pushGenericLocked(im image.Image, rect image.Rectangle, format PixelFormat) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			col := im.At(x, y)
			r16, g16, b16, _ := col.RGBA()
			r16 = inRange(r16, format.RedMax)
			g16 = inRange(g16, format.GreenMax)
			b16 = inRange(b16, format.BlueMax)
			u32 := (r16 << format.RedShift) |
				(g16 << format.GreenShift) |
				(b16 << format.BlueShift)
			var v interface{}
			switch format.BPP {
			case 32:
				v = u32
			case 16:
				v = uint16(u32)
			case 8:
				v = uint8(u32)
			default:
				c.failf("TODO: BPP of %d", format.BPP)
			}
			if format.BigEndian != 0 {
				_ = binary.Write(c.bw, binary.BigEndian, v)
			} else {
				_ = binary.Write(c.bw, binary.LittleEndian, v)
			}
		}
	}
}

type PixelFormat struct {
	BPP, Depth                      uint8
	BigEndian, TrueColour           uint8
	RedMax, GreenMax, BlueMax       uint16
	RedShift, GreenShift, BlueShift uint8
}

func (f *PixelFormat) isScreensThousands() bool {
	return f.BPP == 16 && (f.Depth == 16 || f.Depth == 15) && f.TrueColour != 0 &&
		f.RedMax == 0x1f && f.GreenMax == 0x1f && f.BlueMax == 0x1f &&
		f.RedShift == 10 && f.GreenShift == 5 && f.BlueShift == 0
}

func (f *PixelFormat) isLittleEndian32TrueColour() bool {
	return f.BPP == 32 && (f.Depth == 32 || f.Depth == 24) &&
		f.BigEndian == 0 && f.TrueColour != 0 &&
		f.RedMax == 0xff && f.GreenMax == 0xff && f.BlueMax == 0xff &&
		f.RedShift == 16 && f.GreenShift == 8 && f.BlueShift == 0
}

func (c *Conn) handleSetPixelFormat() {
	c.readPadding("SetPixelFormat padding", 3)
	var pf PixelFormat
	c.read("pixelformat.bpp", &pf.BPP)
	c.read("pixelformat.depth", &pf.Depth)
	c.read("pixelformat.beflag", &pf.BigEndian)
	c.read("pixelformat.truecolour", &pf.TrueColour)
	c.read("pixelformat.redmax", &pf.RedMax)
	c.read("pixelformat.greenmax", &pf.GreenMax)
	c.read("pixelformat.bluemax", &pf.BlueMax)
	c.read("pixelformat.redshift", &pf.RedShift)
	c.read("pixelformat.greenshift", &pf.GreenShift)
	c.read("pixelformat.blueshift", &pf.BlueShift)
	c.readPadding("SetPixelFormat pixel format padding", 3)
	c.setFormat(pf)
	debugf(
		"RFB pixel format: bpp=%d depth=%d be=%d tc=%d rmax=%d gmax=%d bmax=%d rshift=%d gshift=%d bshift=%d",
		pf.BPP, pf.Depth, pf.BigEndian, pf.TrueColour,
		pf.RedMax, pf.GreenMax, pf.BlueMax,
		pf.RedShift, pf.GreenShift, pf.BlueShift,
	)
}

func (c *Conn) handleSetEncodings() {
	c.readPadding("SetEncodings padding", 1)

	var numEncodings uint16
	c.read("6.4.2:number-of-encodings", &numEncodings)
	encodings := make([]int32, 0, int(numEncodings))
	rawOK := false
	for i := 0; i < int(numEncodings); i++ {
		var t int32
		c.read("encoding-type", &t)
		encodings = append(encodings, t)
		if t == encodingRaw {
			rawOK = true
		}
	}
	debugf("RFB encodings: raw=%t values=%v", rawOK, encodings)
}

type FrameBufferUpdateRequest struct {
	IncrementalFlag     uint8
	X, Y, Width, Height uint16
}

func (r *FrameBufferUpdateRequest) incremental() bool { return r.IncrementalFlag != 0 }

func (c *Conn) handleUpdateRequest() {
	var req FrameBufferUpdateRequest
	c.read("framebuffer-update.incremental", &req.IncrementalFlag)
	c.read("framebuffer-update.x", &req.X)
	c.read("framebuffer-update.y", &req.Y)
	c.read("framebuffer-update.width", &req.Width)
	c.read("framebuffer-update.height", &req.Height)
	debugSeq := atomic.AddUint32(&c.debugRequestCount, 1)
	if debugSeq <= 12 || debugSeq%16 == 0 {
		format := c.currentFormat()
		debugf(
			"RFB update request #%d: incremental=%t x=%d y=%d w=%d h=%d fmt-now=bpp=%d depth=%d",
			debugSeq,
			req.incremental(), req.X, req.Y, req.Width, req.Height,
			format.BPP, format.Depth,
		)
	}
	c.fbupc <- req
}

type KeyEvent struct {
	DownFlag uint8
	Key      uint32
}

func (c *Conn) handleKeyEvent() {
	var req KeyEvent
	c.read("key-event.downflag", &req.DownFlag)
	c.readPadding("key-event.padding", 2)
	c.read("key-event.key", &req.Key)
	select {
	case c.event <- req:
	default:
	}
}

type PointerEvent struct {
	ButtonMask uint8
	X, Y       uint16
}

func (c *Conn) handlePointerEvent() {
	var req PointerEvent
	c.read("pointer-event.mask", &req.ButtonMask)
	c.read("pointer-event.x", &req.X)
	c.read("pointer-event.y", &req.Y)
	select {
	case c.event <- req:
	default:
	}
}

func (c *Conn) handleClientCutText() {
	c.readPadding("client-cut-text.padding", 3)
	var textLength uint32
	c.read("client-cut-text.length", &textLength)
	for textLength > 0 {
		chunk := int(textLength)
		if chunk > 256 {
			chunk = 256
		}
		var discard [256]byte
		if _, err := io.ReadFull(c.br, discard[:chunk]); err != nil {
			c.failf("reading client cut text: %v", err)
		}
		textLength -= uint32(chunk)
	}
}

func inRange(v uint32, max uint16) uint32 {
	switch {
	case max == 0:
		return 0
	case max == 0xffff:
		return v
	default:
		// Scale 16-bit channel values into the client-requested channel range.
		return (v*uint32(max) + 0x7fff) / 0xffff
	}
}

func scale8ToMax(value byte, max uint32) uint32 {
	switch {
	case max == 0:
		return 0
	case max == 0xff:
		return uint32(value)
	default:
		return (uint32(value)*max + 127) / 255
	}
}
