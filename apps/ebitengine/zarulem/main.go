package main

import (
	rand "math/rand"

	"kos"
	"surface"
)

const (
	zaRulemTitle   = "Za Rulem"
	zaRulemWindowX = 88
	zaRulemWindowY = 24
)

const (
	worldWidth  = 320
	worldHeight = 350
)

const (
	carWidth  = 26
	carHeight = 50
)

const (
	desiredScale = 2
	updateStepNS = uint64(1_000_000_000 / 60)
	maxFrameNS   = updateStepNS * 5
)

const (
	keyEscape = 27
	keySpace  = ' '
	keyWLow   = 'w'
	keyWHigh  = 'W'
	keyALow   = 'a'
	keyAHigh  = 'A'
	keySLow   = 's'
	keySHigh  = 'S'
	keyDLow   = 'd'
	keyDHigh  = 'D'
)

const (
	scanW      = 0x11
	scanA      = 0x1E
	scanS      = 0x1F
	scanD      = 0x20
	scanSpace  = 0x39
	scanEscape = 0x01
	scanUp     = 72
	scanLeft   = 75
	scanRight  = 77
	scanDown   = 80
)

const (
	enemySpawnTicks = 60
)

const (
	colorWhite kos.Color = surface.White
	colorHud   kos.Color = 0xD8D8D8
	colorHit   kos.Color = 0xFF7070
)

type movementState struct {
	W     bool
	A     bool
	S     bool
	D     bool
	Up    bool
	Left  bool
	Right bool
	Down  bool
}

type car struct {
	sprite *surface.Image
	x      float64
	y      float64
	speed  float64
}

type gameApp struct {
	presenter        surface.Presenter
	canvas           *surface.Buffer
	viewport         surface.Rect
	scale            int
	player           car
	enemies          []car
	scroll           float64
	spawnCounter     int
	moveInput        movementState
	moveImpulseX     float64
	moveImpulseY     float64
	inputHotkeys     []kos.Hotkey
	heldInputEnabled bool
	gameOver         bool
	needsRedraw      bool
}

var (
	backgroundImage   *surface.Image
	backgroundScaled  *surface.Image
	playerImage       *surface.Image
	playerScaledImage *surface.Image
	enemyImage        *surface.Image
	enemyScaledImage  *surface.Image
)

func main() {
	rand.Seed(int64(kos.GetTimeCounterPro()))
	app := newGameApp()
	app.run()
}

func newGameApp() *gameApp {
	presenter := surface.NewPresenterClient(zaRulemWindowX, zaRulemWindowY, worldWidth*desiredScale, worldHeight*desiredScale, zaRulemTitle)
	client := presenter.Client
	app := &gameApp{
		presenter: presenter,
		canvas:    surface.NewBuffer(client.Width, client.Height),
		viewport: surface.Rect{
			Width:  client.Width,
			Height: client.Height,
		},
		scale: desiredScale,
	}
	app.reset()
	return app
}

func (app *gameApp) reset() {
	app.player = car{
		sprite: loadScaledPlayerImage(),
		x:      float64(worldWidth)/2 + 5,
		y:      float64(worldHeight - carHeight),
		speed:  5,
	}
	app.enemies = nil
	app.scroll = 0
	app.spawnCounter = 0
	app.moveInput = movementState{}
	app.moveImpulseX = 0
	app.moveImpulseY = 0
	app.gameOver = false
	app.needsRedraw = true
}

func (app *gameApp) run() {
	app.heldInputEnabled = app.registerMovementHotkeys()
	defer app.unregisterMovementHotkeys()
	app.redraw(true)
	lastTick := kos.GetTimeCounterPro()
	var accumulator uint64
	for {
		event := kos.WaitEventFor(1)
		now := kos.GetTimeCounterPro()
		delta := now - lastTick
		lastTick = now
		if delta > maxFrameNS {
			delta = maxFrameNS
		}
		accumulator += delta

		switch event {
		case kos.EventRedraw:
			app.redraw(true)
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				return
			}
		case kos.EventKey:
			if app.handleKey(kos.ReadKey()) {
				return
			}
		}

		for accumulator >= updateStepNS {
			app.update()
			accumulator -= updateStepNS
		}

		if app.needsRedraw {
			app.redraw(false)
		}
	}
}

