package elements

import (
	"kos"
	"ui"
)

// Button creates a neutral button that uses CreateElement defaults
// and adds hover/active visuals similar to standard UI buttons.
func Button(text string) *ui.Element {
	button := ui.CreateButton()
	button.Label = text
	button.Text = ""
	button.StyleHover = ButtonHoverStyle()
	button.StyleActive = ButtonActiveStyle()
	return button
}

// ButtonAt creates a button at a fixed position with the given Kolibri button ID.
func ButtonAt(id kos.ButtonID, label string, x int, y int) *ui.Element {
	button := ui.CreateButton()
	button.ID = id
	button.Label = label
	button.Text = ""
	button.SetLeft(x)
	button.SetTop(y)
	button.StyleHover = ButtonHoverStyle()
	button.StyleActive = ButtonActiveStyle()
	return button
}

// ButtonStyle returns a neutral base style if you want to override defaults.
func ButtonStyle() ui.Style {
	style := ui.Style{}
	style.SetBackground(ui.Silver)
	style.SetForeground(ui.Black)
	style.SetBorderColor(ui.Gray)
	style.SetBorderWidth(1)
	style.SetGradient(ui.Gradient{
		From:      ui.White,
		To:        ui.Silver,
		Direction: ui.GradientVertical,
	})
	style.SetShadow(ui.Shadow{
		OffsetX: 1,
		OffsetY: 1,
		Blur:    2,
		Color:   ui.Black,
		Alpha:   60,
	})
	style.SetTextAlign(ui.TextAlignCenter)
	style.SetPadding(4, 6, 4, 6)
	return style
}

func ButtonHoverStyle() ui.Style {
	style := ui.Style{}
	style.SetBackground(ui.White)
	style.SetBorderColor(ui.Gray)
	style.SetBorderWidth(1)
	style.SetGradient(ui.Gradient{
		From:      ui.White,
		To:        ui.Silver,
		Direction: ui.GradientVertical,
	})
	style.SetShadow(ui.Shadow{
		OffsetX: 1,
		OffsetY: 1,
		Blur:    2,
		Color:   ui.Black,
		Alpha:   70,
	})
	return style
}

func ButtonActiveStyle() ui.Style {
	style := ui.Style{}
	style.SetBackground(ui.Gray)
	style.SetForeground(ui.Black)
	style.SetBorderWidth(1)
	style.SetBorderColor(ui.Gray)
	style.SetGradient(ui.Gradient{
		From:      ui.Gray,
		To:        ui.Silver,
		Direction: ui.GradientVertical,
	})
	style.SetShadow(ui.Shadow{
		OffsetX: 0,
		OffsetY: 0,
		Blur:    1,
		Color:   ui.Black,
		Alpha:   40,
	})
	return style
}
