package main

import (
	"strconv"

	"ui"
	"ui/elements"
)

const (
	windowWidth  = 360
	windowHeight = 340
	// Extra window thread stack for gccgo (default is 0x10000).
	extraWindowStackSize = 0x20000
)

func Run() {
	window := ui.NewWindowDefault()
	window.UpdateStyle(func(style *ui.Style) {
		style.SetWidth(windowWidth)
		style.SetHeight(windowHeight)
		style.SetOverflow(ui.OverflowAuto)
		style.SetGradient(ui.Gradient{
			From:      ui.White,
			To:        ui.Silver,
			Direction: ui.GradientVertical,
		})
	})
	window.SetTitle("UI Window Demo")
	window.CenterOnScreen()

	count := 0
	fontPath := "assets/OpenSans-Regular.ttf"
	monoFontPath := "assets/RobotoMono-Regular.ttf"

	apply := func(element *ui.Element, update func(*ui.Style)) {
		element.UpdateStyle(func(style *ui.Style) {
			style.SetFontPath(fontPath)
			if update != nil {
				update(style)
			}
		})
	}
	applyHover := func(element *ui.Element, update func(*ui.Style)) {
		element.UpdateHoverStyle(update)
	}
	applyActive := func(element *ui.Element, update func(*ui.Style)) {
		element.UpdateActiveStyle(update)
	}

	root := ui.CreateBox()
	apply(root, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(12)
	})

	header := ui.CreateBox()
	apply(header, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(10, 12)
		style.SetMargin(0, 0, 10, 0)
		style.SetBorderRadius(10)
		style.SetGradient(ui.Gradient{
			From:      ui.Navy,
			To:        ui.Blue,
			Direction: ui.GradientHorizontal,
		})
		style.SetShadow(ui.Shadow{OffsetX: 0, OffsetY: 2, Blur: 4, Color: ui.Black, Alpha: 70})
	})

	title := elements.Label("UI Window Demo")
	apply(title, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(ui.White)
		style.SetMargin(0, 0, 4, 0)
		style.SetFontSize(18)
	})

	subtitle := elements.Label("Inline layout, display modes, nested elements")
	apply(subtitle, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(ui.Silver)
		style.SetMargin(0, 0, 6, 0)
		style.SetFontSize(12)
	})

	badgeRow := ui.CreateBox()
	apply(badgeRow, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
	})

	badge := func(text string) *ui.Element {
		b := elements.Label(text)
		apply(b, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayInlineBlock)
			style.SetPadding(2, 6)
			style.SetMargin(0, 6, 0, 0)
			style.SetBorderRadius(6)
			style.SetBackground(ui.White)
			style.SetForeground(ui.Navy)
			style.SetFontSize(10)
		})
		return b
	}

	header.Append(title)
	header.Append(subtitle)
	badgeRow.Append(badge("INLINE"))
	badgeRow.Append(badge("BLOCK"))
	badgeRow.Append(badge("NESTED"))
	header.Append(badgeRow)

	card := ui.CreateBox()
	apply(card, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetPadding(10, 12)
		style.SetMargin(0, 0, 10, 0)
		style.SetBorderRadius(10)
		style.SetBackground(ui.White)
		style.SetBorderColor(ui.Silver)
		style.SetBorderWidth(1)
		style.SetShadow(ui.Shadow{OffsetX: 0, OffsetY: 1, Blur: 3, Color: ui.Black, Alpha: 40})
	})

	flowGap := []int{0, 8, 8, 0}

	counterRow := ui.CreateBox()
	apply(counterRow, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
	})

	counterLabel := elements.Label("Count")
	apply(counterLabel, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(flowGap...)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
	})

	counterValue := elements.Label("0")
	apply(counterValue, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetPadding(2, 8)
		style.SetMargin(0, 0, 8, 0)
		style.SetBorderRadius(6)
		style.SetBackground(ui.Aqua)
		style.SetForeground(ui.Navy)
		style.SetFontPath(monoFontPath)
		style.SetFontSize(18)
	})

	counterRow.Append(counterLabel)
	counterRow.Append(counterValue)

	updateLabel := func() {
		counterValue.SetText(window, strconv.Itoa(count))
	}

	spawned := 0
	spawnExtraWindow := func() {
		spawned++
		index := spawned
		extra := ui.NewWindowDefault()
		extra.UpdateStyle(func(style *ui.Style) {
			style.SetWidth(windowWidth)
			style.SetHeight(windowHeight)
			style.SetOverflow(ui.OverflowAuto)
			style.SetGradient(ui.Gradient{
				From:      ui.White,
				To:        ui.Silver,
				Direction: ui.GradientVertical,
			})
		})
		extra.SetTitle("UI Extra Window")

		extraLabel := elements.Label("Extra window #" + strconv.Itoa(index))
		apply(extraLabel, func(style *ui.Style) {
			style.SetLeft(20)
			style.SetTop(20)
		})

		extraClose := elements.Button("Close")
		apply(extraClose, func(style *ui.Style) {
			style.SetLeft(20)
			style.SetTop(70)
			style.SetWidth(70)
		})
		extraClose.OnClick = func() {
			extra.Close()
		}

		extra.Append(extraLabel)
		extra.Append(extraClose)

		extra.StartThreadedWithStack(extraWindowStackSize)
	}

	controlsTitle := elements.Label("Controls")
	apply(controlsTitle, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetForeground(ui.Navy)
		style.SetFontSize(14)
	})

	styleButton := func(btn *ui.Element) {
		apply(btn, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayInlineBlock)
			style.SetMargin(flowGap...)
			style.SetBorderRadius(8)
			style.SetPadding(3, 10)
		})
	}

	inc := elements.Button("+")
	styleButton(inc)

	dec := elements.Button("-")
	styleButton(dec)

	reset := elements.Button("Reset")
	styleButton(reset)

	spawn := elements.Button("New")
	styleButton(spawn)
	apply(spawn, func(style *ui.Style) {
		style.SetBackground(ui.Blue)
		style.SetForeground(ui.White)
		style.SetBorderColor(ui.Navy)
		style.SetBorderWidth(1)
	})

	exit := elements.Button("Exit")
	styleButton(exit)
	apply(exit, func(style *ui.Style) {
		style.SetBackground(ui.Maroon)
		style.SetForeground(ui.White)
		style.SetBorderColor(ui.Maroon)
		style.SetBorderWidth(1)
	})

	divider := ui.CreateBox()
	apply(divider, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetHeight(1)
		style.SetBackground(ui.Silver)
		style.SetMargin(6, 0)
	})
	dividerAfterForm := ui.CreateBox()
	apply(dividerAfterForm, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetHeight(1)
		style.SetBackground(ui.Silver)
		style.SetMargin(6, 0)
	})

	formTitle := elements.Label("Inputs")
	apply(formTitle, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(4, 0, 4, 0)
		style.SetForeground(ui.Navy)
		style.SetFontSize(14)
	})

	inputLabel := elements.Label("Input")
	apply(inputLabel, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(2, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
	})

	input := elements.Input("Type here...")
	apply(input, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetBorderRadius(6)
		style.SetFontSize(13)
	})

	textareaLabel := elements.Label("Textarea")
	apply(textareaLabel, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(2, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
	})

	textarea := elements.Textarea("Multiline text area\nLine 2\nLine 3")
	apply(textarea, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetHeight(60)
		style.SetBorderRadius(6)
		style.SetPadding(6)
		style.SetFontSize(13)
	})

	styled := elements.Button("Rounded")
	styleButton(styled)
	apply(styled, func(style *ui.Style) {
		style.SetPadding(3, 12)
		style.SetBorderWidth(2)
		style.SetBorderColor(ui.Navy)
		style.SetBorderRadius(10)
		style.SetBackground(ui.Aqua)
		style.SetForeground(ui.Navy)
		style.SetGradientPtr(nil)
		style.SetShadowPtr(nil)
	})
	applyHover(styled, func(style *ui.Style) {
		style.SetGradientPtr(nil)
		style.SetShadowPtr(nil)
	})
	applyActive(styled, func(style *ui.Style) {
		style.SetGradientPtr(nil)
		style.SetShadowPtr(nil)
	})

	pill := elements.Button("Pill")
	styleButton(pill)
	apply(pill, func(style *ui.Style) {
		style.SetPadding(3, 16)
		style.SetBorderWidth(1)
		style.SetBorderColor(ui.Teal)
		style.SetBorderRadius(999)
		style.SetBackground(ui.White)
		style.SetForeground(ui.Teal)
		style.SetGradientPtr(nil)
		style.SetShadowPtr(nil)
	})
	applyHover(pill, func(style *ui.Style) {
		style.SetGradientPtr(nil)
		style.SetShadowPtr(nil)
	})
	applyActive(pill, func(style *ui.Style) {
		style.SetGradientPtr(nil)
		style.SetShadowPtr(nil)
	})

	inc.OnClick = func() {
		count++
		updateLabel()
	}
	dec.OnClick = func() {
		count--
		updateLabel()
	}
	reset.OnClick = func() {
		count = 0
		updateLabel()
	}
	spawn.OnClick = func() {
		spawnExtraWindow()
	}
	exit.OnClick = func() {
		window.Close()
	}
	styled.OnClick = func() {
		count += 10
		updateLabel()
	}
	pill.OnClick = func() {
		count = 0
		updateLabel()
	}

	card.Append(counterRow)
	card.Append(controlsTitle)
	card.Append(inc)
	card.Append(dec)
	card.Append(reset)
	card.Append(spawn)
	card.Append(exit)
	card.Append(divider)
	card.Append(formTitle)
	card.Append(inputLabel)
	card.Append(input)
	card.Append(textareaLabel)
	card.Append(textarea)
	card.Append(dividerAfterForm)
	card.Append(styled)
	card.Append(pill)

	footer := elements.Label("Tip: resize the window to see inline flow wrap. Click inputs to type.")
	apply(footer, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
	})

	root.Append(header)
	root.Append(card)
	root.Append(footer)

	window.Append(root)

	window.Start()
}

func main() {
	Run()
}
