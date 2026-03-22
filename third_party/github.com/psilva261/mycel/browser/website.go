package browser

import (
	"9fans.net/go/draw"
	"fmt"
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

type preparedLayout struct {
	ui       duit.UI
	scroller *duitx.Scroll
}

type preparedDocument struct {
	origin  string
	htm     string
	body    *html.Node
	nt      *nodes.Node
	csss    []string
	scripts []string
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
	doc, err := w.prepareDocument(f, htm, layouting)
	if err != nil {
		log.Errorf("layout: %v", err)
		return
	}
	prepared, nt, err := w.buildPreparedLayout(doc)
	if err != nil {
		log.Errorf("layout build: %v", err)
		return
	}
	if scroller != nil {
		scroller.Free()
		scroller = nil
	}
	scroller = prepared.scroller
	w.UI = prepared.ui
	w.b.fs.Update(doc.origin, doc.htm, doc.csss, doc.scripts)
	w.b.fs.SetDOM(nt)
}

func (w *Website) prepareLayout(f mycel.Fetcher, htm string, layouting int) (prepared preparedLayout, err error) {
	doc, err := w.prepareDocument(f, htm, layouting)
	if err != nil {
		return prepared, err
	}
	prepared, _, err = w.buildPreparedLayout(doc)
	return prepared, err
}

func (w *Website) prepareDocument(f mycel.Fetcher, htm string, layouting int) (prepared preparedDocument, err error) {
	phase := "start"
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("prepare document panic (%s): %v", phase, rec)
		}
	}()

	prepared.origin = f.Origin().String()
	prepared.htm = htm
	pass := func(htm string, csss ...string) (*html.Node, map[*html.Node]style.Map, error) {
		if f.Ctx().Err() != nil {
			return nil, nil, f.Ctx().Err()
		}

		if debugPrintHtml {
			log.Printf("%v\n", htm)
		}

		var doc *html.Node
		var err error
		phase = "parse html"
		doc, err = html.ParseWithOptions(
			strings.NewReader(htm),
			html.ParseOptionEnableScripting(ExperimentalJsInsecure),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("parse html: %w", err)
		}

		log.Printf("Retrieving CSS Rules...")
		var cssSize int
		nodeMap := make(map[*html.Node]style.Map)
		for i, css := range csss {
			phase = fmt.Sprintf("apply css sheet %d", i)

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

		return doc, nodeMap, nil
	}

	phase = "html pass 1"
	select {
	case w.b.StatusCh <- "HTML pass 1...":
	default:
	}
	log.Printf("1st pass")
	doc, _, err := pass(htm)
	if err != nil {
		return prepared, err
	}

	phase = "scan styles"
	select {
	case w.b.StatusCh <- "Scan styles...":
	default:
	}
	log.Printf("2nd pass")
	log.Printf("Download style...")
	csss := cssSrcs(f, doc)
	phase = "apply css"
	select {
	case w.b.StatusCh <- "Apply CSS...":
	default:
	}
	doc, nodeMap, err := pass(htm, csss...)
	if err != nil {
		return prepared, err
	}
	prepared.csss = csss

	// 3rd pass is only needed initially to load the scripts and set the js VM
	// state. During subsequent calls from click handlers that state is kept.
	var scripts []string
	if ExperimentalJsInsecure && layouting != ClickRelayout {
		phase = "javascript"
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
				doc, nodeMap, err = pass(htm, csss...)
				if err != nil {
					return prepared, err
				}
			} else if err != nil {
				log.Errorf("JS error: %v", err)
			}
			log.Infof("JS pipeline end")
		}
	prepared.scripts = scripts
	if f.Ctx().Err() != nil {
		return
	}
	if doc == nil {
		return prepared, fmt.Errorf("document is nil after %s", phase)
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
		return prepared, fmt.Errorf("html has no body")
	}

	prepared.body = body

	phase = "build dom"
	select {
	case w.b.StatusCh <- "Build DOM...":
	default:
	}
	log.Printf("Layout website...")
	nt := nodes.NewNodeTree(body, style.Map{}, nodeMap, &nodes.Node{})
	prepared.nt = nt
	return prepared, nil
}

func (w *Website) buildPreparedLayout(prepared preparedDocument) (layout preparedLayout, nt *nodes.Node, err error) {
	phase := "start"
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("build layout panic (%s): %v", phase, rec)
		}
	}()

	if prepared.body == nil || prepared.nt == nil {
		return layout, nil, fmt.Errorf("prepared document incomplete")
	}
	log.Printf("Layout website...")
	nt = prepared.nt
	phase = "body background"
	canvasBg := bodyCanvasBackground(nt)
	var rootUI duit.UI
	phase = "node to box"
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
	phase = "scroll"
	scroll := duitx.NewScroll(dui, rootUI)
	numElements := 0
	phase = "traverse tree"
	TraverseTree(scroll, func(ui duit.UI) {
		numElements++
	})
	log.Printf("Layouting done (%v elements created)", numElements)
	if numElements < 10 {
		log.Errorf("Less than 10 elements layouted, seems css processing failed. Will layout without css")
		phase = "fallback node tree"
		nt = nodes.NewNodeTree(prepared.body, style.Map{}, make(map[*html.Node]style.Map), nil)
		phase = "fallback node to box"
		rootUI = NodeToBox(0, w.b, nt)
		if rootUI == nil {
			rootUI = &duitx.Box{
				Width:      -1,
				Height:     -1,
				Background: dui.Background,
			}
		}
		phase = "fallback scroll"
		scroll = duitx.NewScroll(dui, rootUI)
	}
	layout.ui = scroll
	layout.scroller = scroll
	return layout, nt, nil
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
