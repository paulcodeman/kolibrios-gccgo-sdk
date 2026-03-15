package main

import (
	"os"
	"strconv"
	"strings"

	"kos"
	"ui"
	"ui/elements"
)

const (
	stressWindowTitle  = "UI Stress Test"
	stressWindowX      = 240
	stressWindowY      = 120
	stressWindowWidth  = 820
	stressWindowHeight = 640

	gridRows       = 38
	gridCols       = 20
	includeLabels  = true
	labelsAsDiv    = false
	labelsOnly     = true
	updatesPerTick = 40
	tickTimeoutCS  = 3

	uistressDebugPort = 0x402

	uistressHeadlessMarker = "/FD/1/UISTRESS.AUTO"
	uistressReportPath     = "/FD/1/UISTRESS.TXT"
	uistressDefaultFrames  = 300
	uistressMaxFrames      = 2000
)

type headlessConfig struct {
	frames        int
	updates       int
	noDisk        bool
	debug         bool
	layout        bool
	full          bool
	slowMs        int
	slowNs        int
	nodeTiming    bool
	fast          bool
	noShadow      bool
	noRadius      bool
	noGradient    bool
	noText        bool
	noTextDraw    bool
	noBorder      bool
	noTextShadow  bool
	noCache       bool
	gcPoll        bool
	heapStats     bool
	drawOnly      bool
	traceRow      int
	traceCol      int
	traceCell     bool
	traceStride   int
	traceRangeLo  int
	traceRangeHi  int
	renderOnlyIdx int
}

var uistressDebugPortReserved bool
var uistressAllowDebugNoReserve bool

type App struct {
	window                *ui.Window
	values                []*ui.Element
	bars                  []*ui.Element
	tick                  uint32
	next                  int
	headless              bool
	maxFrames             int
	frames                int
	done                  bool
	updatesPerTick        int
	headlessNoDisk        bool
	headlessLayout        bool
	headlessFull          bool
	headlessGCPoll        bool
	headlessHeapStats     bool
	headlessDebug         bool
	drawOnly              bool
	drawOnlyPrimed        bool
	prevHeapAllocCount    uint32
	prevHeapAllocBytes    uint32
	prevHeapFreeCount     uint32
	prevHeapReallocCount  uint32
	prevHeapReallocBytes  uint32
	prevGCAllocCount      uint32
	prevGCAllocBytes      uint32
	prevGCCollectionCount uint32
	gcPollNs              uint64
	sumTotal              uint32
	sumClear              uint32
	sumDraw               uint32
	sumNodes              uint32
	sumBlit               uint32
	maxTotal              uint32
	maxClear              uint32
	maxDraw               uint32
	maxNodes              uint32
	maxBlit               uint32
}

func Run() {
	app := NewApp()
	app.Run()
}

func makeLabel(text string) *ui.Element {
	if labelsAsDiv {
		el := ui.CreateBox()
		el.Text = text
		return el
	}
	return elements.Label(text)
}

