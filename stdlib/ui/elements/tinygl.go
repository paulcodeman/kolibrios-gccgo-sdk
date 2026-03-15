package elements

import "ui"

// TinyGL creates a tinygl rendering container using CreateElement defaults.
func TinyGL() *ui.Element {
	view := ui.CreateTinyGL()
	return view
}

// TinyGLStyle returns a neutral tinygl style if you want to override defaults.
func TinyGLStyle() ui.Style {
	return ui.DefaultTinyGLStyle()
}
