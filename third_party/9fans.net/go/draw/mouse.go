package draw

type Mouse struct {
	Point
	Buttons int
	Msec    uint32
}

type Mousectl struct {
	Mouse
	C       <-chan Mouse
	Resize  <-chan bool
	Display *Display
}

func (mc *Mousectl) Read() Mouse {
	if mc == nil || mc.C == nil {
		return Mouse{}
	}
	m := <-mc.C
	mc.Mouse = m
	return m
}

type Keyboardctl struct {
	C <-chan rune
}

type Cursor struct {
	Offset Point
	Black  [32]uint8
	White  [32]uint8
}
