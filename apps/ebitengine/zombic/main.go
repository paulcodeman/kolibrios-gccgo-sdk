package main

import (
	"math"
	rand "math/rand"

	"kos"
	"surface"
)

const (
	zombicTitle   = "Zombic"
	zombicWindowX = 72
	zombicWindowY = 52
)

const (
	worldWidth  = 320
	worldHeight = 180
)

const (
	playerBoundsWidth  = 14
	playerBoundsHeight = 16
)

const (
	weaponDrawOffsetX = 1
	weaponDrawOffsetY = 7
)

const (
	desiredScale = 2
	updateStepNS = uint64(1_000_000_000 / 60)
	maxFrameNS   = updateStepNS * 5
)

const (
	keyEscape = 27
	keyWLow   = 'w'
	keyWHigh  = 'W'
	keyALow   = 'a'
	keyAHigh  = 'A'
	keySLow   = 's'
	keySHigh  = 'S'
	keyDLow   = 'd'
	keyDHigh  = 'D'
	keyFLow   = 'f'
	keyFHigh  = 'F'
	keyRLow   = 'r'
	keyRHigh  = 'R'
)

const (
	scanW      = 0x11
	scanR      = 0x13
	scanA      = 0x1E
	scanS      = 0x1F
	scanD      = 0x20
	scanF      = 0x21
	scanEscape = 1
	scanUp     = 72
	scanLeft   = 75
	scanRight  = 77
	scanDown   = 80
)

const (
	playerMaxHitPoints         = 5
	playerContactCooldownTicks = 30
	playerHitFlashTicks        = 10
)

const (
	colorBlack     kos.Color = surface.Black
	colorWhite     kos.Color = surface.White
	colorSky       kos.Color = 0x82CEEB
	colorHud       kos.Color = 0xC8D2D8
	colorHudDim    kos.Color = 0x8DA2B2
	colorPlayerAim kos.Color = 0xFFCF5C
	colorBullet    kos.Color = 0x780000
	colorHitFlash  kos.Color = 0xF25F5C
	colorViewport  kos.Color = 0x4F7CFF
)

const (
	stateAttacking enemyState = iota
	stateHit
	stateDead
)

type enemyState int

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

type position struct {
	X float64
	Y float64
}

type direction struct {
	X float64
	Y float64
}

type collider struct {
	Position *position
	Width    float64
	Height   float64
}

type weapon struct {
	position             *position
	sprite               *surface.Image
	aimDirection         direction
	aimValid             bool
	shootingSpeed        int
	shootingSpeedCounter int
}

type missile struct {
	position   position
	direction  direction
	speed      float64
	collider   collider
	lifetime   int
	ticksLived int
	removable  bool
}

type path struct {
	Points []position
}

type pathFollowLoop struct {
	path                    path
	position                position
	velocity                float64
	currentTargetPointIndex int
}

type spawner struct {
	pathFollowLoop  pathFollowLoop
	secondsInterval float64
	timeCounter     int
}

type zombieFrameSet struct {
	walkRight []*surface.Image
	walkLeft  []*surface.Image
	walkUp    []*surface.Image
	walkDown  []*surface.Image
	hitRight  []*surface.Image
	hitLeft   []*surface.Image
	hitUp     []*surface.Image
	hitDown   []*surface.Image
	idle      *surface.Image
}

type enemySpec struct {
	name      string
	speedMin  float64
	speedMax  float64
	hitPoints int
	frames    *zombieFrameSet
}

type player struct {
	animated       *surface.AnimatedSprite
	position       position
	weaponPosition position
	speed          float64
	collider       collider
	weapon         weapon
	lastMove       direction
	hitPoints      int
	maxHitPoints   int
	hitCooldown    int
	hitFlashTicks  int
}

type enemy struct {
	spec        *enemySpec
	animated    *surface.AnimatedSprite
	position    position
	speed       float64
	baseSpeed   float64
	target      *position
	collider    collider
	state       enemyState
	hitCooldown int
	hitPoints   int
}

type gameApp struct {
	presenter        surface.Presenter
	canvas           *surface.Buffer
	viewport         surface.Rect
	scale            int
	player           *player
	enemies          []*enemy
	missiles         []*missile
	spawner          *spawner
	kills            int
	ticks            int
	needsRedraw      bool
	mouseX           int
	mouseY           int
	mouseAiming      bool
	moveInput        movementState
	moveImpulseX     float64
	moveImpulseY     float64
	inputHotkeys     []kos.Hotkey
	heldInputEnabled bool
	autoShoot        bool
	gameOver         bool
}

