package main

import (
	"errors"
	"fmt"
	"image"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bradfitz/rfbgo/rfb"
	"kos"
)

const (
	appTitle      = "KolibriOS VNC Server"
	defaultPort   = 5900
	defaultFPS    = 4
	maxFPS        = 30
	ipcBufferSize = 4096
)

var errUsage = errors.New("usage")
var vncIPCBuffer [ipcBufferSize]byte

type options struct {
	port  int
	fps   int
	once  bool
	debug bool
}

var debugLogs bool

type screenStream struct {
	width      int
	height     int
	bpp        int
	pitch      int
	pixelOrder pixelOrder
	frames     []frameSlot
	captureIdx int
	currentIdx int
	hasCurrent bool
	raw        []byte
	bgr        []byte
}

type frameSlot struct {
	frame *rfb.LockableImage
	rgba  *image.RGBA
	rect  image.Rectangle
}

type pixelOrder int

const (
	pixelOrderUnknown pixelOrder = iota
	pixelOrderBGR
	pixelOrderRGB
)

func (order pixelOrder) String() string {
	switch order {
	case pixelOrderBGR:
		return "bgr"
	case pixelOrderRGB:
		return "rgb"
	default:
		return "unknown"
	}
}

func debugLogf(format string, args ...interface{}) {
	if debugLogs {
		log.Printf(format, args...)
	}
}

func main() {
	runtime.GOMAXPROCS(4)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.SetFlags(0)

	console, ok := kos.OpenConsole(appTitle)
	if !ok {
		kos.DebugString("vncsrv: failed to open /sys/lib/console.obj")
		os.Exit(1)
		return
	}
	if console.SupportsTitle() {
		console.SetTitle(appTitle)
	}

	kos.RegisterIPCBuffer(vncIPCBuffer[:])
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskIPC | kos.EventMaskNetwork)

	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintf(os.Stderr, "vncsrv panic: %T %v\n", recovered, recovered)
			waitForExit(console)
			os.Exit(2)
		}
	}()

	opts, err := parseArgs()
	if err != nil {
		if errors.Is(err, errUsage) {
			printUsage()
			waitForExit(console)
			os.Exit(2)
			return
		}
		fmt.Fprintf(os.Stderr, "vncsrv: %v\n", err)
		waitForExit(console)
		os.Exit(2)
		return
	}
	debugLogs = opts.debug
	rfb.EnableDebugLogs = opts.debug

	screenW, screenH := kos.ScreenSize()
	if screenW <= 0 || screenH <= 0 {
		fmt.Fprintln(os.Stderr, "vncsrv: invalid screen size")
		waitForExit(console)
		os.Exit(1)
		return
	}

	listenAddr := net.JoinHostPort("0.0.0.0", strconv.Itoa(opts.port))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vncsrv: listen %s failed: %v\n", listenAddr, err)
		waitForExit(console)
		os.Exit(1)
		return
	}
	defer listener.Close()

	server := rfb.NewServer(screenW, screenH)
	server.Name = "KolibriOS"
	stream := newScreenStream(screenW, screenH)

	serveErrs := make(chan error, 1)
	go func() {
		serveErrs <- server.Serve(listener)
	}()

	log.Printf("Listening on %s", listenAddr)
	log.Printf("Runtime: GOMAXPROCS=%d", runtime.GOMAXPROCS(0))
	log.Printf("Screen: %dx%d / %d fps", screenW, screenH, opts.fps)
	log.Printf("Mode: view-only / full-frame raw updates / resolution fixed at startup")
	if opts.once {
		log.Printf("Mode detail: one client, then exit")
	}

	for {
		select {
		case err = <-serveErrs:
			if err == nil || errors.Is(err, net.ErrClosed) {
				return
			}
			fmt.Fprintf(os.Stderr, "vncsrv: serve failed: %v\n", err)
			waitForExit(console)
			os.Exit(1)
			return
		case conn := <-server.Conns:
			remote := addrString(conn.RemoteAddr())
			if remote == "" {
				remote = "<unknown>"
			}
			log.Printf("Client connected: %s", remote)
			serveClient(conn, stream, opts.fps)
			log.Printf("Client disconnected: %s", remote)
			if opts.once {
				_ = listener.Close()
				return
			}
		}
	}
}