func NewApp() *App {
	headless := headlessModeRequested()
	cfg := headlessConfig{}
	if headless {
		cfg = headlessConfigFromMarker()
	}
	traceCell := false
	traceRow := -1
	traceCol := -1
	if headless && cfg.traceCell {
		traceCell = true
		traceRow = cfg.traceRow
		traceCol = cfg.traceCol
	}

	screenW, screenH := kos.ScreenSize()
	width := stressWindowWidth
	height := stressWindowHeight
	if screenW > 0 && width > screenW-20 {
		width = screenW - 20
		if width < 1 {
			width = 1
		}
	}
	if screenH > 0 && height > screenH-40 {
		height = screenH - 40
		if height < 1 {
			height = 1
		}
	}
	x := stressWindowX
	y := stressWindowY
	if screenW > 0 && x+width > screenW {
		x = (screenW - width) / 2
		if x < 0 {
			x = 0
		}
	}
	if screenH > 0 && y+height > screenH {
		y = (screenH - height) / 2
		if y < 0 {
			y = 0
		}
	}

	window := ui.NewWindow(x, y, width, height, stressWindowTitle)
	window.ImplicitDirty = false

	apply := func(element *ui.Element, update func(*ui.Style)) {
		element.UpdateStyle(update)
	}

	root := ui.CreateBox()
	apply(root, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(12)
	})

	if includeLabels {
		header := makeLabel("UI stress test: auto updates + render timing")
		apply(header, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetForeground(ui.White)
			style.SetPadding(6, 10)
			style.SetMargin(0, 0, 10, 0)
			style.SetBorderRadius(8)
			style.SetBackground(ui.Navy)
		})
		root.Append(header)

		legend := makeLabel("Each tick updates a subset of elements; timing goes to QEMU debug console")
		apply(legend, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetForeground(ui.Gray)
			style.SetMargin(0, 0, 10, 0)
		})
		root.Append(legend)
	}

	grid := ui.CreateBox()
	apply(grid, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
	})

	values := make([]*ui.Element, 0, gridRows*gridCols)
	bars := make([]*ui.Element, 0, gridRows*gridCols)

	for r := 0; r < gridRows; r++ {
		if labelsOnly {
			for c := 0; c < gridCols; c++ {
				label := makeLabel("Cell " + strconv.Itoa(r) + ":" + strconv.Itoa(c))
				apply(label, func(style *ui.Style) {
					style.SetDisplay(ui.DisplayBlock)
					style.SetForeground(ui.Navy)
					style.SetMargin(0, 0, 2, 0)
				})
				if traceCell && r == traceRow && c == traceCol {
					ui.DebugTraceElement = label
				}
				grid.Append(label)
			}
			continue
		}

		row := ui.CreateBox()
		apply(row, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
		})
		for c := 0; c < gridCols; c++ {
			card := ui.CreateBox()
			apply(card, func(style *ui.Style) {
				style.SetDisplay(ui.DisplayInlineBlock)
				style.SetPadding(6)
				style.SetMargin(0, 6, 6, 0)
				style.SetBorderRadius(6)
				style.SetBorderColor(ui.Silver)
				style.SetBorderWidth(1)
				style.SetBackground(ui.White)
			})

			var label *ui.Element
			if includeLabels {
				label = makeLabel("Cell " + strconv.Itoa(r) + ":" + strconv.Itoa(c))
				apply(label, func(style *ui.Style) {
					style.SetDisplay(ui.DisplayBlock)
					style.SetForeground(ui.Navy)
					style.SetMargin(0, 0, 2, 0)
				})
			}
			if traceCell && r == traceRow && c == traceCol && label != nil {
				ui.DebugTraceElement = label
			}

			value := makeLabel("0")
			apply(value, func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetPadding(2, 6)
				style.SetBorderRadius(4)
				style.SetBackground(ui.Aqua)
				style.SetForeground(ui.Navy)
			})
			if traceCell && r == traceRow && c == traceCol && ui.DebugTraceElement == nil {
				ui.DebugTraceElement = value
			}

			bar := ui.CreateBox()
			apply(bar, func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetHeight(6)
				style.SetMargin(4, 0, 0, 0)
				style.SetBackground(ui.Lime)
				style.SetBorderRadius(3)
				style.SetWidth(60)
			})

			if label != nil {
				card.Append(label)
			}
			card.Append(value)
			card.Append(bar)
			row.Append(card)

			values = append(values, value)
			bars = append(bars, bar)
		}
		grid.Append(row)
	}

	var inputLabel *ui.Element
	var input *ui.Element
	var textareaLabel *ui.Element
	var textarea *ui.Element
	if !labelsOnly {
		if includeLabels {
			inputLabel = makeLabel("Input")
			apply(inputLabel, func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetMargin(6, 0, 2, 0)
				style.SetForeground(ui.Gray)
			})
		}
		input = elements.Input("Typing disabled in this test")
		apply(input, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetBorderRadius(6)
		})

		if includeLabels {
			textareaLabel = makeLabel("Textarea")
			apply(textareaLabel, func(style *ui.Style) {
				style.SetDisplay(ui.DisplayBlock)
				style.SetMargin(6, 0, 2, 0)
				style.SetForeground(ui.Gray)
			})
		}
		textarea = elements.Textarea("Stress test textarea\nLine 2\nLine 3\nLine 4")
		apply(textarea, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayBlock)
			style.SetHeight(70)
			style.SetBorderRadius(6)
		})
	}

	root.Append(grid)
	if inputLabel != nil {
		root.Append(inputLabel)
	}
	if input != nil {
		root.Append(input)
	}
	if textareaLabel != nil {
		root.Append(textareaLabel)
	}
	if textarea != nil {
		root.Append(textarea)
	}

	window.Append(root)

	maxFrames := 0
	updates := updatesPerTick
	headlessNoDisk := false
	headlessLayout := true
	headlessFull := true
	headlessGCPoll := true
	headlessHeapStats := false
	headlessDebug := false
	drawOnly := false
	slowThresholdNs := uint64(0)
	if headless {
		maxFrames = cfg.frames
		if cfg.updates >= 0 {
			updates = cfg.updates
		}
		headlessNoDisk = cfg.noDisk
		headlessLayout = cfg.layout
		headlessFull = cfg.full
		headlessGCPoll = cfg.gcPoll
		headlessHeapStats = cfg.heapStats
		headlessDebug = cfg.debug
		drawOnly = cfg.drawOnly
		if cfg.slowMs > 0 {
			slowThresholdNs = uint64(cfg.slowMs) * 1000000
		} else if cfg.slowNs > 0 {
			slowThresholdNs = uint64(cfg.slowNs)
		}
		if cfg.debug || cfg.traceCell || cfg.traceStride > 0 || cfg.traceRangeLo >= 0 || slowThresholdNs > 0 {
			uistressAllowDebugNoReserve = true
		}
		if cfg.traceCell {
			enableUIDebugTrace()
			if ui.DebugTrace != nil {
				ui.DebugTrace("trace.cell." + strconv.Itoa(cfg.traceRow) + "." + strconv.Itoa(cfg.traceCol))
			}
		}
		if cfg.traceStride > 0 {
			enableUIDebugTrace()
			ui.DebugTraceRenderStride = cfg.traceStride
			if ui.DebugTrace != nil {
				ui.DebugTrace("trace.render.stride." + strconv.Itoa(cfg.traceStride))
			}
		}
		if cfg.traceRangeLo >= 0 && cfg.traceRangeHi >= 0 {
			enableUIDebugTrace()
			ui.DebugTraceRangeStart = cfg.traceRangeLo
			ui.DebugTraceRangeEnd = cfg.traceRangeHi
			if ui.DebugTraceElement == nil {
				ui.DebugTraceElement = &ui.Element{}
			}
			if ui.DebugTrace != nil {
				ui.DebugTrace("trace.render.range." + strconv.Itoa(cfg.traceRangeLo) + "." + strconv.Itoa(cfg.traceRangeHi))
			}
		}
		if cfg.renderOnlyIdx >= 0 {
			ui.DebugRenderOnlyIndex = cfg.renderOnlyIdx
		}
		if slowThresholdNs > 0 {
			headlessDebug = true
			enableUIDebugTrace()
			ui.DebugSlowNodeThresholdNs = slowThresholdNs
			if ui.DebugTrace != nil {
				ui.DebugTrace("slow.threshold.ns." + strconv.FormatUint(slowThresholdNs, 10))
			}
		}
		if !cfg.nodeTiming {
			window.DisableNodeTiming = true
		}
		if headlessFull {
			window.LockRenderList = true
		}
		if cfg.debug {
			enableUIDebugTrace()
		}
		if cfg.fast {
			ui.FastNoShadows = true
			ui.FastNoRadius = true
			ui.FastNoGradients = true
			ui.FastNoTextShadow = true
		}
		if cfg.noShadow {
			ui.FastNoShadows = true
		}
		if cfg.noRadius {
			ui.FastNoRadius = true
		}
		if cfg.noGradient {
			ui.FastNoGradients = true
		}
		if cfg.noText {
			ui.FastNoText = true
		}
		if cfg.noTextDraw {
			ui.FastNoTextDraw = true
		}
		if cfg.noBorder {
			ui.FastNoBorders = true
		}
		if cfg.noTextShadow {
			ui.FastNoTextShadow = true
		}
		if cfg.noCache {
			ui.FastNoCache = true
		}
		if !headlessNoDisk {
			writeHeadlessStart()
		}
	}

	return &App{
		window:            window,
		values:            values,
		bars:              bars,
		headless:          headless,
		maxFrames:         maxFrames,
		updatesPerTick:    updates,
		headlessNoDisk:    headlessNoDisk,
		headlessLayout:    headlessLayout,
		headlessFull:      headlessFull,
		headlessGCPoll:    headlessGCPoll,
		headlessHeapStats: headlessHeapStats,
		headlessDebug:     headlessDebug,
		drawOnly:          drawOnly,
	}
}

