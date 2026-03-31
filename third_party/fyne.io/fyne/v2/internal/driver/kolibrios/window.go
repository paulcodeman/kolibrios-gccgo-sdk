//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package kolibrios

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	intdriver "fyne.io/fyne/v2/internal/driver"
	"surface"

	"kos"
)

const dragStartThreshold = 2.0

type window struct {
	title      string
	fullScreen bool
	fixedSize  bool
	shown      bool
	master     bool

	onClosed         func()
	onCloseIntercept func()

	canvas    *canvas
	clipboard fyne.Clipboard
	driver    *driverState
	menu      *fyne.MainMenu
	icon      fyne.Resource
	buffer    *surface.Buffer
	presented bool

	needsFullRedraw bool

	x int
	y int

	prevButtons kos.MouseButtonInfo
	mousePos    fyne.Position
	hovered     desktop.Hoverable
	pressed     fyne.CanvasObject

	dragCandidate fyne.Draggable
	dragStartPos  fyne.Position
	dragOffset    fyne.Position
	dragStartObj  fyne.Position
	dragged       fyne.Draggable
}

var _ fyne.Window = (*window)(nil)

func (w *window) Canvas() fyne.Canvas {
	return w.canvas
}

func (w *window) CenterOnScreen() {
}

func (w *window) Clipboard() fyne.Clipboard {
	return w.clipboard
}

func (w *window) Close() {
	if w.onClosed != nil {
		w.onClosed()
	}
	w.shown = false
	w.presented = false
	w.driver.removeWindow(w)
}

func (w *window) Content() fyne.CanvasObject {
	return w.canvas.Content()
}

func (w *window) FixedSize() bool {
	return w.fixedSize
}

func (w *window) FullScreen() bool {
	return w.fullScreen
}

func (w *window) Hide() {
	w.shown = false
	w.presented = false
	w.needsFullRedraw = true
	if w.driver.active == w {
		w.driver.active = nil
	}
}

func (w *window) Icon() fyne.Resource {
	if w.icon != nil {
		return w.icon
	}
	return fyne.CurrentApp().Icon()
}

func (w *window) MainMenu() *fyne.MainMenu {
	return w.menu
}

func (w *window) Padded() bool {
	return w.canvas.Padded()
}

func (w *window) RequestFocus() {
	w.shown = true
	w.needsFullRedraw = true
	w.driver.activate(w)
}

func (w *window) Resize(size fyne.Size) {
	w.canvas.Resize(size)
	w.needsFullRedraw = true
	if w.driver.active == w {
		w.render()
	}
}

func (w *window) SetContent(obj fyne.CanvasObject) {
	w.canvas.SetContent(obj)
	w.needsFullRedraw = true
	if w.driver.active == w {
		w.render()
	}
}

func (w *window) SetFixedSize(fixed bool) {
	w.fixedSize = fixed
}

func (w *window) SetIcon(icon fyne.Resource) {
	w.icon = icon
}

func (w *window) SetFullScreen(fullScreen bool) {
	w.fullScreen = fullScreen
}

func (w *window) SetMainMenu(menu *fyne.MainMenu) {
	w.menu = menu
}

func (w *window) SetMaster() {
	w.master = true
}

func (w *window) SetOnClosed(closed func()) {
	w.onClosed = closed
}

func (w *window) SetCloseIntercept(callback func()) {
	w.onCloseIntercept = callback
}

func (w *window) SetPadded(padded bool) {
	w.canvas.SetPadded(padded)
}

func (w *window) SetTitle(title string) {
	w.title = title
	w.needsFullRedraw = true
	if w.driver.active == w {
		w.render()
	}
}

func (w *window) Show() {
	w.shown = true
	w.needsFullRedraw = true
	w.RequestFocus()
	w.render()
}

func (w *window) ShowAndRun() {
	w.Show()
	w.driver.Run()
}

func (w *window) Title() string {
	return w.title
}

func (w *window) render() {
	if !w.shown {
		return
	}
	size := w.canvas.Size()
	width := int(size.Width)
	height := int(size.Height)
	if width <= 0 || height <= 0 {
		width = 1
		height = 1
		w.canvas.Resize(fyne.NewSize(1, 1))
	}
	if w.buffer == nil {
		w.buffer = surface.NewBuffer(width, height)
	} else {
		w.buffer.Resize(width, height)
	}
	image := surface.NewImageFromSource(w.canvas.Capture())
	if image == nil {
		return
	}
	w.buffer.DrawImageRect(surface.Rect{Width: width, Height: height}, image)
	presenter := surface.NewPresenterClient(w.x, w.y, width, height, w.title)
	if !w.presented || w.needsFullRedraw {
		presenter.PresentFull(w.buffer)
		w.presented = true
		w.needsFullRedraw = false
		return
	}
	presenter.PresentClient(w.buffer)
}