func parseArgs() (options, error) {
	opts := options{
		port: defaultPort,
		fps:  defaultFPS,
	}

	args := os.Args[1:]
	for len(args) > 0 {
		arg := args[0]
		if arg == "--" {
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return opts, fmt.Errorf("unexpected argument: %s", arg)
		}

		switch {
		case arg == "-h" || arg == "--help":
			return opts, errUsage
		case arg == "-debug" || arg == "--debug":
			opts.debug = true
			args = args[1:]
		case arg == "-once" || arg == "--once":
			opts.once = true
			args = args[1:]
		case arg == "-p" || arg == "--port" || strings.HasPrefix(arg, "-p=") || strings.HasPrefix(arg, "--port="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, err
			}
			port, err := strconv.Atoi(value)
			if err != nil || port <= 0 || port > 65535 {
				return opts, fmt.Errorf("invalid port: %q", value)
			}
			opts.port = port
			args = rest
		case arg == "-fps" || arg == "--fps" || strings.HasPrefix(arg, "-fps=") || strings.HasPrefix(arg, "--fps="):
			value, rest, err := optionValue(arg, args)
			if err != nil {
				return opts, err
			}
			fps, err := strconv.Atoi(value)
			if err != nil || fps <= 0 || fps > maxFPS {
				return opts, fmt.Errorf("invalid fps: %q", value)
			}
			opts.fps = fps
			args = rest
		default:
			return opts, fmt.Errorf("unknown option: %s", arg)
		}
	}

	return opts, nil
}

func optionValue(arg string, args []string) (string, []string, error) {
	if cut := strings.IndexByte(arg, '='); cut >= 0 {
		value := arg[cut+1:]
		if value == "" {
			return "", nil, fmt.Errorf("missing value for %s", arg[:cut])
		}
		return value, args[1:], nil
	}
	if len(args) < 2 {
		return "", nil, fmt.Errorf("missing value for %s", arg)
	}
	return args[1], args[2:], nil
}

func printUsage() {
	fmt.Println("Usage: vncsrv [-p PORT] [-fps FPS] [-once] [-debug]")
	fmt.Printf("Defaults: port=%d fps=%d\n", defaultPort, defaultFPS)
	fmt.Printf("Limits: fps=1..%d\n", maxFPS)
}

func serveClient(conn *rfb.Conn, stream *screenStream, fps int) {
	if conn == nil {
		return
	}
	debugLogf("serveClient: wait for RFB ready")
	select {
	case <-conn.Ready:
		debugLogf("serveClient: RFB ready")
	case <-conn.Done:
		debugLogf("serveClient: connection closed before ready")
		return
	}

	frameCount := 1
	logFrameProgress(frameCount, "capture start")
	if !stream.capture() {
		log.Printf("serveClient: first frame capture failed")
		return
	}
	logFrameProgress(frameCount, "capture done")
	logConnProgress(conn, stream, frameCount)
	logFirstFrame(stream)
	feedFrame(conn, stream.currentUpdate())
	logFrameProgress(frameCount, "feed queued")

	delay := time.Second / time.Duration(fps)
	for {
		select {
		case <-conn.Done:
			debugLogf("serveClient: connection done")
			return
		case _, ok := <-conn.Event:
			if !ok {
				debugLogf("serveClient: event channel closed")
				return
			}
		default:
		}

		time.Sleep(delay)
		select {
		case <-conn.Done:
			debugLogf("serveClient: connection done")
			return
		default:
		}
		frameCount++
		logFrameProgress(frameCount, "capture start")
		if !stream.capture() {
			log.Printf("serveClient: periodic capture failed")
			return
		}
		logFrameProgress(frameCount, "capture done")
		logConnProgress(conn, stream, frameCount)
		feedFrame(conn, stream.currentUpdate())
		logFrameProgress(frameCount, "feed queued")
	}
}

func logFrameProgress(frame int, stage string) {
	if !debugLogs {
		return
	}
	if frame <= 12 || frame%16 == 0 {
		log.Printf("serveClient: frame #%d %s", frame, stage)
	}
}