func (app *gameApp) handleKey(key kos.KeyEvent) bool {
	if key.Empty {
		return false
	}
	if key.Hotkey {
		app.handleMovementHotkey(key)
		return false
	}
	switch {
	case key.Code == keyEscape || key.ScanCode == scanEscape:
		return true
	case key.Code == keySpace || key.ScanCode == scanSpace:
		if app.gameOver {
			app.reset()
		}
	case !app.heldInputEnabled && (key.Code == keyWLow || key.Code == keyWHigh || key.ScanCode == scanUp):
		app.moveImpulseY = -1
	case !app.heldInputEnabled && (key.Code == keySLow || key.Code == keySHigh || key.ScanCode == scanDown):
		app.moveImpulseY = 1
	case !app.heldInputEnabled && (key.Code == keyALow || key.Code == keyAHigh || key.ScanCode == scanLeft):
		app.moveImpulseX = -1
	case !app.heldInputEnabled && (key.Code == keyDLow || key.Code == keyDHigh || key.ScanCode == scanRight):
		app.moveImpulseX = 1
	}
	return false
}

func (app *gameApp) update() {
	if app.gameOver {
		return
	}

	app.scroll += app.player.speed
	if app.scroll >= worldHeight {
		app.scroll -= worldHeight
	}

	app.movePlayer()
	app.spawnCounter++
	if app.spawnCounter >= enemySpawnTicks {
		app.spawnEnemy()
		app.spawnCounter = 0
	}
	app.updateEnemies()
	app.removeOffscreenEnemies()
	app.checkCollisions()
	app.needsRedraw = true
}

func (app *gameApp) movePlayer() {
	dx, dy := app.moveInput.direction()
	if dx == 0 && dy == 0 {
		dx = app.moveImpulseX
		dy = app.moveImpulseY
	}
	app.moveImpulseX = 0
	app.moveImpulseY = 0

	app.player.x += dx * app.player.speed
	app.player.y += dy * app.player.speed
	app.player.x = clamp(app.player.x, 0, float64(worldWidth-carWidth))
	app.player.y = clamp(app.player.y, 0, float64(worldHeight-carHeight))
}

func (app *gameApp) spawnEnemy() {
	app.enemies = append(app.enemies, car{
		sprite: loadScaledEnemyImage(),
		x:      float64(rand.Intn(worldWidth - carWidth)),
		y:      -float64(carHeight),
		speed:  5,
	})
}

func (app *gameApp) updateEnemies() {
	for index := range app.enemies {
		app.enemies[index].y += app.enemies[index].speed
	}
}

func (app *gameApp) removeOffscreenEnemies() {
	filtered := app.enemies[:0]
	for _, enemy := range app.enemies {
		if enemy.y <= float64(worldHeight) {
			filtered = append(filtered, enemy)
		}
	}
	app.enemies = filtered
}

func (app *gameApp) checkCollisions() {
	for _, enemy := range app.enemies {
		if carsOverlap(app.player, enemy) {
			app.gameOver = true
			app.needsRedraw = true
			return
		}
	}
}

func (app *gameApp) redraw(full bool) {
	app.needsRedraw = false
	app.drawBackground()
	for _, enemy := range app.enemies {
		app.drawCar(enemy)
	}
	app.drawCar(app.player)
	app.drawHUD()
	if full {
		app.presenter.PresentFull(app.canvas)
		return
	}
	app.presenter.PresentClient(app.canvas)
}

func (app *gameApp) drawBackground() {
	app.canvas.Clear(surface.Black)
	background := loadScaledBackgroundImage()
	if background == nil {
		return
	}
	scrollY := int(app.scroll * float64(app.scale))
	if background.Height > 0 {
		scrollY %= background.Height
	}
	app.canvas.DrawImage(0, scrollY, background)
	app.canvas.DrawImage(0, scrollY-background.Height, background)
}

func (app *gameApp) drawCar(value car) {
	if value.sprite == nil {
		return
	}
	app.canvas.DrawImage(int(value.x*float64(app.scale)), int(value.y*float64(app.scale)), value.sprite)
}

func (app *gameApp) drawHUD() {
	hints := []string{
		"<WASD/ARROWS> - move",
		"<SPACE> - restart",
		"<ESC> - exit",
	}
	lineHeight := surface.DefaultFontHeight + 2
	startY := app.viewport.Height - 6 - len(hints)*lineHeight
	for index, hint := range hints {
		app.canvas.DrawText(8, startY+index*lineHeight, colorHud, hint)
	}
	if app.gameOver {
		centerX := app.viewport.Width/2 - 72
		centerY := app.viewport.Height/2 - 8
		app.canvas.DrawText(centerX, centerY, colorHit, "GAME OVER")
		app.canvas.DrawText(centerX-36, centerY+18, colorWhite, "<SPACE> - restart")
	}
}