func (w *window) requestClose() bool {
	if w.onCloseIntercept != nil {
		w.onCloseIntercept()
		return true
	}
	w.Close()
	return false
}

func (w *window) handleKey() bool {
	key := kos.ReadKey()
	if key.Empty || key.Hotkey {
		return false
	}
	modifier := currentModifiers()
	localized, ascii, printable, hasPrintable := convertKey(key)
	keyEvent := &fyne.KeyEvent{
		Name: localized,
		Physical: fyne.HardwareKey{
			ScanCode: int(key.ScanCode),
		},
	}

	if focused, ok := w.canvas.Focused().(desktop.Keyable); ok {
		focused.KeyDown(keyEvent)
	}

	if localized == fyne.KeyTab && !w.capturesTab(modifier) {
		return true
	}
	if w.triggersShortcut(localized, ascii, modifier) {
		return true
	}

	if localized != fyne.KeyUnknown {
		if focused := w.canvas.Focused(); focused != nil {
			focused.TypedKey(keyEvent)
		} else if handler := w.canvas.OnTypedKey(); handler != nil {
			handler(keyEvent)
		}
	}

	if hasPrintable && modifier&(fyne.KeyModifierControl|fyne.KeyModifierAlt|fyne.KeyModifierSuper) == 0 {
		if focused := w.canvas.Focused(); focused != nil {
			focused.TypedRune(printable)
		} else if handler := w.canvas.OnTypedRune(); handler != nil {
			handler(printable)
		}
	}
	return true
}

func (w *window) handleMouse() bool {
	pos := kos.MouseWindowPosition()
	modifier := currentModifiers()
	current := fyne.NewPos(float32(pos.X), float32(pos.Y))
	buttons := kos.MouseButtons()
	held := kos.MouseHeldButtons()
	scroll := kos.MouseScrollDelta()
	size := w.canvas.Size()
	inside := pos.X >= 0 && pos.Y >= 0 && float32(pos.X) < size.Width && float32(pos.Y) < size.Height

	needsRender := false
	if inside {
		if w.processMouseMove(current, modifier, held.LeftHeld) {
			needsRender = true
		}
	} else if w.hovered != nil {
		w.hovered.MouseOut()
		w.hovered = nil
		needsRender = true
	}

	leftPressed := buttons.LeftPressed || (held.LeftHeld && !w.prevButtons.LeftHeld)
	leftReleased := buttons.LeftReleased || (!held.LeftHeld && w.prevButtons.LeftHeld)
	rightPressed := buttons.RightPressed || (held.RightHeld && !w.prevButtons.RightHeld)
	rightReleased := buttons.RightReleased || (!held.RightHeld && w.prevButtons.RightHeld)

	if inside && leftPressed {
		if w.processMousePress(desktop.MouseButtonPrimary, modifier, current) {
			needsRender = true
		}
	}
	if inside && rightPressed {
		if w.processMousePress(desktop.MouseButtonSecondary, modifier, current) {
			needsRender = true
		}
	}
	if leftReleased {
		if w.processMouseRelease(desktop.MouseButtonPrimary, modifier, current) {
			needsRender = true
		}
	}
	if rightReleased {
		if w.processMouseRelease(desktop.MouseButtonSecondary, modifier, current) {
			needsRender = true
		}
	}
	if inside && (scroll.X != 0 || scroll.Y != 0) {
		if w.processMouseScroll(current, float32(scroll.X), float32(scroll.Y)) {
			needsRender = true
		}
	}

	w.prevButtons = held
	w.mousePos = current
	return needsRender
}

