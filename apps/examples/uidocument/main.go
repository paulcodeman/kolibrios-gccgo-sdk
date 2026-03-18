package main

import (
	"strconv"

	"kos"
	"ui"
	"ui/elements"
)

const (
	documentWindowWidth  = 440
	documentWindowHeight = 460
)

type demoState struct {
	count   int
	accent  bool
	details bool
}

func style(update func(*ui.Style)) ui.Style {
	value := ui.Style{}
	if update != nil {
		update(&value)
	}
	return value
}

func attachDocumentClick(node *ui.DocumentNode, handler func()) {
	if node == nil || handler == nil {
		return
	}
	node.OnClick = handler
	for _, child := range node.Children {
		attachDocumentClick(child, handler)
	}
}

func documentAction(title string, subtitle string, fill kos.Color, handler func()) *ui.DocumentNode {
	titleNode := ui.NewDocumentText(title, style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(14)
		value.SetMargin(0, 0, 2, 0)
	}))
	subtitleNode := ui.NewDocumentText(subtitle, style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
	}))
	card := ui.NewDocumentElement("action", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 6, 0)
		value.SetPadding(8, 10)
		value.SetBorderRadius(10)
		value.SetBorder(1, ui.Silver)
		value.SetBackground(fill)
	}), titleNode, subtitleNode)
	card.Focusable = true
	card.StyleHover = style(func(value *ui.Style) {
		value.SetBorderColor(ui.Teal)
		value.SetBackground(ui.White)
	})
	card.StyleActive = style(func(value *ui.Style) {
		value.SetBorderColor(ui.Navy)
		value.SetBackground(ui.Silver)
	})
	card.StyleFocus = style(func(value *ui.Style) {
		value.SetBorderColor(ui.Blue)
		value.SetBorderWidth(2)
	})
	attachDocumentClick(card, handler)
	return card
}