var (
	playerFrames *zombieFrameSet
	weaponSprite *surface.Image
	bigZombie    *enemySpec
	kidZombie    *enemySpec
	skinnyZombie *enemySpec
)

func main() {
	rand.Seed(int64(kos.GetTimeCounterPro()))
	app := newGameApp()
	app.run()
}

func newGameApp() *gameApp {
	presenter := surface.NewPresenterClient(zombicWindowX, zombicWindowY, worldWidth*desiredScale, worldHeight*desiredScale, zombicTitle)
	client := presenter.Client
	viewport := surface.Rect{
		Width:  client.Width,
		Height: client.Height,
	}
	app := &gameApp{
		presenter: presenter,
		canvas:    surface.NewBuffer(client.Width, client.Height),
		viewport:  viewport,
		scale:     desiredScale,
	}
	app.reset()
	return app
}

func (app *gameApp) reset() {
	app.player = newPlayer()
	app.enemies = nil
	app.missiles = nil
	app.spawner = newSpawner()
	app.kills = 0
	app.ticks = 0
	app.needsRedraw = true
	app.moveInput = movementState{}
	app.moveImpulseX = 0
	app.moveImpulseY = 0
	app.autoShoot = true
	app.gameOver = false
}

func newPlayer() *player {
	sprite := surface.NewAnimatedSprite()
	frames := loadPlayerFrames()
	registerDirectionalAnimations(sprite, frames, nil)
	p := &player{
		animated:     sprite,
		position:     position{X: 150, Y: 80},
		speed:        1.0,
		lastMove:     direction{X: 1, Y: 0},
		hitPoints:    playerMaxHitPoints,
		maxHitPoints: playerMaxHitPoints,
	}
	p.weapon = weapon{
		position:             &p.weaponPosition,
		sprite:               loadWeaponSprite(),
		aimDirection:         direction{X: 1, Y: 0},
		aimValid:             true,
		shootingSpeed:        15,
		shootingSpeedCounter: 15,
	}
	p.updateCollider()
	p.updateWeaponPosition()
	return p
}

func newSpawner() *spawner {
	return &spawner{
		pathFollowLoop: pathFollowLoop{
			path: path{
				Points: []position{
					{X: -10, Y: -10},
					{X: worldWidth + 10, Y: -10},
					{X: worldWidth + 10, Y: worldHeight + 10},
					{X: -10, Y: worldHeight + 10},
				},
			},
			position: position{X: -10, Y: -10},
			velocity: 1.0,
		},
		secondsInterval: 0.5,
	}
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
	case key.Code == keyFLow || key.Code == keyFHigh || key.ScanCode == scanF:
		app.autoShoot = !app.autoShoot
	case key.Code == keyRLow || key.Code == keyRHigh || key.ScanCode == scanR:
		app.reset()
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
	app.ticks++
	moveX, moveY := app.moveInput.direction()
	if moveX == 0 && moveY == 0 {
		moveX = app.moveImpulseX
		moveY = app.moveImpulseY
	}
	app.moveImpulseX = 0
	app.moveImpulseY = 0
	if app.gameOver {
		app.player.update(0, 0)
		app.updateMouseCombat()
		app.needsRedraw = true
		return
	}
	app.player.update(moveX, moveY)
	app.releaseBoundaryInput(moveX, moveY)
	app.updateMouseCombat()
	app.spawner.update(app.player, &app.enemies)

	for _, bullet := range app.missiles {
		bullet.update()
	}

	enemyColliders := make([]*collider, 0, len(app.enemies))
	for _, current := range app.enemies {
		enemyColliders = append(enemyColliders, &current.collider)
	}
	for _, current := range app.enemies {
		current.update(enemyColliders)
	}

	app.handleMissileHits()
	app.handleMissileExpiry()
	app.handlePlayerContactDamage()
	app.needsRedraw = true
}

func (app *gameApp) redraw(full bool) {
	app.needsRedraw = false
	app.drawBackdrop()
	app.drawWorld()
	app.drawHUD()
	if full {
		app.presenter.PresentFull(app.canvas)
		return
	}
	app.presenter.PresentClient(app.canvas)
}

func (app *gameApp) drawBackdrop() {
	app.canvas.Clear(colorSky)
}

func (app *gameApp) drawWorld() {
	app.player.draw(app)
	for _, bullet := range app.missiles {
		bullet.draw(app)
	}
	for _, current := range app.enemies {
		current.draw(app)
	}
	app.drawCrosshair()
}

