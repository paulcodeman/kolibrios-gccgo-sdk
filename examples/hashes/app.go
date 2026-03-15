package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"ui/elements"

	"kos"
	"ui"
)

const (
	hashButtonExit    kos.ButtonID = 1
	hashButtonRefresh kos.ButtonID = 2

	hashWindowTitle  = "KolibriOS Hash Demo"
	hashWindowX      = 176
	hashWindowY      = 120
	hashWindowWidth  = 1120
	hashWindowHeight = 360
)

type App struct {
	summary       string
	md5Line       string
	sha1Line      string
	sha256Line    string
	sha512Line    string
	sha512Line2   string
	streamLine    string
	infoLine      string
	ok            bool
	refreshButton *ui.Element
}

func main() {
	app := NewApp()
	app.Run()
}

func NewApp() App {
	refresh := elements.ButtonAt(hashButtonRefresh, "Refresh", 28, 304)
	refresh.SetWidth(116)

	app := App{
		refreshButton: refresh,
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
	case hashButtonRefresh:
		app.refreshProbe()
		app.Redraw()
	case hashButtonExit:
		kos.Exit()
		return true
	}

	return false
}

func (app *App) Redraw() {
	exit := elements.ButtonAt(hashButtonExit, "Exit", 170, 304)
	exit.SetWidth(96)

	kos.BeginRedraw()
	kos.OpenWindow(hashWindowX, hashWindowY, hashWindowWidth, hashWindowHeight, hashWindowTitle)
	kos.DrawText(28, 44, app.summaryColor(), app.summary)
	kos.DrawText(28, 66, ui.Silver, "This sample imports crypto/md5, sha1, sha256, sha512, and encoding/hex")
	kos.DrawText(28, 92, ui.Aqua, app.md5Line)
	kos.DrawText(28, 114, ui.Lime, app.sha1Line)
	kos.DrawText(28, 136, ui.Yellow, app.sha256Line)
	kos.DrawText(28, 158, ui.White, app.sha512Line)
	kos.DrawText(28, 180, ui.White, app.sha512Line2)
	kos.DrawText(28, 202, ui.Silver, app.streamLine)
	kos.DrawText(28, 224, ui.Black, app.infoLine)
	app.refreshButton.Draw()
	exit.Draw()
	kos.EndRedraw()
}

func (app *App) refreshProbe() {
	const message = "hello"
	const expectedMD5 = "5d41402abc4b2a76b9719d911017c592"
	const expectedSHA1 = "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"
	const expectedSHA256 = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	const expectedSHA512 = "9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca72323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043"

	md5Sum := md5.Sum([]byte(message))
	md5Hex := hex.EncodeToString(md5Sum[:])
	md5Stream, err := streamHashHex(md5.New(), message)
	if err != nil {
		app.fail("md5 stream failed", "Info: "+err.Error())
		return
	}
	if md5Hex != expectedMD5 || md5Stream != expectedMD5 || !decodeMatches(expectedMD5, md5Sum[:]) {
		app.fail("md5 mismatch", fmt.Sprintf("Info: sum %s / stream %s", md5Hex, md5Stream))
		return
	}

	sha1Sum := sha1.Sum([]byte(message))
	sha1Hex := hex.EncodeToString(sha1Sum[:])
	sha1Stream, err := streamHashHex(sha1.New(), message)
	if err != nil {
		app.fail("sha1 stream failed", "Info: "+err.Error())
		return
	}
	if sha1Hex != expectedSHA1 || sha1Stream != expectedSHA1 || !decodeMatches(expectedSHA1, sha1Sum[:]) {
		app.fail("sha1 mismatch", fmt.Sprintf("Info: sum %s / stream %s", sha1Hex, sha1Stream))
		return
	}

	sha256Sum := sha256.Sum256([]byte(message))
	sha256Hex := hex.EncodeToString(sha256Sum[:])
	sha256Stream, err := streamHashHex(sha256.New(), message)
	if err != nil {
		app.fail("sha256 stream failed", "Info: "+err.Error())
		return
	}
	if sha256Hex != expectedSHA256 || sha256Stream != expectedSHA256 || !decodeMatches(expectedSHA256, sha256Sum[:]) {
		app.fail("sha256 mismatch", fmt.Sprintf("Info: sum %s / stream %s", sha256Hex, sha256Stream))
		return
	}

	sha512Sum := sha512.Sum512([]byte(message))
	sha512Hex := hex.EncodeToString(sha512Sum[:])
	sha512Stream, err := streamHashHex(sha512.New(), message)
	if err != nil {
		app.fail("sha512 stream failed", "Info: "+err.Error())
		return
	}
	if sha512Hex != expectedSHA512 || sha512Stream != expectedSHA512 || !decodeMatches(expectedSHA512, sha512Sum[:]) {
		app.fail("sha512 mismatch", fmt.Sprintf("Info: sum %s / stream %s", sha512Hex, sha512Stream))
		return
	}

	first := sha512Hex
	second := ""
	if len(sha512Hex) > 64 {
		first = sha512Hex[:64]
		second = sha512Hex[64:]
	}

	app.ok = true
	app.summary = "hash probe ok / md5 sha1 sha256 sha512 working"
	app.md5Line = fmt.Sprintf("MD5:    %s", md5Hex)
	app.sha1Line = fmt.Sprintf("SHA1:   %s", sha1Hex)
	app.sha256Line = fmt.Sprintf("SHA256: %s", sha256Hex)
	app.sha512Line = fmt.Sprintf("SHA512: %s", first)
	app.sha512Line2 = fmt.Sprintf("        %s", second)
	app.streamLine = fmt.Sprintf("Stream: md5 %s / sha1 %s / sha256 %s / sha512 %s", formatBool(md5Stream == expectedMD5), formatBool(sha1Stream == expectedSHA1), formatBool(sha256Stream == expectedSHA256), formatBool(sha512Stream == expectedSHA512))
	app.infoLine = fmt.Sprintf("Info: input %q / Sum and streaming outputs agree", message)
}

func (app *App) fail(detail string, info string) {
	app.ok = false
	app.summary = "hash probe failed / " + detail
	app.infoLine = info
	app.md5Line = ""
	app.sha1Line = ""
	app.sha256Line = ""
	app.sha512Line = ""
	app.sha512Line2 = ""
	app.streamLine = ""
}

func (app *App) summaryColor() kos.Color {
	if app.ok {
		return ui.Lime
	}
	return ui.Red
}

func streamHashHex(hasher hash.Hash, text string) (string, error) {
	split := len(text) / 2
	if split < 1 {
		split = len(text)
	}
	if _, err := hasher.Write([]byte(text[:split])); err != nil {
		return "", err
	}
	if _, err := hasher.Write([]byte(text[split:])); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func decodeMatches(expected string, sum []byte) bool {
	decoded, err := hex.DecodeString(expected)
	if err != nil {
		return false
	}
	return bytes.Equal(decoded, sum)
}

func formatBool(v bool) string {
	if v {
		return "ok"
	}
	return "fail"
}