func logConnProgress(conn *rfb.Conn, stream *screenStream, frame int) {
	if !debugLogs {
		return
	}
	if frame > 12 && frame%8 != 0 {
		return
	}
	reqs, pushes, empties, publishes, replaces := conn.DebugCounters()
	log.Printf(
		"serveClient: state frame=%d requests=%d pushes=%d empties=%d publishes=%d replaces=%d sig=%08x",
		frame, reqs, pushes, empties, publishes, replaces, frameSignature(stream),
	)
}

func newScreenStream(width int, height int) *screenStream {
	frames := make([]frameSlot, 3)
	for i := range frames {
		debugLogf("newScreenStream: alloc rgba(slot=%d) / bytes=%d", i, width*height*4)
		frames[i].frame, frames[i].rgba = newFrameBuffer(width, height)
		debugLogf("newScreenStream: alloc rgba(slot=%d) ok", i)
	}
	stream := &screenStream{
		width:      width,
		height:     height,
		frames:     frames,
		currentIdx: -1,
	}
	debugLogf("newScreenStream: base stream ready")
	debugLogf("newScreenStream: query graphics bpp")
	stream.bpp = kos.GraphicsBitsPerPixel()
	debugLogf("newScreenStream: graphics bpp=%d", stream.bpp)
	debugLogf("newScreenStream: query graphics pitch")
	stream.pitch = kos.GraphicsBytesPerLine()
	debugLogf("newScreenStream: graphics pitch=%d", stream.pitch)
	if (stream.bpp == 24 || stream.bpp == 32) && stream.pitch >= width*(stream.bpp/8) {
		debugLogf("newScreenStream: alloc raw / bytes=%d", stream.pitch*height)
		stream.raw = make([]byte, stream.pitch*height)
		debugLogf("newScreenStream: alloc raw ok")
		log.Printf("Capture path: gs direct / bpp=%d / pitch=%d", stream.bpp, stream.pitch)
	} else {
		debugLogf("newScreenStream: alloc bgr / bytes=%d", width*height*3)
		stream.bgr = make([]byte, width*height*3)
		debugLogf("newScreenStream: alloc bgr ok")
		log.Printf("Capture path: syscall36 fallback / bpp=%d / pitch=%d", stream.bpp, stream.pitch)
	}
	return stream
}

func (stream *screenStream) capture() bool {
	slot := stream.captureSlot()
	if slot == nil || slot.frame == nil || slot.rgba == nil {
		return false
	}
	if len(stream.raw) != 0 {
		if !stream.captureDirect(slot) {
			return false
		}
		slot.rect = stream.computeDirtyRect(slot.rgba)
		stream.commitCapture()
		return true
	}
	if !kos.ReadScreenArea(stream.bgr, stream.width, stream.height, 0, 0) {
		return false
	}

	slot.frame.Lock()
	convertBGRToRGBA(slot.rgba.Pix, stream.bgr)
	slot.frame.Unlock()
	slot.rect = stream.computeDirtyRect(slot.rgba)
	stream.commitCapture()
	return true
}

func (stream *screenStream) captureDirect(slot *frameSlot) bool {
	if !kos.CopyGraphicsBuffer(stream.raw, 0) {
		return false
	}
	stream.detectPixelOrder()

	slot.frame.Lock()
	switch stream.bpp {
	case 24:
		convertRaw24ToRGBA(slot.rgba.Pix, stream.raw, stream.width, stream.height, stream.pitch, stream.pixelOrder)
	case 32:
		convertRaw32ToRGBA(slot.rgba.Pix, stream.raw, stream.width, stream.height, stream.pitch, stream.pixelOrder)
	default:
		slot.frame.Unlock()
		return false
	}
	slot.frame.Unlock()
	return true
}

func (stream *screenStream) captureSlot() *frameSlot {
	if stream == nil || len(stream.frames) == 0 {
		return nil
	}
	return &stream.frames[stream.captureIdx]
}

func (stream *screenStream) commitCapture() {
	if stream == nil || len(stream.frames) == 0 {
		return
	}
	stream.currentIdx = stream.captureIdx
	stream.hasCurrent = true
	stream.captureIdx++
	if stream.captureIdx >= len(stream.frames) {
		stream.captureIdx = 0
	}
}