func Run() {
	window := ui.NewWindowDefault()
	window.UpdateStyle(func(value *ui.Style) {
		value.SetWidth(documentWindowWidth)
		value.SetHeight(documentWindowHeight)
		value.SetOverflow(ui.OverflowAuto)
		value.SetGradient(ui.Gradient{
			From:      ui.White,
			To:        ui.Silver,
			Direction: ui.GradientVertical,
		})
	})
	window.SetTitle("UI Document Demo")
	window.CenterOnScreen()

	apply := func(element *ui.Element, update func(*ui.Style)) {
		element.UpdateStyle(update)
	}
	styleButton := func(button *ui.Element) {
		apply(button, func(value *ui.Style) {
			value.SetDisplay(ui.DisplayInlineBlock)
			value.SetMargin(0, 8, 8, 0)
			value.SetBorderRadius(8)
			value.SetPadding(4, 10)
		})
	}

	root := ui.CreateBox()
	apply(root, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetPadding(12)
	})

	header := ui.CreateBox()
	apply(header, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 10, 0)
		value.SetPadding(10, 12)
		value.SetBorderRadius(12)
		value.SetGradient(ui.Gradient{
			From:      ui.Navy,
			To:        ui.Blue,
			Direction: ui.GradientHorizontal,
		})
		value.SetShadow(ui.Shadow{OffsetX: 0, OffsetY: 2, Blur: 4, Color: ui.Black, Alpha: 70})
	})

	title := elements.Label("DocumentView Host Demo")
	apply(title, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetForeground(ui.White)
		value.SetMargin(0, 0, 4, 0)
		value.SetFontSize(18)
	})

	subtitle := elements.Label("Native code mutates DocumentNode directly; browser HTML/CSS can later feed the same renderer.")
	apply(subtitle, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetForeground(ui.Silver)
		value.SetFontSize(11)
	})

	header.Append(title)
	header.Append(subtitle)

	panel := ui.CreateBox()
	apply(panel, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 10, 0)
		value.SetPadding(10)
		value.SetBorderRadius(12)
		value.SetBackground(ui.White)
		value.SetBorder(1, ui.Silver)
		value.SetShadow(ui.Shadow{OffsetX: 0, OffsetY: 1, Blur: 3, Color: ui.Black, Alpha: 40})
	})

	panelTitle := elements.Label("Embedded DocumentView")
	apply(panelTitle, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetForeground(ui.Navy)
		value.SetMargin(0, 0, 8, 0)
		value.SetFontSize(14)
	})

	state := demoState{
		count:   2,
		accent:  false,
		details: true,
	}

	countValue := ui.NewDocumentText("", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(22)
		value.SetMargin(0, 0, 8, 0)
	}))
	accentValue := ui.NewDocumentText("", style(func(value *ui.Style) {
		value.SetFontSize(12)
		value.SetMargin(0, 0, 4, 0)
	}))
	detailsBody := ui.NewDocumentText("", style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
	}))

	hero := ui.NewDocumentElement("hero", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 8, 0)
		value.SetPadding(10, 12)
		value.SetBorderRadius(10)
		value.SetGradient(ui.Gradient{
			From:      ui.Aqua,
			To:        ui.White,
			Direction: ui.GradientHorizontal,
		})
	}), ui.NewDocumentText("Shared render pipeline", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(16)
		value.SetMargin(0, 0, 4, 0)
	})), ui.NewDocumentText("Document nodes below are rendered inside a normal Box and share state with native buttons.", style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
	})))

	statusCard := ui.NewDocumentElement("status", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 8, 0)
		value.SetPadding(8, 10)
		value.SetBorderRadius(10)
		value.SetBorder(1, ui.Silver)
		value.SetBackground(ui.White)
	}), ui.NewDocumentText("Count", style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
		value.SetMargin(0, 0, 2, 0)
	})), countValue, ui.NewDocumentText("Accent state", style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
		value.SetMargin(0, 0, 2, 0)
	})), accentValue)

	detailsCard := ui.NewDocumentElement("details", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(8, 0, 0, 0)
		value.SetPadding(8, 10)
		value.SetBorderRadius(10)
		value.SetBackground(ui.Silver)
	}), ui.NewDocumentText("Details", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(12)
		value.SetMargin(0, 0, 3, 0)
	})), detailsBody)

	var updateState func()

	incrementCard := documentAction("+1 inside document", "Mutates the same state as native buttons.", ui.White, func() {
		state.count++
		updateState()
	})
	resetCard := documentAction("Reset from document", "Sets the shared counter back to zero.", ui.White, func() {
		state.count = 0
		updateState()
	})
	accentCard := documentAction("Toggle accent", "Flips colors on both the document and native summary.", ui.Aqua, func() {
		state.accent = !state.accent
		updateState()
	})
	detailsToggleCard := documentAction("Toggle details", "Collapses or expands a document-only details block.", ui.Silver, func() {
		state.details = !state.details
		updateState()
	})
	outlineCard := documentAction("Outline focus card", "Tab to this card; focus should show a custom outline without changing layout.", ui.White, func() {})
	outlineCard.StyleFocus = style(func(value *ui.Style) {
		value.SetOutline(2, ui.Blue)
		value.SetOutlineOffset(1)
		value.SetBorderColor(ui.Blue)
	})
	styleLabTitle := ui.NewDocumentText("Style coverage inside DocumentView", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(13)
		value.SetMargin(8, 0, 6, 0)
	}))
	borderLab := ui.NewDocumentElement("border-lab", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetWidth(190)
		value.SetMargin(0, 0, 8, 0)
		value.SetPadding(8, 10)
		value.SetBackground(ui.White)
		value.SetBorderTop(4, ui.Blue)
		value.SetBorderRight(3, ui.Teal)
		value.SetBorderBottom(5, ui.Maroon)
		value.SetBorderLeft(7, ui.Navy)
		value.SetBorderRadius(8)
	}), ui.NewDocumentText("Per-side border", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(12)
	})))
	boxBorderBox := ui.NewDocumentElement("box-border", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetWidth(160)
		value.SetMargin(0, 0, 6, 0)
		value.SetPadding(8)
		value.SetBorder(6, ui.Navy)
		value.SetBackground(ui.White)
		value.SetBoxSizing(ui.BoxSizingBorderBox)
	}), ui.NewDocumentText("border-box width:160", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(12)
	})))
	boxContentBox := ui.NewDocumentElement("box-content", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetWidth(160)
		value.SetMargin(0, 0, 8, 0)
		value.SetPadding(8)
		value.SetBorder(6, ui.Navy)
		value.SetBackground(ui.Aqua)
		value.SetBoxSizing(ui.BoxSizingContentBox)
	}), ui.NewDocumentText("content-box width:160", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(12)
	})))
	textLab := ui.NewDocumentText("Underline + line-height sample\nSecond line should sit lower, like CSS line-height.", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 8, 0)
		value.SetPadding(6, 8)
		value.SetBackground(ui.White)
		value.SetBorder(1, ui.Silver)
		value.SetBorderRadius(8)
		value.SetTextDecoration(ui.TextDecorationUnderline)
		value.SetLineHeight(20)
	}))
	textNormal := ui.NewDocumentElement("text-normal", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetWidth(150)
		value.SetMargin(0, 0, 6, 0)
		value.SetPadding(6, 8)
		value.SetBackground(ui.White)
		value.SetBorder(1, ui.Silver)
		value.SetBorderRadius(8)
	}), ui.NewDocumentText("normal: words wrap by spaces inside a narrow card.", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(11)
		value.SetLineHeight(15)
	})))
	textNoWrap := ui.NewDocumentElement("text-nowrap", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetWidth(150)
		value.SetMargin(0, 0, 6, 0)
		value.SetPadding(6, 8)
		value.SetBackground(ui.White)
		value.SetBorder(1, ui.Silver)
		value.SetBorderRadius(8)
	}), ui.NewDocumentText("nowrap: this text should stay on one line even in a narrow card.", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(11)
		value.SetLineHeight(15)
		value.SetWhiteSpace(ui.WhiteSpaceNoWrap)
	})))
	textBreakWord := ui.NewDocumentElement("text-break", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetWidth(150)
		value.SetMargin(0, 0, 8, 0)
		value.SetPadding(6, 8)
		value.SetBackground(ui.White)
		value.SetBorder(1, ui.Silver)
		value.SetBorderRadius(8)
	}), ui.NewDocumentText("break-word: superlongtoken_without_spaces_should_break_here", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(11)
		value.SetLineHeight(15)
		value.SetOverflowWrap(ui.OverflowWrapBreakWord)
	})))
	textPreWrap := ui.NewDocumentText("pre-wrap:\nline 1\n    keeps indent\nline 3", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetWidth(180)
		value.SetMargin(0, 0, 8, 0)
		value.SetPadding(6, 8)
		value.SetBackground(ui.White)
		value.SetBorder(1, ui.Silver)
		value.SetBorderRadius(8)
		value.SetFontSize(11)
		value.SetLineHeight(15)
		value.SetWhiteSpace(ui.WhiteSpacePreWrap)
	}))
	visibilityBefore := ui.NewDocumentElement("vis-before", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 4, 0)
		value.SetPadding(4, 8)
		value.SetBorderRadius(999)
		value.SetBackground(ui.Navy)
	}), ui.NewDocumentText("Visible before", style(func(value *ui.Style) {
		value.SetForeground(ui.White)
		value.SetFontSize(11)
	})))
	visibilityHidden := ui.NewDocumentElement("vis-hidden", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 4, 0)
		value.SetPadding(4, 8)
		value.SetBorderRadius(999)
		value.SetBackground(ui.Teal)
		value.SetVisibility(ui.VisibilityHidden)
	}), ui.NewDocumentText("Hidden gap stays reserved", style(func(value *ui.Style) {
		value.SetForeground(ui.White)
		value.SetFontSize(11)
	})))
	visibilityAfter := ui.NewDocumentElement("vis-after", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 8, 0)
		value.SetPadding(4, 8)
		value.SetBorderRadius(999)
		value.SetBackground(ui.Maroon)
	}), ui.NewDocumentText("Visible after", style(func(value *ui.Style) {
		value.SetForeground(ui.White)
		value.SetFontSize(11)
	})))
	styleLabDeferred := ui.NewDocumentText("The only deferred case now is custom font loading; the pre-wrap block itself is enabled again.", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(8, 0, 8, 0)
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
	}))
	notes := ui.NewDocumentElement("notes", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(8, 0, 0, 0)
		value.SetPadding(8, 10)
		value.SetBorderRadius(10)
		value.SetBackground(ui.White)
		value.SetBorder(1, ui.Silver)
	}), ui.NewDocumentText("Host checks", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(12)
		value.SetMargin(0, 0, 4, 0)
	})), ui.NewDocumentText("1. Wheel inside this area should scroll only the embedded document.", style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
		value.SetMargin(0, 0, 3, 0)
	})), ui.NewDocumentText("2. Click once inside the document, then use Tab and Shift+Tab to move focus between action cards.", style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
		value.SetMargin(0, 0, 3, 0)
	})), ui.NewDocumentText("3. Press Enter or Space on a focused card to trigger its click handler.", style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
		value.SetMargin(0, 0, 3, 0)
	})), ui.NewDocumentText("4. Hover, active and focus styles are now handled inside the document host, not by native widgets.", style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
		value.SetMargin(0, 0, 3, 0)
	})), ui.NewDocumentText("5. Native buttons below still mutate the same shared state, so both frontends stay in sync.", style(func(value *ui.Style) {
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
	})))

	documentRoot := ui.NewDocumentElement("root", style(func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
	}), hero, statusCard, ui.NewDocumentText("Document actions", style(func(value *ui.Style) {
		value.SetForeground(ui.Navy)
		value.SetFontSize(13)
		value.SetMargin(0, 0, 6, 0)
	})), incrementCard, resetCard, accentCard, detailsToggleCard, outlineCard, detailsCard, styleLabTitle, borderLab, boxBorderBox, boxContentBox, textLab, textNormal, textNoWrap, textBreakWord, textPreWrap, visibilityBefore, visibilityHidden, visibilityAfter, styleLabDeferred, notes)
	document := ui.NewDocument(documentRoot)
	view := ui.CreateDocumentView(document)
	view.Style.SetDisplay(ui.DisplayBlock)
	view.Style.SetPadding(10)
	view.Style.SetHeight(210)
	view.Style.SetOverflow(ui.OverflowAuto)
	view.Style.SetScrollbarWidth(8)
	view.Style.SetBorderRadius(12)
	view.Style.SetBorder(1, ui.Silver)
	view.Style.SetBackground(ui.White)
	view.Style.SetMargin(0, 0, 8, 0)
	view.StyleFocus = style(func(value *ui.Style) {
		value.SetBorderColor(ui.Blue)
		value.SetBorderWidth(2)
	})

	panelHint := elements.Label("Clicks, wheel, Tab and Enter/Space now route inside the document host before they fall back to the window.")
	apply(panelHint, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
	})

	panel.Append(panelTitle)
	panel.Append(view)
	panel.Append(panelHint)

	controls := ui.CreateBox()
	apply(controls, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetPadding(10)
		value.SetBorderRadius(12)
		value.SetBackground(ui.White)
		value.SetBorder(1, ui.Silver)
	})

	controlsTitle := elements.Label("Native Controls")
	apply(controlsTitle, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetForeground(ui.Navy)
		value.SetMargin(0, 0, 8, 0)
		value.SetFontSize(14)
	})

	nativeStatus := elements.Label("")
	apply(nativeStatus, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(0, 0, 8, 0)
		value.SetFontSize(12)
	})

	plusFive := elements.Button("Native +5")
	styleButton(plusFive)
	plusFive.OnClick = func() {
		state.count += 5
		updateState()
	}

	toggleAccent := elements.Button("Accent")
	styleButton(toggleAccent)
	toggleAccent.OnClick = func() {
		state.accent = !state.accent
		updateState()
	}

	toggleDetails := elements.Button("Details")
	styleButton(toggleDetails)
	toggleDetails.OnClick = func() {
		state.details = !state.details
		updateState()
	}

	reset := elements.Button("Reset")
	styleButton(reset)
	reset.OnClick = func() {
		state.count = 0
		updateState()
	}

	closeButton := elements.Button("Close")
	styleButton(closeButton)
	closeButton.SetBackground(ui.Maroon)
	closeButton.SetForeground(ui.White)
	closeButton.SetBorderColor(ui.Maroon)
	closeButton.SetBorderWidth(1)
	closeButton.OnClick = func() {
		window.Close()
	}

	footer := elements.Label("Use this path for native apps now; later the browser frontend can map HTML/CSS into the same document/fragments/display-list pipeline.")
	apply(footer, func(value *ui.Style) {
		value.SetDisplay(ui.DisplayBlock)
		value.SetMargin(10, 0, 0, 0)
		value.SetForeground(ui.Gray)
		value.SetFontSize(11)
	})

	updateState = func() {
		countValue.Text = strconv.Itoa(state.count)
		if state.accent {
			accentValue.Text = "enabled"
			accentValue.Style.SetForeground(ui.Teal)
			statusCard.Style.SetBorderColor(ui.Teal)
			statusCard.Style.SetBackground(ui.Aqua)
			view.Style.SetBorderColor(ui.Teal)
			nativeStatus.SetForeground(ui.Teal)
		} else {
			accentValue.Text = "disabled"
			accentValue.Style.SetForeground(ui.Gray)
			statusCard.Style.SetBorderColor(ui.Silver)
			statusCard.Style.SetBackground(ui.White)
			view.Style.SetBorderColor(ui.Silver)
			nativeStatus.SetForeground(ui.Navy)
		}
		if state.details {
			detailsCard.Style.SetDisplay(ui.DisplayBlock)
			detailsBody.Text = "Count=" + strconv.Itoa(state.count) + ", accent=" + strconv.FormatBool(state.accent) + ", details=" + strconv.FormatBool(state.details) + "."
		} else {
			detailsCard.Style.SetDisplay(ui.DisplayNone)
			detailsBody.Text = ""
		}
		nativeStatus.SetText(window, "Count "+strconv.Itoa(state.count)+" | accent "+accentValue.Text+" | details "+strconv.FormatBool(state.details))
		document.MarkLayoutDirty()
	}

	updateState()

	controls.Append(controlsTitle)
	controls.Append(nativeStatus)
	controls.Append(plusFive)
	controls.Append(toggleAccent)
	controls.Append(toggleDetails)
	controls.Append(reset)
	controls.Append(closeButton)
	controls.Append(footer)

	root.Append(header)
	root.Append(panel)
	root.Append(controls)
	window.Append(root)
	window.Start()
}

func main() {
	Run()
}
