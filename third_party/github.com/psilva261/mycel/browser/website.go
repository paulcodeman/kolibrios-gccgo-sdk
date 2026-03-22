package browser

import (
	"9fans.net/go/draw"
	"github.com/mjl-/duit"
	"github.com/psilva261/mycel"
	"github.com/psilva261/mycel/browser/duitx"
	//"github.com/psilva261/mycel/browser/fs"
	"github.com/psilva261/mycel/js"
	log "github.com/psilva261/mycel/logger"
	"github.com/psilva261/mycel/nodes"
	"github.com/psilva261/mycel/style"
	textencoding "golang.org/x/text/encoding"
	"net/html"
	"net/url"
	"strings"
)

const (
	InitialLayout = iota
	ClickRelayout
)

type Website struct {
	b *Browser
	duit.UI
	mycel.ContentType
}

func bodyCanvasBackground(body *nodes.Node) *draw.Image {
	if body == nil {
		return nil
	}
	bg, err := body.BoxBackground()
	if err != nil || bg == nil {
		return nil
	}
	for _, key := range []string{
		"background",
		"background-color",
		"background-image",
	} {
		delete(body.Map.Declarations, key)
	}
	return bg
}

func (w *Website) layout(f mycel.Fetcher, htm string, layouting int) {
	defer func() {
		select {
		case w.b.StatusCh <- "":
		default:
		}
	}()
	pass := func(htm string, csss ...string) (*html.Node, map[*html.Node]style.Map) {
		if f.Ctx().Err() != nil {
			return nil, nil
		}

		if debugPrintHtml {
			log.Printf("%v\n", htm)
		}

		var doc *html.Node
		var err error
		doc, err = html.ParseWithOptions(
			strings.NewReader(htm),
			html.ParseOptionEnableScripting(ExperimentalJsInsecure),
		)
		if err != nil {
			panic(err.Error())
		}

		log.Printf("Retrieving CSS Rules...")
		var cssSize int
		nodeMap := make(map[*html.Node]style.Map)
		for i, css := range csss {

			log.Printf("CSS size %v kB", cssSize/1024)

			nm, err := style.FetchNodeMap(doc, css)
			if err == nil {
				if debugPrintHtml {
					log.Printf("%v", nm)
				}
				style.MergeNodeMaps(nodeMap, nm)
			} else {
				log.Errorf("%v/css/%v.css: Fetch CSS Rules failed: %v", mycel.PathPrefix, i, err)
			}
		}

		return doc, nodeMap
	}

	log.Printf("1st pass")
	doc, _ := pass(htm)

	log.Printf("2nd pass")
	log.Printf("Download style...")
	csss := cssSrcs(f, doc)
	doc, nodeMap := pass(htm, csss...)

	// 3rd pass is only needed initially to load the scripts and set the js VM
	// state. During subsequent calls from click handlers that state is kept.
	var scripts []string
	if ExperimentalJsInsecure && layouting != ClickRelayout {
		var (
			jsProcessed string
			changed     bool
			err         error
		)

		log.Printf("3rd pass")
		nt := nodes.NewNodeTree(doc, style.Map{}, nodeMap, nil)
		jsSrcs := js.Srcs(nt)
		downloads := make(map[string]string)
		for _, src := range jsSrcs {
			url, err := f.LinkedUrl(src)
			if err != nil {
				log.Printf("error parsing %v", src)
				continue
			}
			log.Printf("Download %v", url)
			buf, _, err := f.Get(url)
			if err != nil {
				log.Printf("error downloading %v", url)
				continue
			}
			downloads[src] = string(buf)
		}
		scripts = js.Scripts(nt, downloads)
		w.b.fs.Update(f.Origin().String(), htm, csss, scripts)
		w.b.fs.SetDOM(nt)
		log.Infof("JS pipeline start")
		w.b.js.Stop()
		w.b.js, jsProcessed, changed, err = processJS2(f)
		if changed && err == nil {
			htm = jsProcessed
			if debugPrintHtml {
				log.Printf("%v\n", jsProcessed)
			}
			doc, nodeMap = pass(htm, csss...)
		} else if err != nil {
			log.Errorf("JS error: %v", err)
		}
		log.Infof("JS pipeline end")
	}
	if f.Ctx().Err() != nil {
		return
	}
	var countHtmlNodes func(*html.Node) int
	countHtmlNodes = func(n *html.Node) (num int) {
		num++
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			num += countHtmlNodes(c)
		}
		return
	}
	log.Printf("%v html nodes found...", countHtmlNodes(doc))

	body := grep(doc, "body")
	if body == nil {
		// TODO: handle frameset without noframes
		log.Errorf("html has no body")
		return
	}

	log.Printf("Layout website...")
	nt := nodes.NewNodeTree(body, style.Map{}, nodeMap, &nodes.Node{})
	canvasBg := bodyCanvasBackground(nt)
	if scroller != nil {
		scroller.Free()
		scroller = nil
	}
	var rootUI duit.UI
	rootUI = NodeToBox(0, w.b, nt)
	if rootUI == nil {
		rootUI = &duitx.Box{
			Width:      -1,
			Height:     -1,
			Background: dui.Background,
		}
	}
	if canvasBg != nil {
		rootUI = &duitx.Box{
			Width:      -1,
			Height:     -1,
			Background: canvasBg,
			Kids:       duit.NewKids(rootUI),
		}
	}
	scroller = duitx.NewScroll(dui, rootUI)
	numElements := 0
	TraverseTree(scroller, func(ui duit.UI) {
		numElements++
	})
	w.UI = scroller
	log.Printf("Layouting done (%v elements created)", numElements)
	if numElements < 10 {
		log.Errorf("Less than 10 elements layouted, seems css processing failed. Will layout without css")
		nt = nodes.NewNodeTree(body, style.Map{}, make(map[*html.Node]style.Map), nil)
		rootUI = NodeToBox(0, w.b, nt)
		if rootUI == nil {
			rootUI = &duitx.Box{
				Width:      -1,
				Height:     -1,
				Background: dui.Background,
			}
		}
		scroller = duitx.NewScroll(dui, rootUI)
		w.UI = scroller
	}

	w.b.fs.Update(f.Origin().String(), htm, csss, scripts)
	w.b.fs.SetDOM(nt)
}

