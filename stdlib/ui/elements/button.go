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
	return ui.Style{
		Background:  ui.ColorPtr(ui.Silver),
		Foreground:  ui.ColorPtr(ui.Black),
		BorderColor: ui.ColorPtr(ui.Gray),
		BorderWidth: ui.IntPtr(1),
		Gradient: &ui.Gradient{
			From:      ui.White,
			To:        ui.Silver,
			Direction: ui.GradientVertical,
		},
		Shadow: &ui.Shadow{
			OffsetX: 1,
			OffsetY: 1,
			Blur:    2,
			Color:   ui.Black,
			Alpha:   60,
		},
		TextAlign: ui.AlignPtr(ui.TextAlignCenter),
		Padding: &ui.Spacing{
			Left:   6,
			Top:    4,
			Right:  6,
			Bottom: 4,
		},
	}
}

func ButtonHoverStyle() ui.Style {
	return ui.Style{
		Background:  ui.ColorPtr(ui.White),
		BorderColor: ui.ColorPtr(ui.Gray),
		BorderWidth: ui.IntPtr(1),
		Gradient: &ui.Gradient{
			From:      ui.White,
			To:        ui.Silver,
			Direction: ui.GradientVertical,
		},
		Shadow: &ui.Shadow{
			OffsetX: 1,
			OffsetY: 1,
			Blur:    2,
			Color:   ui.Black,
			Alpha:   70,
		},
	}
}

func ButtonActiveStyle() ui.Style {
	return ui.Style{
		Background:  ui.ColorPtr(ui.Gray),
		Foreground:  ui.ColorPtr(ui.Black),
		BorderWidth: ui.IntPtr(0),
		Gradient: &ui.Gradient{
			From:      ui.Gray,
			To:        ui.Silver,
			Direction: ui.GradientVertical,
		},
		Shadow: &ui.Shadow{
			OffsetX: 0,
			OffsetY: 0,
			Blur:    1,
			Color:   ui.Black,
			Alpha:   40,
		},
	}
}
