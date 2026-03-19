package ui

import "kos"

const (
	windowClientLeft   = 5
	windowClientRight  = 4
	windowClientBottom = 4
)

const (
	DefaultWindowX      = 0
	DefaultWindowY      = 0
	DefaultWindowWidth  = 320
	DefaultWindowHeight = 240
	DefaultWindowTitle  = "Window"
)

type Window struct {
	X      int
	Y      int
	Width  int
	Height int
	Title  string
	Style  Style

	Background kos.Color
	OnClose    func()
	OnResize   func(Rect)
	// ImplicitDirty scans nodes for Dirty() changes each frame.
	// Leave enabled for compatibility; disable for explicit invalidation.
	ImplicitDirty bool
	// LockRenderList disables render-list invalidation on element visual changes.
	// Intended for headless stress runs to avoid expensive rebuilds.
	LockRenderList bool
	// DisableNodeTiming skips per-node timing in drawRenderList.
	DisableNodeTiming bool
	primary           bool

	nodes                   []Node
	canvas                  *Canvas
	client                  Rect
	mouseDown               Node
	mouseHover              Node
	lastMouseX              int
	lastMouseY              int
	lastMouseValid          bool
	hoverDirty              bool
	focused                 Node
	running                 bool
	threadSlot              int
	threadSlotSet           bool
	threadSlotRetry         uint8
	prevMouseButtons        kos.MouseButtonInfo
	awaitingPress           bool
	lastMouseInteractive    bool
	pendingEvent            kos.EventType
	dirty                   Rect
	dirtySet                bool
	visualDirtyOnly         bool
	presentRect             Rect
	presentRectSet          bool
	lastBackground          kos.Color
	lastGCPollAt            uint32
	lastEventAt             uint32
	scrollY                 int
	drawnScrollY            int
	scrollMaxY              int
	scrollDragActive        bool
	scrollDragOffset        int
	scrollRedraw            bool
	translateBlits          []translateBlitOp
	propertyState           windowPropertyState
	displayState            windowDisplayState
	frameState              windowFrameState
	frameStateActive        bool
	caretBlinkResetAt       uint32
	caretBlinkVisibleSet    bool
	caretBlinkVisibleCached bool
	layoutDirty             bool
	renderList              []renderItem
	renderListValid         bool
	backgroundCache         *Canvas
	backgroundCacheKey      styleVisualKey
	backgroundCacheRect     Rect
	tinyglNodes             []*Element
	hitGrid                 hitTestGrid
	hitGridValid            bool
	allNodes                []Node
	nodeBounds              map[Node]Rect
	renderIndex             map[Node]int
	renderVisited           map[Node]struct{}
	dirtyCandidates         map[Node]struct{}
	dirtyList               []Node
	dirtyPlanNodes          []Node
	renderVisitGen          uint32
	layoutVisitGen          uint32
	dirtyQueueGen           uint32
}

type FrameStats struct {
	TotalNs      uint64
	DrawNs       uint64
	ClearNs      uint64
	NodesNs      uint64
	BlitNs       uint64
	LayoutNs     uint64
	RenderListNs uint64
}

type MouseDebugEvent struct {
	X            int
	Y            int
	Buttons      kos.MouseButtonInfo
	Held         kos.MouseButtonInfo
	LeftPressed  bool
	LeftReleased bool
	LeftHeld     bool
	Hovered      bool
	MouseDown    bool
}

var windowStartCount int

// MouseEventThrottleMs caps mouse-move processing rate when no buttons or scroll
// are active. Set to 0 to disable throttling.
var MouseEventThrottleMs uint32 = 0

// WindowPollRuntimeGC enables runtime GC polling during the window event loop.
// Disable for profiling if you want to isolate event-loop overhead.
var WindowPollRuntimeGC = true

// WindowGCPollIntervalMs throttles how often GC is polled while idle.
// 0 means poll on every idle wait.
var WindowGCPollIntervalMs uint32 = 0

// WindowGCPollIdleMs delays GC polling until the window has been idle for at
// least this many milliseconds. 0 means no idle delay.
var WindowGCPollIdleMs uint32 = 0

// WindowScrollYEnabled enables vertical scrolling for the window body.
var WindowScrollYEnabled = true

// WindowGCPollActiveIntervalMs throttles GC polling while the window is
// processing active events. 0 disables active polling.
var WindowGCPollActiveIntervalMs uint32 = 0

// WindowEnableTinyGL enables TinyGL rendering for tinygl elements.
// Disabled by default to avoid extra overhead in the UI loop.
var WindowEnableTinyGL = false

// WindowTinyGLRedrawOnDirty controls whether TinyGL elements redraw when their
// bounds intersect dirty regions. When false, TinyGL redraws only on explicit
// MarkTinyGLDirty calls or when its rect changes.
var WindowTinyGLRedrawOnDirty = true

// WindowCaretBlinkMs controls the caret blink period for text inputs.
// 0 disables blinking (caret always visible).
var WindowCaretBlinkMs uint32 = 1000

// DebugStartHook is an optional callback invoked once per window after the
// initial redraw. Intended for diagnostics only.
var DebugStartHook func(window *Window)

// DebugMouseHook is an optional callback invoked for mouse events on active
// windows. Intended for diagnostics only.
var DebugMouseHook func(window *Window, event MouseDebugEvent)