func (stream *screenStream) currentFrame() *rfb.LockableImage {
	if stream == nil || !stream.hasCurrent || stream.currentIdx < 0 || stream.currentIdx >= len(stream.frames) {
		return nil
	}
	return stream.frames[stream.currentIdx].frame
}

func (stream *screenStream) currentUpdate() *rfb.FrameUpdate {
	if stream == nil || !stream.hasCurrent || stream.currentIdx < 0 || stream.currentIdx >= len(stream.frames) {
		return nil
	}
	slot := &stream.frames[stream.currentIdx]
	return &rfb.FrameUpdate{
		Frame: slot.frame,
		Rect:  slot.rect,
	}
}

func (stream *screenStream) currentRGBA() *image.RGBA {
	if stream == nil || !stream.hasCurrent || stream.currentIdx < 0 || stream.currentIdx >= len(stream.frames) {
		return nil
	}
	return stream.frames[stream.currentIdx].rgba
}

func newFrameBuffer(width int, height int) (*rfb.LockableImage, *image.RGBA) {
	rgbaPix := make([]byte, width*height*4)
	rgba := &image.RGBA{
		Pix:    rgbaPix,
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}
	return &rfb.LockableImage{Img: rgba}, rgba
}

func (stream *screenStream) detectPixelOrder() {
	if stream == nil || len(stream.raw) == 0 || stream.pixelOrder != pixelOrderUnknown {
		return
	}
	if stream.bpp != 24 && stream.bpp != 32 {
		return
	}

	samples := [][2]int{
		{0, 0},
		{stream.width / 2, stream.height / 2},
		{stream.width - 1, stream.height - 1},
		{stream.width / 3, stream.height / 3},
	}

	rgbScore := 0
	bgrScore := 0
	for _, sample := range samples {
		x, y := sample[0], sample[1]
		if x < 0 || y < 0 || x >= stream.width || y >= stream.height {
			continue
		}
		offset := y*stream.pitch + x*(stream.bpp/8)
		if offset+2 >= len(stream.raw) {
			continue
		}

		color := kos.GetPixelColorFromScreenRaw(y*stream.width + x)
		rr := byte(color >> 16)
		gg := byte(color >> 8)
		bb := byte(color)

		if stream.raw[offset] == rr && stream.raw[offset+1] == gg && stream.raw[offset+2] == bb {
			rgbScore++
		}
		if stream.raw[offset] == bb && stream.raw[offset+1] == gg && stream.raw[offset+2] == rr {
			bgrScore++
		}
	}

	switch {
	case rgbScore > bgrScore:
		stream.pixelOrder = pixelOrderRGB
	case bgrScore > 0:
		stream.pixelOrder = pixelOrderBGR
	default:
		stream.pixelOrder = pixelOrderBGR
	}
	debugLogf("captureDirect: detected pixel order=%s (rgb=%d bgr=%d)", stream.pixelOrder, rgbScore, bgrScore)
}

func convertBGRToRGBA(dst []byte, src []byte) {
	di := 0
	for si := 0; si+2 < len(src) && di+3 < len(dst); si, di = si+3, di+4 {
		dst[di] = src[si+2]
		dst[di+1] = src[si+1]
		dst[di+2] = src[si]
		dst[di+3] = 0xff
	}
}

func convertRaw24ToRGBA(dst []byte, src []byte, width int, height int, pitch int, order pixelOrder) {
	di := 0
	for y := 0; y < height; y++ {
		row := src[y*pitch:]
		for x := 0; x < width; x++ {
			si := x * 3
			switch order {
			case pixelOrderRGB:
				dst[di] = row[si]
				dst[di+1] = row[si+1]
				dst[di+2] = row[si+2]
			default:
				dst[di] = row[si+2]
				dst[di+1] = row[si+1]
				dst[di+2] = row[si]
			}
			dst[di+3] = 0xff
			di += 4
		}
	}
}

func convertRaw32ToRGBA(dst []byte, src []byte, width int, height int, pitch int, order pixelOrder) {
	di := 0
	for y := 0; y < height; y++ {
		row := src[y*pitch:]
		for x := 0; x < width; x++ {
			si := x * 4
			switch order {
			case pixelOrderRGB:
				dst[di] = row[si]
				dst[di+1] = row[si+1]
				dst[di+2] = row[si+2]
			default:
				dst[di] = row[si+2]
				dst[di+1] = row[si+1]
				dst[di+2] = row[si]
			}
			dst[di+3] = 0xff
			di += 4
		}
	}
}