func cssSrcs(f mycel.Fetcher, doc *html.Node) (srcs []string) {
	srcs = make([]string, 0, 20)
	srcs = append(srcs, style.AddOnCSS)
	ntAll := nodes.NewNodeTree(doc, style.Map{}, make(map[*html.Node]style.Map), nil)
	ntAll.Traverse(func(r int, n *nodes.Node) {
		switch n.Data() {
		case "style":
			if t := strings.ToLower(n.Attr("type")); t == "" || t == "text/css" {
				srcs = append(srcs, n.ContentString(true))
			}
		case "link":
			isStylesheet := n.Attr("rel") == "stylesheet"
			if m := n.Attr("media"); m != "" {
				matches, errMatch := style.MatchQuery(m, style.MediaValues)
				if errMatch != nil {
					log.Errorf("match query %v: %v", m, errMatch)
				}
				if !matches {
					return
				}
			}
			href := n.Attr("href")
			if isStylesheet {
				url, err := f.LinkedUrl(href)
				if err != nil {
					log.Errorf("error parsing %v", href)
					return
				}
				buf, contentType, err := f.Get(url)
				if err != nil {
					log.Errorf("error downloading %v", url)
					return
				}
				if contentType.IsCSS() {
					srcs = append(srcs, string(buf))
				} else {
					log.Printf("css: unexpected %v", contentType)
				}
			}
		}
	})
	return
}

func formData(n, submitBtn *html.Node) (data url.Values) {
	data = make(url.Values)
	nm := attr(*n, "name")

	switch n.Data {
	case "input", "select":
		if attr(*n, "type") == "submit" && n != submitBtn {
			return
		}
		if nm != "" {
			data.Set(nm, attr(*n, "value"))
		}
	case "textarea":
		nn := nodes.NewNodeTree(n, style.Map{}, make(map[*html.Node]style.Map), nil)

		if nm != "" {
			data.Set(nm, nn.ContentString(false))
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		for k, vs := range formData(c, submitBtn) {
			for _, v := range vs {
				data.Add(k, v)
			}
		}
	}

	return
}

func escapeValues(ct mycel.ContentType, q url.Values) (qe url.Values) {
	qe = make(url.Values)
	enc := textencoding.HTMLEscapeUnsupported(ct.Encoding().NewEncoder())

	for k, vs := range q {
		ke, err := enc.String(k)
		if err != nil {
			log.Errorf("string: %v", err)
			ke = k
		}
		for _, v := range vs {
			ve, err := enc.String(v)
			if err != nil {
				log.Errorf("string: %v", err)
				ve = v
			}
			qe.Add(ke, ve)
		}
	}

	return
}

func (b *Browser) submit(form *html.Node, submitBtn *html.Node) {
	var err error
	var buf []byte
	var contentType mycel.ContentType

	method := "GET" // TODO
	if m := attr(*form, "method"); m != "" {
		method = strings.ToUpper(m)
	}
	uri := b.URL()
	if action := attr(*form, "action"); action != "" {
		uri, err = b.LinkedUrl(action)
		if err != nil {
			log.Printf("error parsing %v", action)
			return
		}
	}

	if method == "GET" {
		q := uri.Query()
		for k, vs := range formData(form, submitBtn) {
			if len(vs) == 0 {
				continue
			}
			q.Set(k, vs[0]) // TODO: what is with the rest?
		}
		uri.RawQuery = escapeValues(b.Website.ContentType, q).Encode()
		buf, contentType, err = b.get(uri, true)
	} else {
		buf, contentType, err = b.PostForm(uri, formData(form, submitBtn))
	}

	if err != nil {
		log.Errorf("submit form: %v", err)
		return
	}

	if !contentType.IsHTML() {
		log.Errorf("post: unexpected %v", contentType)
		return
	}

	b.render(contentType, buf, b.currentLoadSeq())
}