func (app *gameApp) drawHUD() {
	app.canvas.DrawText(8, 8, colorWhite, "HP "+itoa(app.player.hitPoints))
	hints := []string{
		"<WASD/ARROWS> - move",
		"<MOUSE> - aim",
	}
	if app.autoShoot {
		hints = append(hints, "<F> - manual fire")
	} else {
		hints = append(hints, "<LMB> - fire")
		hints = append(hints, "<F> - auto fire")
	}
	hints = append(hints, "<R> - reset", "<ESC> - exit")
	lineHeight := surface.DefaultFontHeight + 2
	startY := app.viewport.Height - 6 - len(hints)*lineHeight
	for index, hint := range hints {
		app.canvas.DrawText(8, startY+index*lineHeight, colorHudDim, hint)
	}
	if app.gameOver {
		centerX := app.viewport.Width/2 - 44
		centerY := app.viewport.Height/2 - 8
		app.canvas.DrawText(centerX, centerY, colorHitFlash, "GAME OVER")
		app.canvas.DrawText(centerX-18, centerY+16, colorWhite, "<R> - restart")
	}
}

func (app *gameApp) updateMouseCombat() {
	pos := kos.MouseWindowPosition()
	app.mouseX = pos.X - app.presenter.Client.X
	app.mouseY = pos.Y - app.presenter.Client.Y
	app.mouseAiming = app.viewport.Contains(app.mouseX, app.mouseY)
	if app.gameOver {
		return
	}

	if app.autoShoot {
		target := app.findNearestEnemyTarget()
		if target == nil {
			if app.mouseAiming {
				if mouseTarget := app.mouseWorldTarget(); mouseTarget != nil {
					app.player.weapon.setAimDirection(normalFromPositions(app.player.center(), *mouseTarget))
				}
			}
			return
		}
		app.player.weapon.setAimDirection(normalFromPositions(app.player.center(), *target))
		if !app.player.weapon.canShoot() {
			return
		}
		bullet := app.player.weapon.shootAt(target)
		if bullet != nil {
			app.missiles = append(app.missiles, bullet)
		}
		return
	}

	if !app.mouseAiming {
		return
	}

	target := app.mouseWorldTarget()
	if target == nil {
		return
	}
	playerCenter := app.player.center()
	app.player.weapon.setAimDirection(normalFromPositions(playerCenter, *target))

	buttons := kos.MouseHeldButtons()
	if !buttons.LeftHeld || !app.player.weapon.canShoot() {
		return
	}
	bullet := app.player.weapon.shootAt(target)
	if bullet != nil {
		app.missiles = append(app.missiles, bullet)
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

func (app *gameApp) handleMissileHits() {
	for _, bullet := range app.missiles {
		if bullet.removable {
			continue
		}
		for _, current := range app.enemies {
			if current.state == stateDead {
				continue
			}
			if current.collider.collidesWith(&bullet.collider) {
				current.markAsHit()
				bullet.removable = true
				break
			}
		}
	}

	filteredMissiles := app.missiles[:0]
	for _, bullet := range app.missiles {
		if !bullet.removable {
			filteredMissiles = append(filteredMissiles, bullet)
		}
	}
	app.missiles = filteredMissiles

	filteredEnemies := app.enemies[:0]
	for _, current := range app.enemies {
		if current.state == stateDead {
			app.kills++
			continue
		}
		filteredEnemies = append(filteredEnemies, current)
	}
	app.enemies = filteredEnemies
}

func (app *gameApp) handleMissileExpiry() {
	filtered := app.missiles[:0]
	for _, bullet := range app.missiles {
		if bullet.ticksLived > bullet.lifetime {
			continue
		}
		filtered = append(filtered, bullet)
	}
	app.missiles = filtered
}

func (app *gameApp) handlePlayerContactDamage() {
	if app.gameOver || !app.player.canTakeDamage() {
		return
	}
	for _, current := range app.enemies {
		if current == nil || current.state == stateDead {
			continue
		}
		if !current.collider.collidesWith(&app.player.collider) {
			continue
		}
		if app.player.takeHit() {
			app.gameOver = true
		}
		return
	}
}

func (app *gameApp) findNearestEnemyTarget() *position {
	if app == nil || app.player == nil {
		return nil
	}
	playerCenter := app.player.center()
	var nearest *position
	bestDistance := math.MaxFloat64
	for _, current := range app.enemies {
		if current == nil || current.state == stateDead {
			continue
		}
		target := current.center()
		distance := playerCenter.distanceTo(&target)
		if distance >= bestDistance {
			continue
		}
		bestDistance = distance
		targetCopy := target
		nearest = &targetCopy
	}
	return nearest
}

func (app *gameApp) releaseBoundaryInput(moveX float64, moveY float64) {
	if app == nil || app.player == nil {
		return
	}
	maxX := float64(worldWidth - playerBoundsWidth)
	maxY := float64(worldHeight - playerBoundsHeight)
	if moveX < 0 && app.player.position.X <= 0 {
		app.moveInput.A = false
		app.moveInput.Left = false
	}
	if moveX > 0 && app.player.position.X >= maxX {
		app.moveInput.D = false
		app.moveInput.Right = false
	}
	if moveY < 0 && app.player.position.Y <= 0 {
		app.moveInput.W = false
		app.moveInput.Up = false
	}
	if moveY > 0 && app.player.position.Y >= maxY {
		app.moveInput.S = false
		app.moveInput.Down = false
	}
}

func (player *player) update(dx float64, dy float64) {
	if player.hitCooldown > 0 {
		player.hitCooldown--
	}
	if player.hitFlashTicks > 0 {
		player.hitFlashTicks--
	}
	player.animated.Play("idle")

	if dx != 0 || dy != 0 {
		nx, ny := normal(dx, dy)
		player.position.X += nx * player.speed
		player.position.Y += ny * player.speed
		player.lastMove = direction{X: nx, Y: ny}
		switch {
		case math.Abs(nx) >= math.Abs(ny) && nx > 0:
			player.animated.Play("walk_right")
		case math.Abs(nx) >= math.Abs(ny) && nx < 0:
			player.animated.Play("walk_left")
		case ny > 0:
			player.animated.Play("walk_down")
		case ny < 0:
			player.animated.Play("walk_up")
		}
		player.weapon.setAimDirection(direction{X: nx, Y: ny})
	}

	current := player.animated.Current()
	if current != nil {
		maxX := float64(worldWidth - playerBoundsWidth)
		maxY := float64(worldHeight - playerBoundsHeight)
		player.position.X = clamp(player.position.X, 0, maxX)
		player.position.Y = clamp(player.position.Y, 0, maxY)
	}

	player.updateCollider()
	player.updateWeaponPosition()
	player.weapon.update()
}

func (player *player) draw(app *gameApp) {
	image := player.animated.Current()
	if image != nil {
		drawX, drawY := player.spriteDrawPosition(image)
		rect := app.scaleRect(drawX, drawY, image.Width, image.Height)
		app.canvas.DrawImageRect(rect, image)
		if player.hitFlashTicks > 0 {
			app.canvas.DrawLine(rect.X, rect.Y, rect.X+rect.Width-1, rect.Y+rect.Height-1, colorHitFlash)
			app.canvas.DrawLine(rect.X+rect.Width-1, rect.Y, rect.X, rect.Y+rect.Height-1, colorHitFlash)
		}
	}
	player.weapon.draw(app)
}

func (player *player) center() position {
	return position{
		X: player.position.X + playerBoundsWidth/2,
		Y: player.position.Y + playerBoundsHeight/2,
	}
}

func (player *player) updateCollider() {
	player.collider = collider{
		Position: &player.position,
		Width:    playerBoundsWidth,
		Height:   playerBoundsHeight,
	}
}

func (player *player) updateWeaponPosition() {
	image := player.animated.Current()
	if image == nil {
		player.weaponPosition = player.position
		return
	}
	drawX, drawY := player.spriteDrawPosition(image)
	player.weaponPosition = position{X: drawX, Y: drawY}
}

func (player *player) spriteDrawPosition(image *surface.Image) (float64, float64) {
	if image == nil {
		return player.position.X, player.position.Y
	}
	drawX := player.position.X + float64(playerBoundsWidth-image.Width)/2
	drawY := player.position.Y + float64(playerBoundsHeight-image.Height)/2
	return drawX, drawY
}

func (player *player) canTakeDamage() bool {
	return player.hitPoints > 0 && player.hitCooldown == 0
}

func (player *player) takeHit() bool {
	if !player.canTakeDamage() {
		return false
	}
	player.hitPoints--
	player.hitCooldown = playerContactCooldownTicks
	player.hitFlashTicks = playerHitFlashTicks
	return player.hitPoints <= 0
}

func (weapon *weapon) canShoot() bool {
	return weapon.shootingSpeedCounter >= weapon.shootingSpeed
}

func (weapon *weapon) update() {
	weapon.shootingSpeedCounter++
}

func (weapon *weapon) setAimDirection(value direction) {
	if value.X == 0 && value.Y == 0 {
		return
	}
	weapon.aimDirection = value
	weapon.aimValid = true
}

func (weapon *weapon) barrelPosition() position {
	if weapon.position == nil {
		return position{}
	}
	origin := weapon.drawOrigin()
	if !weapon.aimValid || weapon.sprite == nil {
		return origin
	}
	localX := float64(weapon.sprite.Width)
	localY := float64(weapon.sprite.Height) / 2
	if weapon.shouldMirror() {
		localY = -localY
	}
	angle := weapon.angle()
	cosAngle := math.Cos(angle)
	sinAngle := math.Sin(angle)
	return position{
		X: origin.X + localX*cosAngle - localY*sinAngle,
		Y: origin.Y + localX*sinAngle + localY*cosAngle,
	}
}

func (weapon *weapon) drawOrigin() position {
	if weapon.position == nil {
		return position{}
	}
	return position{
		X: weapon.position.X + weaponDrawOffsetX,
		Y: weapon.position.Y + weaponDrawOffsetY,
	}
}

func (weapon *weapon) angle() float64 {
	return math.Atan2(weapon.aimDirection.Y, weapon.aimDirection.X)
}

func (weapon *weapon) shouldMirror() bool {
	return weapon.aimValid && weapon.aimDirection.X < 0
}

func (weapon *weapon) shootAt(target *position) *missile {
	if target == nil || !weapon.canShoot() {
		return nil
	}
	weapon.shootingSpeedCounter = 0
	barrel := weapon.barrelPosition()
	dir := normalFromPositions(barrel, *target)
	weapon.setAimDirection(dir)
	return newMissile(barrel, dir)
}

func (weapon *weapon) draw(app *gameApp) {
	if !weapon.aimValid || weapon.sprite == nil {
		return
	}
	origin := weapon.drawOrigin()
	scaleY := float64(app.scale)
	if weapon.shouldMirror() {
		scaleY = -scaleY
	}
	app.canvas.DrawImageRotatedScaled(
		float64(app.scaleX(origin.X)),
		float64(app.scaleY(origin.Y)),
		weapon.sprite,
		weapon.angle(),
		float64(app.scale),
		scaleY,
		0,
		0,
	)
}

func (app *gameApp) drawCrosshair() {
	if !app.mouseAiming || app.autoShoot {
		return
	}
	x := app.mouseX
	y := app.mouseY
	app.canvas.DrawLine(x-6, y, x-2, y, colorWhite)
	app.canvas.DrawLine(x+2, y, x+6, y, colorWhite)
	app.canvas.DrawLine(x, y-6, x, y-2, colorWhite)
	app.canvas.DrawLine(x, y+2, x, y+6, colorWhite)
}

func newMissile(origin position, dir direction) *missile {
	bullet := &missile{
		position:  origin,
		direction: dir,
		speed:     5.0,
		lifetime:  5 * 60,
	}
	bullet.updateCollider()
	return bullet
}

func (bullet *missile) update() {
	bullet.position.X += bullet.direction.X * bullet.speed
	bullet.position.Y += bullet.direction.Y * bullet.speed
	bullet.ticksLived++
	bullet.updateCollider()
	if bullet.position.X < -8 || bullet.position.Y < -8 || bullet.position.X > worldWidth+8 || bullet.position.Y > worldHeight+8 {
		bullet.removable = true
	}
}

func (bullet *missile) updateCollider() {
	bullet.collider = collider{
		Position: &bullet.position,
		Width:    2,
		Height:   2,
	}
}

func (bullet *missile) draw(app *gameApp) {
	rect := app.scaleRect(bullet.position.X, bullet.position.Y, 2, 2)
	app.canvas.FillRect(rect.X, rect.Y, rect.Width, rect.Height, colorBullet)
}

func (spawner *spawner) update(player *player, enemies *[]*enemy) {
	spawner.pathFollowLoop.update()
	spawner.timeCounter++
	limit := int(spawner.secondsInterval * 60)
	if spawner.timeCounter <= limit {
		return
	}
	*enemies = append(*enemies, spawner.createEnemy(&player.position))
	spawner.timeCounter = 0
}

func (spawner *spawner) createEnemy(target *position) *enemy {
	spawn := spawner.pathFollowLoop.position
	chance := rand.Float64()
	switch {
	case chance < 0.4:
		return newEnemy(loadEnemySpec("SkinnyZombie", 0.25, 0.75, 2), spawn, target)
	case chance < 0.8:
		return newEnemy(loadEnemySpec("KidZombie", 0.5, 1.0, 1), spawn, target)
	default:
		return newEnemy(loadEnemySpec("BigZombie", 0.1, 0.4, 3), spawn, target)
	}
}

func (loop *pathFollowLoop) update() {
	if len(loop.path.Points) == 0 {
		return
	}
	target := loop.path.Points[loop.currentTargetPointIndex]
	nx, ny := normal(target.X-loop.position.X, target.Y-loop.position.Y)
	loop.position.X += nx * loop.velocity
	loop.position.Y += ny * loop.velocity
	if loop.position.isNear(target) {
		loop.currentTargetPointIndex = (loop.currentTargetPointIndex + 1) % len(loop.path.Points)
	}
}

func newEnemy(spec *enemySpec, spawn position, target *position) *enemy {
	if spec == nil {
		return nil
	}
	sprite := surface.NewAnimatedSprite()
	registerDirectionalAnimations(sprite, spec.frames, spec.frames)
	baseSpeed := spec.speedMin + rand.Float64()*(spec.speedMax-spec.speedMin)
	current := &enemy{
		spec:      spec,
		animated:  sprite,
		position:  spawn,
		speed:     baseSpeed,
		baseSpeed: baseSpeed,
		target:    target,
		state:     stateAttacking,
		hitPoints: spec.hitPoints,
	}
	current.updateCollider()
	return current
}

func (enemy *enemy) center() position {
	return position{
		X: enemy.position.X + enemy.collider.Width/2,
		Y: enemy.position.Y + enemy.collider.Height/2,
	}
}

func (enemy *enemy) update(colliders []*collider) {
	if enemy == nil || enemy.state == stateDead || enemy.target == nil {
		return
	}
	enemy.animated.Play("idle")
	nx, ny := normal(enemy.target.X-enemy.position.X, enemy.target.Y-enemy.position.Y)
	enemy.moveWithCollisions(nx, ny, colliders)

	switch enemy.state {
	case stateAttacking:
		enemy.playDirectional("walk", nx, ny)
	case stateHit:
		enemy.playDirectional("hit", nx, ny)
		enemy.hitCooldown--
		if enemy.hitCooldown <= 0 {
			enemy.state = stateAttacking
			enemy.speed = enemy.baseSpeed
		}
	}

	enemy.updateCollider()
}

func (enemy *enemy) playDirectional(prefix string, nx float64, ny float64) {
	switch {
	case math.Abs(nx) >= math.Abs(ny) && nx > 0:
		enemy.animated.Play(prefix + "_right")
	case math.Abs(nx) >= math.Abs(ny) && nx < 0:
		enemy.animated.Play(prefix + "_left")
	case ny > 0:
		enemy.animated.Play(prefix + "_down")
	case ny < 0:
		enemy.animated.Play(prefix + "_up")
	default:
		enemy.animated.Play("idle")
	}
}

func (enemy *enemy) moveWithCollisions(nx float64, ny float64, colliders []*collider) {
	enemy.moveYWithCollisions(ny, colliders)
	enemy.moveXWithCollisions(nx, colliders)
}

func (enemy *enemy) moveXWithCollisions(nx float64, colliders []*collider) {
	nextX := enemy.position.X + nx*enemy.speed
	moveLeft := nextX < enemy.position.X
	moveRight := nextX > enemy.position.X
	next := collider{
		Position: &position{X: nextX, Y: enemy.position.Y},
		Width:    enemy.collider.Width,
		Height:   enemy.collider.Height,
	}
	for _, other := range colliders {
		if other == nil || other == &enemy.collider {
			continue
		}
		if next.collidesWith(other) && next.collidesFromRightWith(other) && moveRight {
			return
		}
		if next.collidesWith(other) && next.collidesFromLeftWith(other) && moveLeft {
			return
		}
	}
	enemy.position.X = nextX
}

func (enemy *enemy) moveYWithCollisions(ny float64, colliders []*collider) {
	nextY := enemy.position.Y + ny*enemy.speed
	moveUp := nextY < enemy.position.Y
	moveDown := nextY > enemy.position.Y
	next := collider{
		Position: &position{X: enemy.position.X, Y: nextY},
		Width:    enemy.collider.Width,
		Height:   enemy.collider.Height,
	}
	for _, other := range colliders {
		if other == nil || other == &enemy.collider {
			continue
		}
		if next.collidesWith(other) && next.collidesFromTopWith(other) && moveDown {
			return
		}
		if next.collidesWith(other) && next.collidesFromDownWith(other) && moveUp {
			return
		}
	}
	enemy.position.Y = nextY
}

func (enemy *enemy) markAsHit() {
	enemy.hitPoints--
	if enemy.hitPoints <= 0 {
		enemy.state = stateDead
		return
	}
	enemy.state = stateHit
	enemy.hitCooldown = 18
	enemy.speed = enemy.baseSpeed * 0.4
}

func (enemy *enemy) draw(app *gameApp) {
	if enemy == nil || enemy.state == stateDead {
		return
	}
	image := enemy.animated.Current()
	if image == nil {
		return
	}
	rect := app.scaleRect(enemy.position.X, enemy.position.Y, image.Width, image.Height)
	app.canvas.DrawImageRect(rect, image)
	if enemy.state == stateHit {
		app.canvas.DrawLine(rect.X, rect.Y, rect.X+rect.Width-1, rect.Y+rect.Height-1, colorHitFlash)
		app.canvas.DrawLine(rect.X+rect.Width-1, rect.Y, rect.X, rect.Y+rect.Height-1, colorHitFlash)
	}
}

func (enemy *enemy) updateCollider() {
	image := enemy.animated.Current()
	if image == nil {
		enemy.collider = collider{Position: &enemy.position}
		return
	}
	offsetX := 2.0
	offsetY := 2.0
	enemy.collider = collider{
		Position: &position{X: enemy.position.X + offsetX, Y: enemy.position.Y + offsetY},
		Width:    float64(image.Width) - 2*offsetX,
		Height:   float64(image.Height) - 2*offsetY,
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
	return app.viewport.X + int(math.Round(x*float64(app.scale)))
}

func (app *gameApp) scaleY(y float64) int {
	return app.viewport.Y + int(math.Round(y*float64(app.scale)))
}

func (app *gameApp) scaleRect(x float64, y float64, width int, height int) surface.Rect {
	return surface.Rect{
		X:      app.scaleX(x),
		Y:      app.scaleY(y),
		Width:  width * app.scale,
		Height: height * app.scale,
	}
}

func (app *gameApp) mouseWorldTarget() *position {
	if !app.mouseAiming {
		return nil
	}
	return &position{
		X: float64(app.mouseX-app.viewport.X) / float64(app.scale),
		Y: float64(app.mouseY-app.viewport.Y) / float64(app.scale),
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

func loadPlayerFrames() *zombieFrameSet {
	if playerFrames != nil {
		return playerFrames
	}
	right1 := mustSprite("Player/right1.png")
	right2 := mustSprite("Player/right2.png")
	right3 := mustSprite("Player/right3.png")
	up1 := mustSprite("Player/up1.png")
	up2 := mustSprite("Player/up2.png")
	up3 := mustSprite("Player/up3.png")
	down1 := mustSprite("Player/down1.png")
	down2 := mustSprite("Player/down2.png")
	down3 := mustSprite("Player/down3.png")
	playerFrames = &zombieFrameSet{
		walkRight: []*surface.Image{right1, right2, right1, right3},
		walkLeft:  []*surface.Image{surface.MirrorImage(right1), surface.MirrorImage(right2), surface.MirrorImage(right1), surface.MirrorImage(right3)},
		walkUp:    []*surface.Image{up1, up2, up1, up3},
		walkDown:  []*surface.Image{down1, down2, down1, down3},
		idle:      right1,
	}
	return playerFrames
}

func loadWeaponSprite() *surface.Image {
	if weaponSprite != nil {
		return weaponSprite
	}
	weaponSprite = mustSprite("Pickable/shotgun.png")
	return weaponSprite
}

func loadEnemySpec(name string, speedMin float64, speedMax float64, hitPoints int) *enemySpec {
	cache := enemySpecSlot(name)
	if cache == nil {
		return nil
	}
	if *cache != nil {
		return *cache
	}
	frames := loadEnemyFrames(name)
	frames.walkRight = mirrorFrames(frames.walkLeft)
	frames.hitRight = mirrorFrames(frames.hitLeft)
	frames.idle = frames.walkDown[0]
	*cache = &enemySpec{
		name:      name,
		speedMin:  speedMin,
		speedMax:  speedMax,
		hitPoints: hitPoints,
		frames:    frames,
	}
	return *cache
}

func enemySpecSlot(name string) **enemySpec {
	switch name {
	case "BigZombie":
		return &bigZombie
	case "KidZombie":
		return &kidZombie
	case "SkinnyZombie":
		return &skinnyZombie
	default:
		return nil
	}
}

func loadEnemyFrames(name string) *zombieFrameSet {
	return &zombieFrameSet{
		walkLeft: loadWalkFrames(name, "left"),
		walkUp:   loadWalkFrames(name, "up"),
		walkDown: loadWalkFrames(name, "down"),
		hitLeft:  loadHitFrames(name, "left"),
		hitUp:    loadHitFrames(name, "up"),
		hitDown:  loadHitFrames(name, "down"),
	}
}

func loadWalkFrames(name string, direction string) []*surface.Image {
	frame1 := mustSprite(name + "/" + direction + "1.png")
	frame2 := mustSprite(name + "/" + direction + "2.png")
	frame3 := mustSprite(name + "/" + direction + "3.png")
	return []*surface.Image{frame1, frame2, frame1, frame3}
}

func loadHitFrames(name string, direction string) []*surface.Image {
	damaged := name + "Damaged/" + direction
	return []*surface.Image{
		mustSprite(damaged + "1.png"),
		mustSprite(damaged + "2.png"),
		mustSprite(damaged + "3.png"),
	}
}

func registerDirectionalAnimations(sprite *surface.AnimatedSprite, walk *zombieFrameSet, hit *zombieFrameSet) {
	if sprite == nil || walk == nil {
		return
	}
	sprite.RegisterAnimation("walk_right", walk.walkRight, 7)
	sprite.RegisterAnimation("walk_left", walk.walkLeft, 7)
	sprite.RegisterAnimation("walk_up", walk.walkUp, 7)
	sprite.RegisterAnimation("walk_down", walk.walkDown, 7)
	sprite.RegisterAnimation("idle", []*surface.Image{walk.idle}, 20)
	if hit != nil && len(hit.hitRight) != 0 {
		sprite.RegisterAnimation("hit_right", hit.hitRight, 3)
		sprite.RegisterAnimation("hit_left", hit.hitLeft, 3)
		sprite.RegisterAnimation("hit_up", hit.hitUp, 3)
		sprite.RegisterAnimation("hit_down", hit.hitDown, 3)
	}
}

func mustSprite(relative string) *surface.Image {
	return mustImageCandidate("assets/sprites/"+relative, "apps/ebitengine/zombic/assets/sprites/"+relative)
}

func mustImageCandidate(paths ...string) *surface.Image {
	for _, path := range paths {
		if image := surface.GetImage(path); image != nil {
			return image
		}
	}
	panic("zombic: missing image asset")
}

func mirrorFrames(frames []*surface.Image) []*surface.Image {
	result := make([]*surface.Image, len(frames))
	for index, frame := range frames {
		result[index] = surface.MirrorImage(frame)
	}
	return result
}

func normal(x float64, y float64) (float64, float64) {
	length := math.Hypot(x, y)
	if length == 0 {
		return 0, 0
	}
	return x / length, y / length
}

func normalFromPositions(from position, to position) direction {
	nx, ny := normal(to.X-from.X, to.Y-from.Y)
	return direction{X: nx, Y: ny}
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

func (left position) distanceTo(right *position) float64 {
	if right == nil {
		return 0
	}
	dx := right.X - left.X
	dy := right.Y - left.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (value position) isNear(other position) bool {
	return math.Round(value.X-other.X) == 0 && math.Round(value.Y-other.Y) == 0
}

func (left *collider) collidesFromRightWith(right *collider) bool {
	if left == nil || right == nil || left.Position == nil || right.Position == nil {
		return false
	}
	leftX := left.Position.X
	rightX := left.Position.X + left.Width
	otherLeft := right.Position.X
	return leftX < otherLeft && rightX >= otherLeft
}

func (left *collider) collidesFromLeftWith(right *collider) bool {
	if left == nil || right == nil {
		return false
	}
	return right.collidesFromRightWith(left)
}

func (left *collider) collidesFromDownWith(right *collider) bool {
	if left == nil || right == nil || left.Position == nil || right.Position == nil {
		return false
	}
	top := left.Position.Y
	bottom := left.Position.Y + left.Height
	otherBottom := right.Position.Y + right.Height
	return bottom > otherBottom && top <= otherBottom
}

func (left *collider) collidesFromTopWith(right *collider) bool {
	if left == nil || right == nil {
		return false
	}
	return right.collidesFromDownWith(left)
}

func (left *collider) collidesWith(right *collider) bool {
	if left == nil || right == nil || left.Position == nil || right.Position == nil {
		return false
	}
	return left.overlapsX(right) && left.overlapsY(right)
}

func (left *collider) overlapsX(right *collider) bool {
	leftMin := left.Position.X
	leftMax := left.Position.X + left.Width
	rightMin := right.Position.X
	rightMax := right.Position.X + right.Width
	return leftMin < rightMax && leftMax > rightMin
}

func (left *collider) overlapsY(right *collider) bool {
	leftMin := left.Position.Y
	leftMax := left.Position.Y + left.Height
	rightMin := right.Position.Y
	rightMax := right.Position.Y + right.Height
	return leftMin < rightMax && leftMax > rightMin
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		digits[index] = '-'
	}
	return string(digits[index:])
}