func (app *App) Run() {
	if app.headless {
		for !app.done {
			if app.headlessDebug {
				app.debugTick("uistress tick_begin=", app.tick+1)
			}
			if app.headlessGCPoll {
				start := kos.UptimeNanoseconds()
				kos.PollRuntimeGCRaw()
				app.gcPollNs = kos.UptimeNanoseconds() - start
			}
			app.tick++
			app.updateTick()
			if app.headlessDebug {
				app.debugTick("uistress tick_end=", app.tick)
			}
			if app.done {
				return
			}
		}
		return
	}

	app.window.Redraw()

	for {
		event := kos.WaitEventFor(tickTimeoutCS)
		switch event {
		case kos.EventRedraw:
			app.window.Redraw()
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				return
			}
		}
		app.tick++
		app.updateTick()
		if app.done {
			return
		}
	}
}

func (app *App) updateTick() {
	if len(app.values) > 0 && !app.drawOnly {
		for i := 0; i < app.updatesPerTick; i++ {
			idx := app.next % len(app.values)
			value := app.values[idx]
			value.SetText(app.window, strconv.Itoa(int(app.tick)+i))

			bar := app.bars[idx%len(app.bars)]
			width := 30 + int((app.tick+uint32(i))%70)
			if !app.headless || app.headlessLayout {
				bar.SetWidth(width)
			}
			if (app.tick+uint32(i))%2 == 0 {
				bar.SetBackground(ui.Lime)
			} else {
				bar.SetBackground(ui.Yellow)
			}

			app.next++
		}
	}

	var stats ui.FrameStats
	if app.headless {
		if app.drawOnly {
			if app.drawOnlyPrimed {
				app.window.RenderListStats(&stats)
			} else {
				app.window.RenderStatsFull(&stats)
				app.drawOnlyPrimed = true
			}
		} else if app.headlessFull {
			app.window.RenderStatsFull(&stats)
		} else {
			app.window.RenderStats(&stats)
		}
	} else {
		app.window.RedrawContentStats(&stats)
	}
	app.debugFrame(app.tick, stats)
	app.recordStats(stats)
}