func (w *window) processMouseMove(pos fyne.Position, modifier fyne.KeyModifier, primaryHeld bool) bool {
	needsRender := false

	if primaryHeld && w.dragCandidate != nil {
		delta := pos.Subtract(w.dragStartPos)
		if w.dragged == nil && (math.Abs(float64(delta.X)) >= dragStartThreshold || math.Abs(float64(delta.Y)) >= dragStartThreshold) {
			w.dragged = w.dragCandidate
		}
		if w.dragged != nil {
			if draggedObj, ok := w.dragged.(fyne.CanvasObject); ok {
				ev := &fyne.DragEvent{
					PointEvent: fyne.PointEvent{
						AbsolutePosition: pos,
						Position: pos.Subtract(w.dragOffset).Add(w.dragStartObj.Subtract(draggedObj.Position())),
					},
					Dragged: fyne.NewDelta(pos.X-w.mousePos.X, pos.Y-w.mousePos.Y),
				}
				w.dragged.Dragged(ev)
				needsRender = true
			}
		}
	}

	var oldHovered, hovered desktop.Hoverable
	oldHovered = w.hovered
	matches := func(object fyne.CanvasObject) bool {
		_, ok := object.(desktop.Hoverable)
		return ok
	}
	if object, rel, _ := w.findObjectAtPositionMatching(pos, matches); object != nil {
		hovered = object.(desktop.Hoverable)
		event := &desktop.MouseEvent{
			PointEvent: fyne.PointEvent{
				AbsolutePosition: pos,
				Position:         rel,
			},
			Modifier: modifier,
		}
		if hovered == oldHovered {
			hovered.MouseMoved(event)
		} else {
			if oldHovered != nil {
				oldHovered.MouseOut()
			}
			hovered.MouseIn(event)
			needsRender = true
		}
	} else if oldHovered != nil {
		oldHovered.MouseOut()
		needsRender = true
	}
	w.hovered = hovered
	return needsRender
}

func (w *window) processMousePress(button desktop.MouseButton, modifier fyne.KeyModifier, pos fyne.Position) bool {
	matches := func(object fyne.CanvasObject) bool {
		switch object.(type) {
		case fyne.Tappable, fyne.SecondaryTappable, fyne.Focusable, desktop.Mouseable, fyne.Draggable:
			return true
		default:
			return false
		}
	}
	object, rel, _ := w.findObjectAtPositionMatching(pos, matches)
	if object == nil {
		w.pressed = nil
		w.dragCandidate = nil
		return false
	}
	mouseEvent := &desktop.MouseEvent{
		PointEvent: fyne.PointEvent{
			AbsolutePosition: pos,
			Position:         rel,
		},
		Button:   button,
		Modifier: modifier,
	}
	if mouseable, ok := object.(desktop.Mouseable); ok {
		mouseable.MouseDown(mouseEvent)
	}
	if focusable, ok := object.(fyne.Focusable); !ok || focusable != w.canvas.Focused() {
		w.canvas.Unfocus()
	}
	w.pressed = object
	w.dragged = nil
	w.dragCandidate = nil
	if button == desktop.MouseButtonPrimary {
		w.dragStartPos = pos
		if draggable, ok := object.(fyne.Draggable); ok {
			w.dragCandidate = draggable
			w.dragOffset = pos.Subtract(rel)
			if draggableObject, ok := draggable.(fyne.CanvasObject); ok {
				w.dragStartObj = draggableObject.Position()
			}
		}
	}
	return true
}

func (w *window) processMouseRelease(button desktop.MouseButton, modifier fyne.KeyModifier, pos fyne.Position) bool {
	matches := func(object fyne.CanvasObject) bool {
		switch object.(type) {
		case fyne.Tappable, fyne.SecondaryTappable, desktop.Mouseable:
			return true
		default:
			return false
		}
	}
	object, rel, _ := w.findObjectAtPositionMatching(pos, matches)
	mouseEvent := &desktop.MouseEvent{
		PointEvent: fyne.PointEvent{
			AbsolutePosition: pos,
			Position:         rel,
		},
		Button:   button,
		Modifier: modifier,
	}
	if mouseable, ok := object.(desktop.Mouseable); ok {
		mouseable.MouseUp(mouseEvent)
	}

	if button == desktop.MouseButtonPrimary && w.dragged != nil {
		w.dragged.DragEnd()
		w.dragCandidate = nil
		w.dragged = nil
		w.pressed = nil
		return true
	}

	pointEvent := &fyne.PointEvent{
		AbsolutePosition: pos,
		Position:         rel,
	}
	needsRender := false
	if object != nil && object == w.pressed {
		if button == desktop.MouseButtonPrimary {
			handleFocusOnTap(w.canvas, object)
			if tappable, ok := object.(fyne.Tappable); ok {
				tappable.Tapped(pointEvent)
				needsRender = true
			}
		}
		if button == desktop.MouseButtonSecondary {
			handleFocusOnTap(w.canvas, object)
			if tappable, ok := object.(fyne.SecondaryTappable); ok {
				tappable.TappedSecondary(pointEvent)
				needsRender = true
			}
		}
	}
	w.dragCandidate = nil
	w.dragged = nil
	w.pressed = nil
	return needsRender
}

