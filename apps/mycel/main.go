package main

import (
	"net/url"
	"os"
	"runtime"
	"strings"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
	"github.com/psilva261/mycel/browser"
	"github.com/psilva261/mycel/js"
	log "github.com/psilva261/mycel/logger"
	"github.com/psilva261/mycel/style"
	"kos"
)

const (
	appTitle         = "Mycel"
	defaultURL       = "about:welcome"
	defaultTTF       = "assets/OpenSans-Regular.ttf"
	defaultCABundle  = "assets/ca-bundle.pem"
	defaultDownload  = "/tmp0/1/mycel.download"
	windowDimensions = "960x700"
)

var (
	dui *duit.DUI
	b   *browser.Browser
	loc = defaultURL
	v   View
)

type View interface {
	Render() []*duit.Kid
}

type Nav struct {
	LocationField *duit.Field
	StatusBar     *duit.Label
}

func NewNav() *Nav {
	n := &Nav{
		StatusBar: &duit.Label{Text: ""},
	}
	n.LocationField = &duit.Field{
		Text: loc,
		Font: browser.Style.Font(),
		Keys: n.keys,
	}
	return n
}

func (n *Nav) keys(k rune, m draw.Mouse) (e duit.Event) {
	if k != browser.EnterKey || b == nil {
		return
	}
	addr := strings.TrimSpace(n.LocationField.Text)
	if addr == "" {
		return
	}
	lower := strings.ToLower(addr)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "about:") {
		addr = "https://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		log.Errorf("parse url: %v", err)
		return
	}
	return b.LoadUrl(u)
}

func (n *Nav) Render() []*duit.Kid {
	uis := []duit.UI{
		&duit.Grid{
			Columns: 3,
			Halign:  []duit.Halign{duit.HalignLeft, duit.HalignLeft, duit.HalignLeft},
			Valign:  []duit.Valign{duit.ValignMiddle, duit.ValignMiddle, duit.ValignMiddle},
			Kids: duit.NewKids(
				&duit.Button{
					Text:  "Back",
					Font:  browser.Style.Font(),
					Click: b.Back,
				},
				&duit.Button{
					Text: "Stop",
					Font: browser.Style.Font(),
					Click: func() (e duit.Event) {
						b.Cancel()
						e.Consumed = true
						return
					},
				},
				&duit.Box{
					Kids: duit.NewKids(n.LocationField),
				},
			),
		},
		n.StatusBar,
	}
	if b != nil {
		uis = append(uis, b.Website)
	}
	return duit.NewKids(uis...)
}

func setDefaults() {
	if os.Getenv("DRAW_DEFAULT_TTF") == "" {
		_ = os.Setenv("DRAW_DEFAULT_TTF", defaultTTF)
	}
	if os.Getenv("SSL_CERT_FILE") == "" {
		_ = os.Setenv("SSL_CERT_FILE", defaultCABundle)
	}
	if os.Getenv("MYCEL_DEBUG_LOG") == "" {
		log.SetQuiet()
	}
}

func updateViewport() {
	if dui == nil || dui.Display == nil {
		return
	}
	size := dui.Display.ScreenImage.R.Size()
	style.SetViewport(size.X/dui.Scale(1), size.Y/dui.Scale(1))
}

func render() {
	white, err := dui.Display.AllocImage(draw.Rect(0, 0, 1, 1), draw.ARGB32, true, 0xffffffff)
	if err != nil {
		log.Errorf("alloc white: %v", err)
	}
	dui.Top.UI = &duit.Box{
		Kids:       v.Render(),
		Background: white,
	}
	dui.MarkLayout(dui.Top.UI)
	dui.MarkDraw(dui.Top.UI)
	dui.Render()
}

func handleAsync() bool {
	select {
	case fn := <-dui.Call:
		if fn != nil {
			fn()
		}
		return true
	case nextLoc := <-b.LocCh:
		loc = nextLoc
		if nav, ok := v.(*Nav); ok {
			nav.LocationField.Text = loc
			dui.MarkDraw(nav.LocationField)
			dui.Render()
		}
		return true
	case msg := <-b.StatusCh:
		if nav, ok := v.(*Nav); ok {
			nav.StatusBar.Text = msg
			dui.MarkLayout(dui.Top.UI)
			dui.MarkDraw(dui.Top.UI)
			dui.Render()
		}
		return true
	case err := <-dui.Error:
		if err != nil {
			log.Printf("duit: %v", err)
		}
		return true
	default:
		return false
	}
}

func main() {
	runtime.LockOSThread()
	setDefaults()
	browser.EnableNoScriptTag = true

	var err error
	dui, err = duit.NewDUI(appTitle, &duit.DUIOpts{Dimensions: windowDimensions})
	if err != nil {
		log.Fatalf("new dui: %v", err)
	}
	style.Init(dui)
	updateViewport()

	v = NewNav()
	render()

	b = browser.NewBrowser(dui, loc)
	b.Download = func(res chan *string) {
		target := defaultDownload
		res <- &target
	}

	v = NewNav()
	render()

	for {
		drained := false
		for handleAsync() {
			drained = true
		}
		if !dui.StepPoll() {
			break
		}
		updateViewport()
		if !drained {
			kos.SleepCentiseconds(1)
		}
	}

	js.StopAll()
}