func (app *App) debugFrame(tick uint32, stats ui.FrameStats) {
	if !reserveDebugPort() {
		return
	}
	buf := make([]byte, 0, 220)
	buf = append(buf, "uistress frame="...)
	buf = appendUint32(buf, tick)
	buf = append(buf, " total="...)
	buf = appendUint32(buf, uint32(stats.TotalNs))
	if hi := uint32(stats.TotalNs >> 32); hi > 0 {
		buf = append(buf, " total_hi="...)
		buf = appendUint32(buf, hi)
	}
	buf = append(buf, " clear="...)
	buf = appendUint32(buf, uint32(stats.ClearNs))
	buf = append(buf, " draw="...)
	buf = appendUint32(buf, uint32(stats.DrawNs))
	if stats.LayoutNs > 0 {
		buf = append(buf, " layout="...)
		buf = appendUint32(buf, uint32(stats.LayoutNs))
	}
	if stats.RenderListNs > 0 {
		buf = append(buf, " renderlist="...)
		buf = appendUint32(buf, uint32(stats.RenderListNs))
	}
	buf = append(buf, " nodes="...)
	buf = appendUint32(buf, uint32(stats.NodesNs))
	buf = append(buf, " blit="...)
	buf = appendUint32(buf, uint32(stats.BlitNs))
	if app != nil && app.gcPollNs > 0 {
		buf = append(buf, " gc_poll_ns="...)
		buf = appendUint32(buf, uint32(clampUint64(app.gcPollNs)))
	}
	if app != nil && app.headlessHeapStats {
		allocCount, allocBytes, freeCount, reallocCount, reallocBytes, gcAllocCount, gcAllocBytes, gcCollections, gcLive, gcThreshold, gcPollRetry := app.heapCountersDelta()
		buf = append(buf, " heap_alloc="...)
		buf = appendUint32(buf, allocCount)
		buf = append(buf, " heap_bytes="...)
		buf = appendUint32(buf, allocBytes)
		buf = append(buf, " heap_free="...)
		buf = appendUint32(buf, freeCount)
		buf = append(buf, " heap_realloc="...)
		buf = appendUint32(buf, reallocCount)
		buf = append(buf, " heap_realloc_bytes="...)
		buf = appendUint32(buf, reallocBytes)
		buf = append(buf, " gc_alloc="...)
		buf = appendUint32(buf, gcAllocCount)
		buf = append(buf, " gc_bytes="...)
		buf = appendUint32(buf, gcAllocBytes)
		buf = append(buf, " gc_collections="...)
		buf = appendUint32(buf, gcCollections)
		buf = append(buf, " gc_live="...)
		buf = appendUint32(buf, gcLive)
		buf = append(buf, " gc_threshold="...)
		buf = appendUint32(buf, gcThreshold)
		buf = append(buf, " gc_poll_retry="...)
		buf = appendUint32(buf, gcPollRetry)
	}
	buf = append(buf, "\r\n"...)
	kos.WritePortString(uistressDebugPort, string(buf))
}