func feedFrame(conn *rfb.Conn, update *rfb.FrameUpdate) {
	if conn == nil || update == nil || update.Frame == nil {
		return
	}
	conn.PublishFrame(update)
}

func logFirstFrame(stream *screenStream) {
	if !debugLogs {
		return
	}
	current := stream.currentRGBA()
	if current == nil || len(current.Pix) < 4 {
		return
	}

	samples := 64
	if pixels := len(current.Pix) / 4; pixels < samples {
		samples = pixels
	}
	nonBlack := 0
	for i := 0; i < samples; i++ {
		di := i * 4
		if current.Pix[di] != 0 || current.Pix[di+1] != 0 || current.Pix[di+2] != 0 {
			nonBlack++
		}
	}

	center := ((stream.height / 2) * current.Stride) + ((stream.width / 2) * 4)
	if center+3 >= len(current.Pix) {
		center = 0
	}

	log.Printf(
		"First frame: nonblack=%d/%d top-left=%02x%02x%02x center=%02x%02x%02x",
		nonBlack, samples,
		current.Pix[0], current.Pix[1], current.Pix[2],
		current.Pix[center], current.Pix[center+1], current.Pix[center+2],
	)
}

func frameSignature(stream *screenStream) uint32 {
	current := stream.currentRGBA()
	if stream == nil || current == nil || len(current.Pix) < 4 {
		return 0
	}
	points := [][2]int{
		{0, 0},
		{stream.width / 2, 0},
		{stream.width - 1, 0},
		{0, stream.height / 2},
		{stream.width / 2, stream.height / 2},
		{stream.width - 1, stream.height / 2},
		{0, stream.height - 1},
		{stream.width / 2, stream.height - 1},
		{stream.width - 1, stream.height - 1},
		{stream.width / 3, stream.height / 3},
		{2 * stream.width / 3, stream.height / 3},
		{stream.width / 3, 2 * stream.height / 3},
		{2 * stream.width / 3, 2 * stream.height / 3},
	}
	var sig uint32 = 2166136261
	for _, point := range points {
		x, y := point[0], point[1]
		if x < 0 || y < 0 || x >= stream.width || y >= stream.height {
			continue
		}
		di := y*current.Stride + x*4
		if di+2 >= len(current.Pix) {
			continue
		}
		color := uint32(current.Pix[di])<<16 |
			uint32(current.Pix[di+1])<<8 |
			uint32(current.Pix[di+2])
		sig ^= color + 0x9e3779b9 + (sig << 6) + (sig >> 2)
	}
	return sig
}

func (stream *screenStream) computeDirtyRect(next *image.RGBA) image.Rectangle {
	if stream == nil || next == nil {
		return image.Rectangle{}
	}
	prev := stream.currentRGBA()
	full := image.Rect(0, 0, stream.width, stream.height)
	if prev == nil || prev.Stride != next.Stride || len(prev.Pix) != len(next.Pix) {
		return full
	}

	minX, minY := stream.width, stream.height
	maxX, maxY := -1, -1
	for y := 0; y < stream.height; y++ {
		rowStart := y * next.Stride
		firstX := -1
		lastX := -1
		for x := 0; x < stream.width; x++ {
			di := rowStart + x*4
			if prev.Pix[di] == next.Pix[di] &&
				prev.Pix[di+1] == next.Pix[di+1] &&
				prev.Pix[di+2] == next.Pix[di+2] {
				continue
			}
			if firstX < 0 {
				firstX = x
			}
			lastX = x
		}
		if firstX < 0 {
			continue
		}
		if firstX < minX {
			minX = firstX
		}
		if lastX > maxX {
			maxX = lastX
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	if maxX < 0 || maxY < 0 {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

func addrString(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	return addr.String()
}

func waitForExit(console kos.Console) {
	if console.SupportsInput() {
		fmt.Println("Press any key to close.")
		console.Getch()
	}
}