func (app *gameApp) handleMovementHotkey(key kos.KeyEvent) {
	held := key.HotkeyPressed()
	switch key.HotkeyPressScanCode() {
	case scanW:
		app.moveInput.W = held
	case scanA:
		app.moveInput.A = held
	case scanS:
		app.moveInput.S = held
	case scanD:
		app.moveInput.D = held
	case scanUp:
		app.moveInput.Up = held
	case scanLeft:
		app.moveInput.Left = held
	case scanRight:
		app.moveInput.Right = held
	case scanDown:
		app.moveInput.Down = held
	}
}

func (input movementState) direction() (float64, float64) {
	var dx float64
	var dy float64
	if input.W || input.Up {
		dy--
	}
	if input.S || input.Down {
		dy++
	}
	if input.A || input.Left {
		dx--
	}
	if input.D || input.Right {
		dx++
	}
	return dx, dy
}

func (app *gameApp) scaleX(x float64) int {
	return app.viewport.X + int(x*float64(app.scale))
}

func (app *gameApp) scaleY(y float64) int {
	return app.viewport.Y + int(y*float64(app.scale))
}

func (app *gameApp) scaleRect(x float64, y float64, width int, height int) surface.Rect {
	return surface.Rect{
		X:      app.scaleX(x),
		Y:      app.scaleY(y),
		Width:  width * app.scale,
		Height: height * app.scale,
	}
}

func (app *gameApp) registerMovementHotkeys() bool {
	hotkeys := movementHotkeys()
	registered := make([]kos.Hotkey, 0, len(hotkeys))
	for _, hotkey := range hotkeys {
		if !kos.RegisterHotkey(hotkey) {
			for index := len(registered) - 1; index >= 0; index-- {
				kos.UnregisterHotkey(registered[index])
			}
			return false
		}
		registered = append(registered, hotkey)
	}
	app.inputHotkeys = registered
	return true
}

func (app *gameApp) unregisterMovementHotkeys() {
	for _, hotkey := range app.inputHotkeys {
		kos.UnregisterHotkey(hotkey)
	}
	app.inputHotkeys = nil
}

func movementHotkeys() []kos.Hotkey {
	keys := []byte{
		scanW,
		scanA,
		scanS,
		scanD,
		scanUp,
		scanLeft,
		scanRight,
		scanDown,
	}
	hotkeys := make([]kos.Hotkey, 0, len(keys)*2)
	for _, scanCode := range keys {
		hotkey := kos.Hotkey{ScanCode: scanCode}
		hotkeys = append(hotkeys, hotkey, hotkey.Release())
	}
	return hotkeys
}

func loadBackgroundImage() *surface.Image {
	if backgroundImage != nil {
		return backgroundImage
	}
	backgroundImage = mustImageCandidate("assets/road.png", "apps/ebitengine/zarulem/assets/road.png")
	return backgroundImage
}

func loadScaledBackgroundImage() *surface.Image {
	if backgroundScaled != nil {
		return backgroundScaled
	}
	backgroundScaled = surface.ScaleImageNearest(loadBackgroundImage(), worldWidth*desiredScale, worldHeight*desiredScale)
	return backgroundScaled
}

func loadPlayerImage() *surface.Image {
	if playerImage != nil {
		return playerImage
	}
	playerImage = mustImageCandidate("assets/car.png", "apps/ebitengine/zarulem/assets/car.png")
	return playerImage
}

func loadScaledPlayerImage() *surface.Image {
	if playerScaledImage != nil {
		return playerScaledImage
	}
	playerScaledImage = surface.ScaleImageNearest(loadPlayerImage(), carWidth*desiredScale, carHeight*desiredScale)
	return playerScaledImage
}

func loadEnemyImage() *surface.Image {
	if enemyImage != nil {
		return enemyImage
	}
	enemyImage = mustImageCandidate("assets/enemy.png", "apps/ebitengine/zarulem/assets/enemy.png")
	return enemyImage
}

func loadScaledEnemyImage() *surface.Image {
	if enemyScaledImage != nil {
		return enemyScaledImage
	}
	enemyScaledImage = surface.ScaleImageNearest(loadEnemyImage(), carWidth*desiredScale, carHeight*desiredScale)
	return enemyScaledImage
}

func mustImageCandidate(paths ...string) *surface.Image {
	for _, path := range paths {
		if image := surface.GetImage(path); image != nil {
			return image
		}
	}
	panic("zarulem: missing image asset")
}

func carsOverlap(a car, b car) bool {
	return a.x < b.x+carWidth &&
		a.x+carWidth > b.x &&
		a.y < b.y+carHeight &&
		a.y+carHeight > b.y
}

func clamp(value float64, low float64, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