func (app *App) debugTick(prefix string, tick uint32) {
	if !reserveDebugPort() {
		return
	}
	buf := make([]byte, 0, len(prefix)+12)
	buf = append(buf, prefix...)
	buf = appendUint32(buf, tick)
	buf = append(buf, "\r\n"...)
	kos.WritePortString(uistressDebugPort, string(buf))
}

func (app *App) heapCountersDelta() (uint32, uint32, uint32, uint32, uint32, uint32, uint32, uint32, uint32, uint32, uint32) {
	if app == nil {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	}
	allocCount := kos.HeapAllocCountRaw()
	allocBytes := kos.HeapAllocBytesRaw()
	freeCount := kos.HeapFreeCountRaw()
	reallocCount := kos.HeapReallocCountRaw()
	reallocBytes := kos.HeapReallocBytesRaw()
	gcAllocCount := kos.GCAllocCountRaw()
	gcAllocBytes := kos.GCAllocBytesRaw()
	gcCollections := kos.GCCollectionCountRaw()
	gcLive := kos.GCLiveBytesRaw()
	gcThreshold := kos.GCThresholdRaw()
	gcPollRetry := kos.GCPollRetryRaw()

	dAllocCount := allocCount - app.prevHeapAllocCount
	dAllocBytes := allocBytes - app.prevHeapAllocBytes
	dFreeCount := freeCount - app.prevHeapFreeCount
	dReallocCount := reallocCount - app.prevHeapReallocCount
	dReallocBytes := reallocBytes - app.prevHeapReallocBytes
	dGCAllocCount := gcAllocCount - app.prevGCAllocCount
	dGCAllocBytes := gcAllocBytes - app.prevGCAllocBytes
	dGCCollections := gcCollections - app.prevGCCollectionCount

	app.prevHeapAllocCount = allocCount
	app.prevHeapAllocBytes = allocBytes
	app.prevHeapFreeCount = freeCount
	app.prevHeapReallocCount = reallocCount
	app.prevHeapReallocBytes = reallocBytes
	app.prevGCAllocCount = gcAllocCount
	app.prevGCAllocBytes = gcAllocBytes
	app.prevGCCollectionCount = gcCollections

	return dAllocCount, dAllocBytes, dFreeCount, dReallocCount, dReallocBytes, dGCAllocCount, dGCAllocBytes, dGCCollections, gcLive, gcThreshold, gcPollRetry
}

func appendUint32(buf []byte, value uint32) []byte {
	if value == 0 {
		return append(buf, '0')
	}
	var tmp [10]byte
	i := len(tmp)
	for value > 0 {
		q := value / 10
		r := value - q*10
		i--
		tmp[i] = byte('0' + r)
		value = q
	}
	return append(buf, tmp[i:]...)
}

