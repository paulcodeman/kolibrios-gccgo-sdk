package main

import (
	"github.com/mjl-/duit"
	"github.com/psilva261/mycel/browser/duitx"
)

func main() {
	dui, err := duit.NewDUI("duitxdemo", nil)
	if err != nil {
		return
	}

	location := &duit.Field{Text: "https://9p.io/"}
	status := &duitx.Label{Text: "browser/duitx compatibility layer"}
	clicks := 0
	button := &duit.Button{
		Text: "Ping",
		Click: func() duit.Event {
			clicks++
			status.Text = "clicks: " + itoa(clicks) + " target: " + location.Text
			return duit.Event{
				Consumed:   true,
				NeedLayout: true,
				NeedDraw:   true,
			}
		},
	}

	content := duitx.NewBox(
		&duitx.Label{Text: "KolibriOS upstream mycel browser/duitx"},
		&duit.Grid{
			Columns: 2,
			Halign:  []duit.Halign{duit.HalignLeft, duit.HalignLeft},
			Valign:  []duit.Valign{duit.ValignMiddle, duit.ValignMiddle},
			Padding: duit.NSpace(2, duit.SpaceXY(6, 4)),
			Kids: duit.NewKids(
				&duit.Label{Text: "URL"},
				location,
				&duit.Label{Text: "Action"},
				button,
			),
		},
		status,
	)

	dui.Top.UI = duitx.NewScroll(dui, &duitx.Box{
		Padding: duit.SpaceXY(16, 12),
		Width:   -1,
		Kids:    duit.NewKids(content),
	})
	dui.Top.Layout = duit.Dirty
	dui.Top.Draw = duit.Dirty
	dui.Render()
	dui.Run()
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	if value < 0 {
		return "-" + itoa(-value)
	}
	var digits [20]byte
	n := len(digits)
	for value > 0 {
		n--
		digits[n] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[n:])
}
