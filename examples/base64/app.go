package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	base64ButtonExit    kos.ButtonID = 1
	base64ButtonRefresh kos.ButtonID = 2

	base64WindowTitle  = "KolibriOS Base64 Demo"
	base64WindowX      = 208
	base64WindowY      = 126
	base64WindowWidth  = 900
	base64WindowHeight = 344
)

type App struct {
	summary    string
	stdLine    string
	appendLine string
	rawLine    string
	streamLine string
	dllLine    string
	infoLine   string
	ok         bool
	refreshBtn *ui.Element
}

type chunkReader struct {
	text   string
	offset int
	chunk  int
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(base64ButtonRefresh, "Refresh", 28, 288)
	refresh.SetWidth(116)

	app := App{
		refreshBtn: refresh,
	}
	app.refreshProbe()
	return app
}

func (app *App) Run() {
	for {
		switch kos.WaitEvent() {
		case kos.EventRedraw:
			app.Redraw()
		case kos.EventButton:
			if app.handleButton(kos.CurrentButtonID()) {
				return
			}
		}
	}
}

func (app *App) handleButton(id kos.ButtonID) bool {
	switch id {
	case base64ButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case base64ButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(base64ButtonExit, "Exit", 170, 288)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(base64WindowX, base64WindowY, base64WindowWidth, base64WindowHeight, base64WindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary encoding/base64 package")
	kos.DrawText(28, 92, ui.Aqua, app.stdLine)
	kos.DrawText(28, 114, ui.Lime, app.appendLine)
	kos.DrawText(28, 136, ui.Yellow, app.rawLine)
	kos.DrawText(28, 158, ui.White, app.streamLine)
	kos.DrawText(28, 180, ui.Silver, app.dllLine)
	kos.DrawText(28, 202, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	backend, ok := kos.LoadBase64()
	if !ok {
		app.fail("base64.obj unavailable", "Info: failed to load "+kos.Base64DLLPath)
		return
	}

	const plain = "hello, world"
	const encodedExpected = "aGVsbG8sIHdvcmxk"
	const rawPlain = "go/net?ok=1"

	encoded := base64.StdEncoding.EncodeToString([]byte(plain))
	encodedBuf := make([]byte, base64.StdEncoding.EncodedLen(len(plain)))
	base64.StdEncoding.Encode(encodedBuf, []byte(plain))
	decoded, err := base64.StdEncoding.DecodeString(encodedExpected)
	if err != nil {
		app.fail("DecodeString failed", "Info: "+err.Error())
		return
	}

	appended := string(base64.StdEncoding.AppendEncode([]byte("b64="), []byte(plain)))
	appendedDecoded, err := base64.StdEncoding.AppendDecode([]byte("raw="), []byte(encodedExpected))
	if err != nil {
		app.fail("AppendDecode failed", "Info: "+err.Error())
		return
	}

	rawEncoded := base64.RawURLEncoding.EncodeToString([]byte(rawPlain))
	rawDecoded, err := base64.RawURLEncoding.DecodeString(rawEncoded)
	if err != nil {
		app.fail("raw decode failed", "Info: "+err.Error())
		return
	}

	var stream bytes.Buffer
	streamWriter := base64.NewEncoder(base64.StdEncoding, &stream)
	if _, err = streamWriter.Write([]byte("hello, ")); err != nil {
		app.fail("stream write failed", "Info: "+err.Error())
		return
	}
	if _, err = streamWriter.Write([]byte("world")); err != nil {
		app.fail("stream write tail failed", "Info: "+err.Error())
		return
	}
	if err = streamWriter.Close(); err != nil {
		app.fail("stream close failed", "Info: "+err.Error())
		return
	}

	streamDecoded, err := readStream(
		base64.NewDecoder(base64.StdEncoding, &chunkReader{text: "aGVs\r\nbG8sIHdv\r\ncmxk", chunk: 3}),
		5,
	)
	if err != nil {
		app.fail("stream decode failed", "Info: "+err.Error())
		return
	}
	rawStreamDecoded, err := readStream(
		base64.NewDecoder(base64.RawURLEncoding, &chunkReader{text: rawEncoded, chunk: 2}),
		4,
	)
	if err != nil {
		app.fail("raw stream decode failed", "Info: "+err.Error())
		return
	}
	if _, err = base64.StdEncoding.DecodeString("%%%"); err == nil {
		app.fail("corrupt decode mismatch", "Info: expected CorruptInputError")
		return
	}
	_, err = readStream(
		base64.NewDecoder(base64.StdEncoding, &chunkReader{text: "%%%", chunk: 1}),
		2,
	)
	if err == nil {
		app.fail("stream corrupt decode mismatch", "Info: expected streaming CorruptInputError")
		return
	}

	if encoded != encodedExpected || string(encodedBuf) != encodedExpected || string(decoded) != plain {
		app.fail("std encode/decode mismatch", "Info: EncodeToString/Encode/DecodeString contract mismatch")
		return
	}
	if appended != "b64="+encodedExpected || string(appendedDecoded) != "raw="+plain {
		app.fail("append mismatch", "Info: AppendEncode/AppendDecode contract mismatch")
		return
	}
	if string(rawDecoded) != rawPlain {
		app.fail("raw url mismatch", "Info: RawURLEncoding round-trip mismatch")
		return
	}
	if string(rawStreamDecoded) != rawPlain {
		app.fail("raw stream mismatch", "Info: RawURLEncoding streaming round-trip mismatch")
		return
	}
	if stream.String() != encodedExpected || string(streamDecoded) != plain {
		app.fail("stream mismatch", "Info: NewEncoder/NewDecoder contract mismatch")
		return
	}

	app.ok = true
	app.summary = "base64 probe ok / ordinary import encoding/base64 resolved"
	app.stdLine = fmt.Sprintf("Std: %s -> %s -> %s", plain, encoded, string(decoded))
	app.appendLine = fmt.Sprintf("Append: %s / %s", appended, string(appendedDecoded))
	app.rawLine = fmt.Sprintf("RawURL: %s -> %s -> %s", rawPlain, rawEncoded, string(rawDecoded))
	app.streamLine = fmt.Sprintf("Stream: writer %s / reader %s / raw %s", stream.String(), string(streamDecoded), string(rawStreamDecoded))
	app.dllLine = fmt.Sprintf("DLL: %s / table 0x%x", kos.Base64DLLPath, uint32(backend.ExportTable()))
	app.infoLine = "Info: standard, raw-url, append, and chunked stream flows stay on ordinary encoding/base64"
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "base64 probe failed / " + detail
	app.infoLine = info
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func (reader *chunkReader) Read(p []byte) (int, error) {
	if reader.offset >= len(reader.text) {
		return 0, io.EOF
	}

	limit := len(p)
	if limit > reader.chunk && reader.chunk > 0 {
		limit = reader.chunk
	}
	remaining := len(reader.text) - reader.offset
	if limit > remaining {
		limit = remaining
	}
	if limit <= 0 {
		limit = remaining
	}

	copy(p[:limit], reader.text[reader.offset:reader.offset+limit])
	reader.offset += limit
	return limit, nil
}

func readStream(reader io.Reader, chunk int) ([]byte, error) {
	if chunk <= 0 {
		chunk = 4
	}

	buffer := make([]byte, chunk)
	data := make([]byte, 0, 32)
	for {
		read, err := reader.Read(buffer)
		if read > 0 {
			data = append(data, buffer[:read]...)
		}
		if err != nil {
			if err == io.EOF {
				return data, nil
			}

			return data, err
		}
	}
}