func reserveDebugPort() bool {
	if uistressDebugPortReserved {
		return true
	}
	if !kos.ReservePorts(uistressDebugPort, uistressDebugPort) {
		if uistressAllowDebugNoReserve {
			uistressDebugPortReserved = true
			return true
		}
		return false
	}
	uistressDebugPortReserved = true
	return true
}

func enableUIDebugTrace() {
	if !reserveDebugPort() {
		return
	}
	ui.DebugTrace = func(label string) {
		kos.WritePortString(uistressDebugPort, "ui ")
		kos.WritePortString(uistressDebugPort, label)
		kos.WritePortString(uistressDebugPort, "\r\n")
	}
}

func headlessModeRequested() bool {
	_, err := os.Stat(uistressHeadlessMarker)
	if err == nil {
		return true
	}
	_, err = os.Stat("/UISTRESS.AUTO")
	return err == nil
}

func headlessConfigFromMarker() headlessConfig {
	cfg := headlessConfig{
		frames:        uistressDefaultFrames,
		updates:       -1,
		layout:        true,
		full:          true,
		slowMs:        0,
		slowNs:        0,
		nodeTiming:    true,
		gcPoll:        true,
		heapStats:     false,
		drawOnly:      false,
		traceRow:      -1,
		traceCol:      -1,
		traceCell:     false,
		traceStride:   0,
		traceRangeLo:  -1,
		traceRangeHi:  -1,
		renderOnlyIdx: -1,
	}
	data, err := os.ReadFile(uistressHeadlessMarker)
	if err != nil {
		data, err = os.ReadFile("/UISTRESS.AUTO")
		if err != nil {
			return cfg
		}
	}
	return parseHeadlessConfigText(string(data))
}