func (w *window) processMouseScroll(pos fyne.Position, dx float32, dy float32) bool {
	matches := func(object fyne.CanvasObject) bool {
		_, ok := object.(fyne.Scrollable)
		return ok
	}
	object, rel, _ := w.findObjectAtPositionMatching(pos, matches)
	scrollable, ok := object.(fyne.Scrollable)
	if !ok {
		return false
	}
	scrollable.Scrolled(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{
			AbsolutePosition: pos,
			Position:         rel,
		},
		Scrolled: fyne.NewDelta(dx, dy),
	})
	return true
}

func (w *window) findObjectAtPositionMatching(pos fyne.Position, matches func(fyne.CanvasObject) bool) (fyne.CanvasObject, fyne.Position, int) {
	return intdriver.FindObjectAtPositionMatching(pos, matches, w.canvas.Overlays().Top(), w.canvas.Content())
}

func (w *window) capturesTab(modifier fyne.KeyModifier) bool {
	captures := false
	if entry, ok := w.canvas.Focused().(fyne.Tabbable); ok {
		captures = entry.AcceptsTab()
	}
	if captures {
		return true
	}
	switch modifier {
	case 0:
		w.canvas.FocusNext()
		return false
	case fyne.KeyModifierShift:
		w.canvas.FocusPrevious()
		return false
	}
	return true
}

func (w *window) triggersShortcut(localizedKeyName fyne.KeyName, key fyne.KeyName, modifier fyne.KeyModifier) bool {
	var shortcut fyne.Shortcut
	keyName := localizedKeyName
	resemblesShortcut := modifier&(fyne.KeyModifierControl|fyne.KeyModifierSuper) != 0
	if localizedKeyName == fyne.KeyUnknown && resemblesShortcut && key != fyne.KeyUnknown {
		keyName = key
	}

	if modifier == fyne.KeyModifierShortcutDefault {
		switch keyName {
		case fyne.KeyV:
			shortcut = &fyne.ShortcutPaste{Clipboard: w.Clipboard()}
		case fyne.KeyC, fyne.KeyInsert:
			shortcut = &fyne.ShortcutCopy{Clipboard: w.Clipboard()}
		case fyne.KeyX:
			shortcut = &fyne.ShortcutCut{Clipboard: w.Clipboard()}
		case fyne.KeyA:
			shortcut = &fyne.ShortcutSelectAll{}
		}
	}
	if modifier == fyne.KeyModifierShift {
		switch keyName {
		case fyne.KeyInsert:
			shortcut = &fyne.ShortcutPaste{Clipboard: w.Clipboard()}
		case fyne.KeyDelete:
			shortcut = &fyne.ShortcutCut{Clipboard: w.Clipboard()}
		}
	}
	if shortcut == nil && modifier != 0 && !isKeyModifier(keyName) && modifier != fyne.KeyModifierShift {
		shortcut = &desktop.CustomShortcut{
			KeyName:  keyName,
			Modifier: modifier,
		}
	}
	if shortcut == nil {
		return false
	}

	if focused, ok := w.canvas.Focused().(fyne.Shortcutable); ok {
		shouldRun := true
		type selectableText interface {
			fyne.Disableable
			SelectedText() string
		}
		if selectable, ok := focused.(selectableText); ok && selectable.Disabled() {
			shouldRun = shortcut.ShortcutName() == "Copy"
		}
		if shouldRun {
			focused.TypedShortcut(shortcut)
		}
		return shouldRun
	}

	w.canvas.TypedShortcut(shortcut)
	return true
}

func handleFocusOnTap(canvas fyne.Canvas, object interface{}) {
	if canvas == nil {
		return
	}
	unfocus := true
	if focus, ok := object.(fyne.Focusable); ok {
		if disableable, ok := object.(fyne.Disableable); !ok || !disableable.Disabled() {
			unfocus = false
			if focus != canvas.Focused() {
				unfocus = true
			}
		}
	}
	if unfocus {
		canvas.Unfocus()
	}
}
