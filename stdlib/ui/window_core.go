package ui

func NewWindow(x int, y int, width int, height int, title string) *Window {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	offset := nextWindowCascadeOffset(x, y, width, height)
	if offset != 0 {
		x += offset
		y += offset
	}
	style := Style{}
	style.SetLeft(x)
	style.SetTop(y)
	style.SetWidth(width)
	style.SetHeight(height)
	style.SetOverflow(OverflowAuto)
	window := &Window{
		X:             x,
		Y:             y,
		Width:         width,
		Height:        height,
		Title:         title,
		Style:         style,
		Background:    White,
		awaitingPress: true,
		ImplicitDirty: false,
	}
	window.client = windowClientRect(width, height)
	return window
}

func NewWindowDefault() *Window {
	return NewWindow(DefaultWindowX, DefaultWindowY, DefaultWindowWidth, DefaultWindowHeight, DefaultWindowTitle)
}

func (window *Window) Append(node Node) {
	if window == nil || node == nil {
		return
	}
	if aware, ok := node.(windowAware); ok && aware != nil {
		aware.setWindow(window)
	}
	window.nodes = append(window.nodes, node)
	window.layoutDirty = true
	window.renderListValid = false
	window.hoverDirty = true
	window.lastMouseValid = false
}

func (window *Window) Close() {
	if window == nil {
		return
	}
	if window.OnClose != nil {
		window.OnClose()
	}
	window.running = false
}

func (window *Window) ClientRect() Rect {
	if window == nil {
		return Rect{}
	}
	return window.client
}
