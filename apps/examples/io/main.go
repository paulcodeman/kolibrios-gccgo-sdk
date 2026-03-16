package main

import (
	"io"
	"os"
	"ui/elements"

	"kos"
	"ui"
)

const (
	ioButtonExit    kos.ButtonID = 1
	ioButtonRefresh kos.ButtonID = 2

	ioWindowTitle  = "KolibriOS IO Demo"
	ioWindowX      = 244
	ioWindowY      = 150
	ioWindowWidth  = 792
	ioWindowHeight = 320

	ioProbePath      = "/sys/default.skn"
	ioProbeChunkSize = 13
	ioProbeMaxBytes  = 96
)

type chunkReader struct {
	data      []byte
	chunkSize int
	offset    int
	sawEOF    bool
}

func newChunkReader(data []byte, chunkSize int) *chunkReader {
	return &chunkReader{
		data:      data,
		chunkSize: chunkSize,
	}
}

func (reader *chunkReader) Read(buffer []byte) (int, error) {
	if reader.offset >= len(reader.data) {
		reader.sawEOF = true
		return 0, io.EOF
	}

	read := reader.chunkSize
	if read > len(buffer) {
		read = len(buffer)
	}

	remaining := len(reader.data) - reader.offset
	if read > remaining {
		read = remaining
	}

	copy(buffer[:read], reader.data[reader.offset:reader.offset+read])
	reader.offset += read
	return read, nil
}

type bufferWriter struct {
	data []byte
}

func (writer *bufferWriter) Write(buffer []byte) (int, error) {
	writer.data = append(writer.data, buffer...)
	return len(buffer), nil
}

const (
	ioPreviewLimit = 28
)

type App struct {
	summary    string
	readLine   string
	copyLine   string
	writeLine  string
	cwdLine    string
	infoLine   string
	ok         bool
	refreshBtn *ui.Element
}

func NewApp() App {
	refresh := elements.ButtonAt(ioButtonRefresh, "Refresh", 28, 264)
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
	case ioButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case ioButtonExit:
		os.Exit(0)
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(ioButtonExit, "Exit", 170, 264)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(ioWindowX, ioWindowY, ioWindowWidth, ioWindowHeight, ioWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports the ordinary io package: import \"io\"")
	kos.DrawText(28, 92, ui.Aqua, app.readLine)
	kos.DrawText(28, 114, ui.Lime, app.copyLine)
	kos.DrawText(28, 136, ui.Yellow, app.writeLine)
	kos.DrawText(28, 158, ui.White, app.cwdLine)
	kos.DrawText(28, 180, ui.Black, app.infoLine)
	app.refreshBtn.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	data, err := os.ReadFile(ioProbePath)
	if err != nil {
		app.ok = false
		app.summary = "io probe failed / file read unavailable"
		app.infoLine = "Info: " + ioProbePath + " / " + err.Error()
		return
	}

	if len(data) > ioProbeMaxBytes {
		data = data[:ioProbeMaxBytes]
	}

	readReader := newChunkReader(data, ioProbeChunkSize)
	readData, readErr := io.ReadAll(readReader)
	copyReader := newChunkReader(data, ioProbeChunkSize)
	copyTarget := &bufferWriter{}
	written, copyErr := io.Copy(copyTarget, copyReader)
	cwdWriter := &bufferWriter{}
	currentFolder, err := os.Getwd()
	if err != nil {
		app.fail("getwd failed")
		return
	}
	cwdWritten, writeErr := io.WriteString(cwdWriter, currentFolder)

	app.readLine = "ReadAll: " + formatUint32(uint32(len(readData))) + " bytes / eof " + formatBool(readReader.sawEOF) + " / preview " + previewText(readData)
	app.copyLine = "Copy: " + formatInt64(written) + " bytes / eof " + formatBool(copyReader.sawEOF) + " / preview " + previewText(copyTarget.data)
	app.writeLine = "WriteString: " + formatInt(cwdWritten) + " bytes / value " + previewText(cwdWriter.data)
	app.cwdLine = "Current folder: " + currentFolder

	if readErr != nil {
		app.fail("ReadAll returned error")
		return
	}
	if !readReader.sawEOF || !equalBytes(readData, data) {
		app.fail("ReadAll mismatch")
		return
	}
	if copyErr != nil {
		app.fail("Copy returned error")
		return
	}
	if !copyReader.sawEOF || written != int64(len(data)) || !equalBytes(copyTarget.data, data) {
		app.fail("Copy mismatch")
		return
	}
	if writeErr != nil || cwdWritten != len(currentFolder) || string(cwdWriter.data) != currentFolder {
		app.fail("WriteString mismatch")
		return
	}

	info, err := os.Stat(ioProbePath)
	if err != nil {
		app.ok = false
		app.summary = "io probe failed / file info unavailable"
		app.infoLine = "Info: " + ioProbePath + " / " + err.Error()
		return
	}
	rawInfo, ok := info.Sys().(kos.FileInfo)
	if !ok {
		app.fail("stat sys payload mismatch")
		return
	}

	app.ok = true
	app.summary = "io probe ok / ordinary import io package resolved"
	app.infoLine = "Info: size " + formatHex64(uint64(info.Size())) + " bytes / attrs " + formatHex32(uint32(rawInfo.Attributes))
}

func (app *App) fail(detail string) {
	app.ok = false
	app.summary = "io probe failed / " + detail
	app.infoLine = "Info: unavailable"
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}

	return ui.Red
}

func previewText(data []byte) string {
	limit := len(data)
	if limit > ioPreviewLimit {
		limit = ioPreviewLimit
	}

	preview := ""
	for index := 0; index < limit; index++ {
		value := data[index]
		if value < 32 || value > 126 {
			preview += "."
			continue
		}

		preview += string([]byte{value})
	}

	return preview
}

func main() {
	app := NewApp()
	app.Run()
}
