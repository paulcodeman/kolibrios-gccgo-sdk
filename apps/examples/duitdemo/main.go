package main

import "github.com/mjl-/duit"

func main() {
	dui, err := duit.NewDUI("duitdemo", nil)
	if err != nil {
		return
	}
	status := &duit.Label{
		Text: "draw/duit compatibility layer",
	}
	location := &duit.Field{
		Text: "https://9p.io/",
	}
	clicks := 0
	button := &duit.Button{
		Text: "Ping",
		Click: func() duit.Event {
			clicks++
			status.Text = "clicks: " + itoa(clicks) + " target: " + location.Text
			dui.MarkLayout(status)
			dui.MarkDraw(status)
			return duit.Event{
				Consumed: true,
			}
		},
	}
	content := duit.NewBox(
		&duit.Label{Text: "KolibriOS dui sandbox"},
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
	dui.Top.UI = duit.NewScroll(&duit.Box{
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