func parseHeadlessConfigText(text string) headlessConfig {
	cfg := headlessConfig{
		frames:        uistressDefaultFrames,
		updates:       -1,
		layout:        true,
		full:          true,
		slowMs:        0,
		slowNs:        0,
		nodeTiming:    true,
		gcPoll:        true,
		heapStats:     false,
		drawOnly:      false,
		traceRow:      -1,
		traceCol:      -1,
		traceCell:     false,
		traceStride:   0,
		traceRangeLo:  -1,
		traceRangeHi:  -1,
		renderOnlyIdx: -1,
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return cfg
	}
	if !strings.Contains(text, "=") {
		if frames, err := strconv.Atoi(text); err == nil {
			cfg.frames = clampHeadlessFrames(frames)
			return cfg
		}
	}

	for _, field := range strings.Fields(text) {
		if field == "" {
			continue
		}
		lower := strings.ToLower(field)
		if !strings.Contains(lower, "=") {
			switch lower {
			case "nodisk":
				cfg.noDisk = true
			case "debug":
				cfg.debug = true
			case "nolayout":
				cfg.layout = false
			case "dirty":
				cfg.full = false
			case "full":
				cfg.full = true
			case "notiming":
				cfg.nodeTiming = false
			case "fast":
				cfg.fast = true
			case "no_shadow":
				cfg.noShadow = true
			case "no_radius":
				cfg.noRadius = true
			case "no_gradient":
				cfg.noGradient = true
			case "no_text":
				cfg.noText = true
			case "no_text_draw", "no_text_output", "no_text_syscall":
				cfg.noTextDraw = true
			case "no_border":
				cfg.noBorder = true
			case "no_text_shadow":
				cfg.noTextShadow = true
			case "no_cache":
				cfg.noCache = true
			case "nogc", "no_gc", "no_gc_poll":
				cfg.gcPoll = false
			case "heap", "heap_stats", "runtime_stats":
				cfg.heapStats = true
			case "draw_only", "render_only", "renderlist_only":
				cfg.drawOnly = true
			}
			continue
		}
		parts := strings.SplitN(lower, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "frames":
			if frames, err := strconv.Atoi(value); err == nil {
				cfg.frames = clampHeadlessFrames(frames)
			}
		case "updates":
			if updates, err := strconv.Atoi(value); err == nil && updates >= 0 {
				cfg.updates = updates
			}
		case "slow_ms", "slow":
			if v, err := strconv.Atoi(value); err == nil && v > 0 {
				cfg.slowMs = v
			}
		case "slow_ns":
			if v, err := strconv.Atoi(value); err == nil && v > 0 {
				cfg.slowNs = v
			}
		case "trace_cell", "trace":
			if row, col, ok := parseHeadlessTraceCell(value); ok {
				cfg.traceRow = row
				cfg.traceCol = col
				cfg.traceCell = true
			}
		case "trace_row":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.traceRow = v
			}
		case "trace_col":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.traceCol = v
			}
		case "trace_stride", "trace_step":
			if v, err := strconv.Atoi(value); err == nil && v > 0 {
				cfg.traceStride = v
			}
		case "trace_range", "trace_idx_range":
			if lo, hi, ok := parseHeadlessTraceRange(value); ok {
				cfg.traceRangeLo = lo
				cfg.traceRangeHi = hi
			}
		case "trace_range_lo", "trace_range_start":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.traceRangeLo = v
			}
		case "trace_range_hi", "trace_range_end":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.traceRangeHi = v
			}
		case "render_only_idx", "draw_only_idx", "focus_idx":
			if v, err := strconv.Atoi(value); err == nil && v >= 0 {
				cfg.renderOnlyIdx = v
			}
		case "nodisk":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noDisk = ok
			}
		case "debug":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.debug = ok
			}
		case "layout":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.layout = ok
			}
		case "full":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.full = ok
			}
		case "dirty":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.full = !ok
			}
		case "fast":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.fast = ok
			}
		case "no_shadow":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noShadow = ok
			}
		case "no_radius":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noRadius = ok
			}
		case "no_gradient":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noGradient = ok
			}
		case "no_text":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noText = ok
			}
		case "no_text_draw":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noTextDraw = ok
			}
		case "no_text_output":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noTextDraw = ok
			}
		case "no_text_syscall":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noTextDraw = ok
			}
		case "no_border":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noBorder = ok
			}
		case "no_text_shadow":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noTextShadow = ok
			}
		case "no_cache":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.noCache = ok
			}
		case "gc_poll":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.gcPoll = ok
			}
		case "heap_stats":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.heapStats = ok
			}
		case "heap":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.heapStats = ok
			}
		case "runtime_stats":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.heapStats = ok
			}
		case "node_timing":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.nodeTiming = ok
			}
		case "timing":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.nodeTiming = ok
			}
		case "draw_only":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.drawOnly = ok
			}
		case "render_only":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.drawOnly = ok
			}
		case "renderlist_only":
			if ok, parsed := parseHeadlessBool(value); parsed {
				cfg.drawOnly = ok
			}
		}
	}

	if cfg.traceRow >= 0 && cfg.traceCol >= 0 {
		cfg.traceCell = true
	}
	if cfg.traceRangeLo >= 0 && cfg.traceRangeHi >= 0 && cfg.traceRangeHi < cfg.traceRangeLo {
		cfg.traceRangeLo = -1
		cfg.traceRangeHi = -1
	}
	return cfg
}

func parseHeadlessTraceCell(text string) (int, int, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, 0, false
	}
	idx := strings.IndexAny(text, ":,")
	if idx < 0 {
		return 0, 0, false
	}
	left := strings.TrimSpace(text[:idx])
	right := strings.TrimSpace(text[idx+1:])
	if left == "" || right == "" {
		return 0, 0, false
	}
	row, err := strconv.Atoi(left)
	if err != nil {
		return 0, 0, false
	}
	col, err := strconv.Atoi(right)
	if err != nil {
		return 0, 0, false
	}
	return row, col, true
}

func parseHeadlessTraceRange(text string) (int, int, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, 0, false
	}
	idx := strings.IndexAny(text, ":,")
	if idx < 0 {
		return 0, 0, false
	}
	left := strings.TrimSpace(text[:idx])
	right := strings.TrimSpace(text[idx+1:])
	if left == "" || right == "" {
		return 0, 0, false
	}
	lo, err := strconv.Atoi(left)
	if err != nil {
		return 0, 0, false
	}
	hi, err := strconv.Atoi(right)
	if err != nil {
		return 0, 0, false
	}
	return lo, hi, true
}

