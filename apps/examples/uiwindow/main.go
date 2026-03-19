package main

import (
	"strconv"

	"kos"
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
	notificationsEnabled := true
	denseMode := false
	themeMode := "ocean"
	progressValue := 35
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
	applyFocus := func(element *ui.Element, update func(*ui.Style)) {
		element.UpdateFocusStyle(update)
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

	controlLabTitle := elements.Label("Form Controls")
	apply(controlLabTitle, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(10, 0, 6, 0)
		style.SetForeground(ui.Navy)
		style.SetFontSize(14)
	})

	controlLabHint := elements.Label("These are now real Element-based controls built on top of the shared spec registry: checkbox, radio, progress and range.")
	apply(controlLabHint, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
		style.SetLineHeight(15)
	})

	controlSummary := elements.Label("")
	apply(controlSummary, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetPadding(6, 8)
		style.SetBackground(ui.White)
		style.SetBorder(1, ui.Silver)
		style.SetBorderRadius(8)
		style.SetFontPath(monoFontPath)
		style.SetFontSize(12)
	})

	eventSummary := elements.Label("event: idle")
	apply(eventSummary, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetPadding(6, 8)
		style.SetBackground(ui.White)
		style.SetBorder(1, ui.Silver)
		style.SetBorderRadius(8)
		style.SetFontPath(monoFontPath)
		style.SetFontSize(12)
	})

	eventCaptureSummary := elements.Label("capture: idle")
	apply(eventCaptureSummary, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetPadding(6, 8)
		style.SetBackground(ui.White)
		style.SetBorder(1, ui.Silver)
		style.SetBorderRadius(8)
		style.SetFontPath(monoFontPath)
		style.SetFontSize(12)
	})

	eventBubbleHost := elements.Box()
	apply(eventBubbleHost, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetPadding(8)
		style.SetBackground(ui.White)
		style.SetBorder(1, ui.Silver)
		style.SetBorderRadius(8)
	})

	eventBubbleHint := elements.Label("Bubble click through parent, inspect capture phase, and pointerenter/pointerdown compatibility events.")
	apply(eventBubbleHint, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
		style.SetLineHeight(15)
	})
	bubbleButton := elements.Button("Bubble click to parent")
	apply(bubbleButton, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayInlineBlock)
		style.SetMargin(0, 8, 0, 0)
	})

	preventCheckbox := elements.Checkbox("Prevent checkbox default toggle", false)
	apply(preventCheckbox, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(6, 0, 0, 0)
		style.SetBorderRadius(6)
	})

	notifyCheckbox := elements.Checkbox("Enable notifications", true)
	apply(notifyCheckbox, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 4, 0)
		style.SetBorderRadius(6)
	})

	denseCheckbox := elements.Checkbox("Dense mode", false)
	apply(denseCheckbox, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetBorderRadius(6)
	})

	themeLabel := elements.Label("Theme")
	apply(themeLabel, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 4, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
	})

	themeOcean := elements.Radio("Ocean", "theme", true)
	apply(themeOcean, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 2, 0)
		style.SetBorderRadius(6)
	})

	themeSunset := elements.Radio("Sunset", "theme", false)
	apply(themeSunset, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetBorderRadius(6)
	})

	progressLabel := elements.Label("Progress")
	apply(progressLabel, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 4, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
	})

	progress := elements.Progress(0, 100, progressValue)
	apply(progress, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 6, 0)
		style.SetWidth(210)
	})

	rangeLabel := elements.Label("Adjust progress")
	apply(rangeLabel, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 4, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(12)
	})

	rangeInput := elements.Range(0, 100, progressValue)
	apply(rangeInput, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetWidth(210)
	})

	styleTitle := elements.Label("Style Lab")
	apply(styleTitle, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(10, 0, 6, 0)
		style.SetForeground(ui.Navy)
		style.SetFontSize(14)
	})

	styleHint := elements.Label("Compare border-box vs content-box, check underline + line-height, Tab to the outline button, note that the hidden chip still keeps layout space, and the static hint box below now opts into contain/will-change layer hints.")
	apply(styleHint, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
		style.SetLineHeight(15)
	})

	borderDemo := elements.Label("Per-side border")
	apply(borderDemo, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetWidth(180)
		style.SetMargin(0, 0, 8, 0)
		style.SetPadding(8, 10)
		style.SetBackground(ui.White)
		style.SetBorderTop(4, ui.Blue)
		style.SetBorderRight(3, ui.Teal)
		style.SetBorderBottom(5, ui.Maroon)
		style.SetBorderLeft(7, ui.Navy)
		style.SetBorderRadius(8)
	})

	boxRow := ui.CreateBox()
	apply(boxRow, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
	})

	boxSizingCard := func(title string, mode ui.BoxSizing, fill kos.Color) *ui.Element {
		card := elements.Label(title + "\nwidth: 132")
		apply(card, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayInlineBlock)
			style.SetWidth(132)
			style.SetMargin(0, 8, 0, 0)
			style.SetPadding(8)
			style.SetBorder(6, ui.Navy)
			style.SetBackground(fill)
			style.SetBoxSizing(mode)
			style.SetLineHeight(16)
		})
		return card
	}
	boxRow.Append(boxSizingCard("border-box", ui.BoxSizingBorderBox, ui.White))
	boxRow.Append(boxSizingCard("content-box", ui.BoxSizingContentBox, ui.Aqua))

	textDemo := elements.Label("Underline + line-height sample\nSecond line should sit lower, like CSS line-height.")
	apply(textDemo, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetPadding(6, 8)
		style.SetBackground(ui.White)
		style.SetBorder(1, ui.Silver)
		style.SetBorderRadius(8)
		style.SetTextDecoration(ui.TextDecorationUnderline)
		style.SetLineHeight(20)
	})

	textFlowHint := elements.Label("Text-flow modes remain in uidocument for now; this native demo keeps startup on the simpler path while the new whitespace modes are stabilized.")
	apply(textFlowHint, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
		style.SetPadding(6, 8)
		style.SetBackground(ui.White)
		style.SetBorder(1, ui.Silver)
		style.SetBorderRadius(8)
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
		style.SetLineHeight(15)
	})

	hintBox := ui.CreateBox()
	apply(hintBox, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetWidth(220)
		style.SetMargin(0, 0, 8, 0)
		style.SetPadding(8, 10)
		style.SetBorder(1, ui.Silver)
		style.SetBorderRadius(10)
		style.SetBackground(ui.White)
		style.SetContainString("content")
		style.SetWillChangeString("opacity, transform")
	})
	hintTitle := elements.Label("Contain + will-change")
	apply(hintTitle, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(ui.Navy)
		style.SetMargin(0, 0, 4, 0)
		style.SetFontSize(12)
	})
	hintText := elements.Label("This static box uses CSS-like hints to bias retained-layer promotion without changing its visual behavior.")
	apply(hintText, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
		style.SetLineHeight(15)
	})
	hintSub := elements.Label("Current runtime only uses these hints conservatively for safe box-layer caching.")
	apply(hintSub, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetForeground(ui.Gray)
		style.SetFontSize(11)
		style.SetLineHeight(15)
	})
	hintBox.Append(hintTitle)
	hintBox.Append(hintText)
	hintBox.Append(hintSub)

	visibilityRow := ui.CreateBox()
	apply(visibilityRow, func(style *ui.Style) {
		style.SetDisplay(ui.DisplayBlock)
		style.SetMargin(0, 0, 8, 0)
	})

	chip := func(text string, fill kos.Color) *ui.Element {
		label := elements.Label(text)
		apply(label, func(style *ui.Style) {
			style.SetDisplay(ui.DisplayInlineBlock)
			style.SetPadding(3, 8)
			style.SetMargin(0, 6, 0, 0)
			style.SetBorderRadius(999)
			style.SetBackground(fill)
			style.SetForeground(ui.White)
		})
		return label
	}
	visibilityBefore := chip("Before", ui.Navy)
	visibilityHidden := chip("Hidden gap", ui.Teal)
	visibilityHidden.UpdateStyle(func(style *ui.Style) {
		style.SetVisibility(ui.VisibilityHidden)
	})
	visibilityAfter := chip("After", ui.Maroon)
	visibilityRow.Append(visibilityBefore)
	visibilityRow.Append(visibilityHidden)
	visibilityRow.Append(visibilityAfter)

	outlineDemo := elements.Button("Outline Focus")
	styleButton(outlineDemo)
	apply(outlineDemo, func(style *ui.Style) {
		style.SetBackground(ui.White)
		style.SetForeground(ui.Navy)
		style.SetBorder(1, ui.Silver)
		style.SetPadding(3, 12)
	})
	applyFocus(outlineDemo, func(style *ui.Style) {
		style.SetOutline(2, ui.Blue)
		style.SetOutlineOffset(1)
		style.SetBorderColor(ui.Blue)
	})

	updateControlLab := func() {
		progress.SetValue(progressValue)
		rangeInput.SetValue(progressValue)
		if themeMode == "ocean" {
			progress.UpdateStyle(func(style *ui.Style) {
				style.SetForeground(ui.Blue)
				style.SetBorderColor(ui.Navy)
			})
			rangeInput.UpdateStyle(func(style *ui.Style) {
				style.SetForeground(ui.Blue)
			})
		} else {
			progress.UpdateStyle(func(style *ui.Style) {
				style.SetForeground(ui.Maroon)
				style.SetBorderColor(ui.Maroon)
			})
			rangeInput.UpdateStyle(func(style *ui.Style) {
				style.SetForeground(ui.Maroon)
			})
		}
		if denseMode {
			controlSummary.UpdateStyle(func(style *ui.Style) {
				style.SetBackground(ui.Silver)
			})
		} else {
			controlSummary.UpdateStyle(func(style *ui.Style) {
				style.SetBackground(ui.White)
			})
		}
		controlSummary.SetText(window, "notify="+strconv.FormatBool(notificationsEnabled)+" | dense="+strconv.FormatBool(denseMode)+" | theme="+themeMode+" | progress="+strconv.Itoa(progressValue))
	}

	setEvent := func(text string) {
		eventSummary.SetText(window, "event: "+text)
	}
	setCapture := func(text string) {
		eventCaptureSummary.SetText(window, "capture: "+text)
	}
	phaseName := func(phase ui.EventPhase) string {
		switch phase {
		case ui.EventPhaseTarget:
			return "target"
		case ui.EventPhaseBubble:
			return "bubble"
		default:
			return "none"
		}
	}
	nodeName := func(node ui.Node) string {
		if element, ok := node.(*ui.Element); ok && element != nil {
			if spec := element.Spec(); spec != nil && spec.Name != "" {
				return spec.Name
			}
			return element.Kind().String()
		}
		return "node"
	}

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
	denseCheckbox.OnChange = func(checked bool) {
		denseMode = checked
		updateControlLab()
	}
	themeOcean.OnChange = func(checked bool) {
		if checked {
			themeMode = "ocean"
			updateControlLab()
		}
	}
	themeSunset.OnChange = func(checked bool) {
		if checked {
			themeMode = "sunset"
			updateControlLab()
		}
	}
	rangeInput.OnChange = func(value int) {
		progressValue = value
		updateControlLab()
	}
	input.OnFocus = func() {
		setEvent("input focus")
	}
	input.OnBlur = func() {
		setEvent("input blur")
	}
	input.OnInput = func(value string) {
		setEvent("input value=" + value)
	}
	textarea.OnInput = func(value string) {
		setEvent("textarea len=" + strconv.Itoa(len(value)))
	}
	notifyCheckbox.OnChange = func(checked bool) {
		notificationsEnabled = checked
		updateControlLab()
		setEvent("checkbox change=" + strconv.FormatBool(checked))
	}
	rangeInput.OnInput = func(value int) {
		setEvent("range input=" + strconv.Itoa(value))
	}
	styled.OnMouseEnter = func() {
		setEvent("rounded enter")
	}
	styled.OnMouseLeave = func() {
		setEvent("rounded leave")
	}
	eventBubbleHost.OnClick = func(_ *ui.Element, event *ui.Event) {
		setEvent("bubble target=" + nodeName(event.Target) + " current=" + nodeName(event.CurrentTarget) + " phase=" + phaseName(event.Phase))
	}
	eventBubbleHost.OnEventCapture = func(_ *ui.Element, event *ui.Event) {
		if event.Type == ui.EventClick || event.Type == ui.EventPointerDown {
			setCapture("target=" + nodeName(event.Target) + " current=" + nodeName(event.CurrentTarget) + " phase=" + phaseName(event.Phase))
		}
	}
	bubbleButton.OnPointerEnter = func(_ *ui.Element, event *ui.Event) {
		setEvent("pointerenter current=" + nodeName(event.CurrentTarget))
	}
	bubbleButton.OnPointerDown = func(_ *ui.Element, event *ui.Event) {
		setEvent("pointerdown primary=" + strconv.FormatBool(event.IsPrimary) + " buttons=" + strconv.Itoa(int(event.Buttons)))
	}
	bubbleButton.OnPointerCancel = func() {
		setEvent("pointercancel")
	}
	preventCheckbox.OnClick = func(_ *ui.Element, event *ui.Event) {
		event.PreventDefault()
		setEvent("preventDefault on checkbox click")
	}
	preventCheckbox.OnChange = func(checked bool) {
		setEvent("checkbox changed=" + strconv.FormatBool(checked))
	}
	updateControlLab()
	eventBubbleHost.Append(eventBubbleHint)
	eventBubbleHost.Append(bubbleButton)
	eventBubbleHost.Append(preventCheckbox)

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
	card.Append(controlLabTitle)
	card.Append(controlLabHint)
	card.Append(controlSummary)
	card.Append(eventSummary)
	card.Append(eventCaptureSummary)
	card.Append(eventBubbleHost)
	card.Append(notifyCheckbox)
	card.Append(denseCheckbox)
	card.Append(themeLabel)
	card.Append(themeOcean)
	card.Append(themeSunset)
	card.Append(progressLabel)
	card.Append(progress)
	card.Append(rangeLabel)
	card.Append(rangeInput)
	card.Append(styleTitle)
	card.Append(styleHint)
	card.Append(borderDemo)
	card.Append(boxRow)
	card.Append(textDemo)
	card.Append(textFlowHint)
	card.Append(hintBox)
	card.Append(visibilityRow)
	card.Append(outlineDemo)

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