func parseHeadlessBool(text string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func clampHeadlessFrames(frames int) int {
	if frames < 1 {
		return uistressDefaultFrames
	}
	if frames > uistressMaxFrames {
		return uistressMaxFrames
	}
	return frames
}

func clampUint64(value uint64) uint32 {
	if value > 0xFFFFFFFF {
		return 0xFFFFFFFF
	}
	return uint32(value)
}

func addClamp(sum uint32, value uint32) uint32 {
	if sum > 0xFFFFFFFF-value {
		return 0xFFFFFFFF
	}
	return sum + value
}

func (app *App) recordStats(stats ui.FrameStats) {
	if !app.headless || app.done {
		return
	}
	total := clampUint64(stats.TotalNs)
	clear := clampUint64(stats.ClearNs)
	draw := clampUint64(stats.DrawNs)
	nodes := clampUint64(stats.NodesNs)
	blit := clampUint64(stats.BlitNs)

	app.sumTotal = addClamp(app.sumTotal, total)
	app.sumClear = addClamp(app.sumClear, clear)
	app.sumDraw = addClamp(app.sumDraw, draw)
	app.sumNodes = addClamp(app.sumNodes, nodes)
	app.sumBlit = addClamp(app.sumBlit, blit)

	if total > app.maxTotal {
		app.maxTotal = total
	}
	if clear > app.maxClear {
		app.maxClear = clear
	}
	if draw > app.maxDraw {
		app.maxDraw = draw
	}
	if nodes > app.maxNodes {
		app.maxNodes = nodes
	}
	if blit > app.maxBlit {
		app.maxBlit = blit
	}

	app.frames++
	if !app.headlessNoDisk {
		writeHeadlessHeartbeat(app.frames)
	}
	if app.maxFrames > 0 && app.frames >= app.maxFrames {
		app.emitHeadlessSummary()
		app.done = true
		kos.PowerOff()
	}
}

func (app *App) emitHeadlessSummary() {
	if app.frames == 0 {
		return
	}
	frames := uint32(app.frames)
	avgTotal := app.sumTotal / frames
	avgClear := app.sumClear / frames
	avgDraw := app.sumDraw / frames
	avgNodes := app.sumNodes / frames
	avgBlit := app.sumBlit / frames

	lines := []string{
		"uistress summary frames=" + strconv.Itoa(app.frames),
		"avg_total=" + strconv.FormatUint(uint64(avgTotal), 10) + " max_total=" + strconv.FormatUint(uint64(app.maxTotal), 10),
		"avg_clear=" + strconv.FormatUint(uint64(avgClear), 10) + " max_clear=" + strconv.FormatUint(uint64(app.maxClear), 10),
		"avg_draw=" + strconv.FormatUint(uint64(avgDraw), 10) + " max_draw=" + strconv.FormatUint(uint64(app.maxDraw), 10),
		"avg_nodes=" + strconv.FormatUint(uint64(avgNodes), 10) + " max_nodes=" + strconv.FormatUint(uint64(app.maxNodes), 10),
		"avg_blit=" + strconv.FormatUint(uint64(avgBlit), 10) + " max_blit=" + strconv.FormatUint(uint64(app.maxBlit), 10),
	}

	writeHeadlessLines(lines)
	if !app.headlessNoDisk {
		writeHeadlessReport(lines)
	}
}

func writeHeadlessLines(lines []string) {
	if !reserveDebugPort() {
		return
	}
	for _, line := range lines {
		kos.WritePortString(uistressDebugPort, line)
		kos.WritePortString(uistressDebugPort, "\r\n")
	}
}

func writeHeadlessReport(lines []string) {
	data := strings.Join(lines, "\r\n") + "\r\n"
	_ = os.WriteFile(uistressReportPath, []byte(data), 0o644)
}

func writeHeadlessStart() {
	_ = os.WriteFile("/FD/1/UISTRESS.START", []byte("ok\r\n"), 0o644)
}

func writeHeadlessHeartbeat(frames int) {
	_ = os.WriteFile("/FD/1/UISTRESS.HB", []byte(strconv.Itoa(frames)+"\r\n"), 0o644)
}
