package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kos"
	"surface"
	surfacetinygl "surface/tinygl"
)

const (
	windowX      = 72
	windowY      = 48
	windowWidth  = 960
	windowHeight = 720

	headerHeight             = 64
	footerHeight             = 76
	viewInset                = 16
	panelRadius              = 16
	transparentCursorBytes   = 32 * 32 * 4
	mouseLookDegreesPerPixel = 0.22
)

const (
	glColorBufferBit = 0x00004000
	glDepthBufferBit = 0x00000100

	glLines     = 0x0001
	glTriangles = 0x0004

	glBack         = 0x0405
	glCullFace     = 0x0B44
	glDepthTest    = 0x0B71
	glSmooth       = 0x1D01
	glModelview    = 0x1700
	glProjection   = 0x1701
	glFrontAndBack = 0x0408
	glLineMode     = 0x1B01
	glFillMode     = 0x1B02
)

const (
	keyEscape          = 27
	keySpace           = ' '
	keyReloadLow       = 'r'
	keyReloadHigh      = 'R'
	keyWireLow         = 't'
	keyWireHigh        = 'T'
	keyForwardLow      = 'w'
	keyForwardHigh     = 'W'
	keyBackLow         = 's'
	keyBackHigh        = 'S'
	keyLeftLow         = 'a'
	keyLeftHigh        = 'A'
	keyRightLow        = 'd'
	keyRightHigh       = 'D'
	keyStrafeLeftLow   = 'q'
	keyStrafeLeftHigh  = 'Q'
	keyStrafeRightLow  = 'e'
	keyStrafeRightHigh = 'E'

	scanEscape = 1
	scanUp     = 72
	scanLeft   = 75
	scanRight  = 77
	scanDown   = 80
)

const (
	colorInk         kos.Color = 0x0B1320
	colorPanelTop    kos.Color = 0x15314A
	colorPanelBottom kos.Color = 0x0E2235
	colorBorder      kos.Color = 0x456785
	colorCard        kos.Color = 0x12283C
	colorText        kos.Color = 0xF2F7FB
	colorMuted       kos.Color = 0xA2B9CC
	colorAccent      kos.Color = 0x8BE9C1
	colorWarn        kos.Color = 0xF6BD60
	colorError       kos.Color = 0xFF7A93
)

type vec3 struct {
	X float32
	Y float32
	Z float32
}

type rgbColor struct {
	R uint8
	G uint8
	B uint8
}

type xmlAttr struct {
	Name  string
	Value string
}

type xmlNode struct {
	Name     string
	Attrs    []xmlAttr
	Children []xmlNode
}

type scene struct {
	Title      string
	Path       string
	Background rgbColor
	Camera     sceneCamera
	Player     playerState
	Root       sceneNode
	Colliders  []collider
	Drawables  int
}

type sceneCamera struct {
	FOV    float32
	Near   float32
	Far    float32
	Height float32
}

type playerState struct {
	Position         vec3
	BaseY            float32
	Yaw              float32
	Pitch            float32
	Radius           float32
	Height           float32
	MoveSpeed        float32
	TurnSpeed        float32
	JumpSpeed        float32
	Gravity          float32
	VerticalVelocity float32
	OnGround         bool
	HP               int
}

type sceneNode struct {
	Kind       string
	ID         string
	Position   vec3
	Rotation   vec3
	Scale      vec3
	Size       vec3
	Color      rgbColor
	Visible    bool
	Solid      bool
	Static     bool
	Mass       float32
	Friction   float32
	Bounciness float32
	GridSize   float32
	GridStep   float32
	Children   []sceneNode
}

type collider struct {
	ID       string
	Min      vec3
	Max      vec3
	Mass     float32
	Friction float32
	Static   bool
}

type sceneApp struct {
	presenter      surface.Presenter
	canvas         *surface.Buffer
	layer          surfacetinygl.Layer
	scenePath      string
	scene          scene
	sceneLoaded    bool
	lastError      string
	wireframe      bool
	viewRect       surface.Rect
	viewWindowRect surface.Rect
	headerRect     surface.Rect
	footerRect     surface.Rect
	frameRect      surface.Rect
	projection     [16]float32
	lastStepNS     uint64
	mouseCaptured  bool
	mouseCursor    kos.CursorHandle
	prevEventMask  kos.EventMask
}

type xmlParser struct {
	input string
	pos   int
}

func defaultScene() scene {
	return scene{
		Title:      "Tagix xml3D Engine",
		Background: mustParseColor("#0b1320"),
		Camera: sceneCamera{
			FOV:    60,
			Near:   0.2,
			Far:    96,
			Height: 1.2,
		},
		Player: playerState{
			Position:  vec3{X: 0, Y: 0, Z: 6},
			BaseY:     0,
			Yaw:       180,
			Pitch:     0,
			Radius:    0.35,
			Height:    1.8,
			MoveSpeed: 0.45,
			TurnSpeed: 8,
			JumpSpeed: 4.8,
			Gravity:   12.0,
			OnGround:  true,
			HP:        100,
		},
		Root: sceneNode{
			Kind:    "group",
			Visible: true,
			Scale:   vec3{X: 1, Y: 1, Z: 1},
		},
	}
}

func main() {
	app := newSceneApp(resolveScenePath(os.Args))
	app.run()
	os.Exit(0)
}

func newSceneApp(scenePath string) *sceneApp {
	presenter := surface.NewPresenter(windowX, windowY, windowWidth, windowHeight, "Tagix xml3D Engine")
	app := &sceneApp{
		presenter: presenter,
		scenePath: scenePath,
	}
	app.prevEventMask = kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskMouse | kos.EventMaskMouseActiveWindowOnly)
	app.syncWindowInfo()
	app.reloadScene()
	return app
}

func resolveScenePath(args []string) string {
	if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
		return args[1]
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return filepath.Join(cwd, "demo.xml")
	}
	return "demo.xml"
}

func (app *sceneApp) resolveFallbackScenePath() string {
	if len(os.Args) > 1 {
		return app.scenePath
	}
	exe := strings.TrimSpace(os.Args[0])
	if exe == "" {
		return app.scenePath
	}
	return filepath.Join(filepath.Dir(exe), "demo.xml")
}

func (app *sceneApp) reloadScene() {
	scenePath := app.scenePath
	loaded, err := loadScene(scenePath)
	if err != nil && len(os.Args) <= 1 {
		fallback := app.resolveFallbackScenePath()
		if fallback != scenePath {
			if retry, retryErr := loadScene(fallback); retryErr == nil {
				loaded = retry
				scenePath = fallback
				err = nil
			}
		}
	}
	if err != nil {
		app.scene = defaultScene()
		app.scene.Path = scenePath
		app.sceneLoaded = false
		app.lastError = err.Error()
		app.presenter.SetTitle("Tagix xml3D Engine / load failed")
		return
	}
	app.scene = loaded
	app.scene.Path = scenePath
	app.sceneLoaded = true
	app.lastError = ""
	title := "Tagix xml3D Engine"
	if app.scene.Title != "" {
		title = title + " / " + app.scene.Title
	}
	app.presenter.SetTitle(title)
}

func loadScene(path string) (scene, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return scene{}, fmt.Errorf("read scene %q: %w", path, err)
	}
	root, err := parseXMLDocument(string(data))
	if err != nil {
		return scene{}, fmt.Errorf("parse scene %q: %w", path, err)
	}
	if !strings.EqualFold(root.Name, "scene") {
		return scene{}, fmt.Errorf("root tag must be <scene>, got <%s>", root.Name)
	}

	result := defaultScene()
	result.Path = path
	result.Title = attrString(root, "title", result.Title)
	result.Background = attrColor(root, "background", result.Background)
	result.Player.MoveSpeed = attrFloat(root, "move_speed", result.Player.MoveSpeed)
	result.Player.TurnSpeed = attrFloat(root, "turn_speed", result.Player.TurnSpeed)
	result.Player.JumpSpeed = attrFloat(root, "jump_speed", result.Player.JumpSpeed)
	result.Player.Gravity = attrFloat(root, "gravity", result.Player.Gravity)

	for _, child := range root.Children {
		switch normalizeTag(child.Name) {
		case "player":
			parsePlayerNode(&result, child)
		case "camera":
			parseCameraNode(&result, child)
		case "group", "cube", "grid":
			node, ok := parseSceneNode(child)
			if ok {
				result.Root.Children = append(result.Root.Children, node)
			}
		}
	}

	result.Drawables = countDrawables(result.Root)
	result.Colliders = flattenColliders(result.Root, vec3{}, vec3{X: 1, Y: 1, Z: 1})
	result.Player.BaseY = result.Player.Position.Y
	result.Player.VerticalVelocity = 0
	result.Player.OnGround = true
	return result, nil
}

func parsePlayerNode(target *scene, node xmlNode) {
	if target == nil {
		return
	}
	target.Player.Position = attrVec3(node, target.Player.Position)
	target.Player.Yaw = attrFloat(node, "yaw", target.Player.Yaw)
	target.Player.Pitch = attrFloat(node, "pitch", target.Player.Pitch)
	target.Player.Radius = attrFloat(node, "radius", target.Player.Radius)
	target.Player.Height = attrFloat(node, "height", target.Player.Height)
	target.Player.MoveSpeed = attrFloat(node, "speed", target.Player.MoveSpeed)
	target.Player.JumpSpeed = attrFloat(node, "jump_speed", target.Player.JumpSpeed)
	target.Player.Gravity = attrFloat(node, "gravity", target.Player.Gravity)
	target.Player.HP = attrInt(node, "hp", target.Player.HP)
}

func parseCameraNode(target *scene, node xmlNode) {
	if target == nil {
		return
	}
	target.Camera.FOV = attrFloat(node, "fov", target.Camera.FOV)
	target.Camera.Near = attrFloat(node, "near", target.Camera.Near)
	target.Camera.Far = attrFloat(node, "far", target.Camera.Far)
	target.Camera.Height = attrFloat(node, "height", target.Camera.Height)
}

func parseSceneNode(node xmlNode) (sceneNode, bool) {
	kind := normalizeTag(node.Name)
	switch kind {
	case "group":
		result := defaultNode(kind)
		fillCommonNodeAttrs(&result, node)
		for _, child := range node.Children {
			childNode, ok := parseSceneNode(child)
			if ok {
				result.Children = append(result.Children, childNode)
			}
		}
		return result, true
	case "cube":
		result := defaultNode(kind)
		fillCommonNodeAttrs(&result, node)
		size := attrFloat(node, "size", 0)
		if size > 0 {
			result.Size = vec3{X: size, Y: size, Z: size}
		}
		result.Size.X = attrFloat(node, "width", result.Size.X)
		result.Size.Y = attrFloat(node, "height", result.Size.Y)
		result.Size.Z = attrFloat(node, "depth", result.Size.Z)
		return result, true
	case "grid":
		result := defaultNode(kind)
		fillCommonNodeAttrs(&result, node)
		result.GridSize = attrFloat(node, "size", 12)
		result.GridStep = attrFloat(node, "step", 1)
		if result.GridStep <= 0 {
			result.GridStep = 1
		}
		return result, true
	default:
		return sceneNode{}, false
	}
}

func defaultNode(kind string) sceneNode {
	return sceneNode{
		Kind:     kind,
		Visible:  true,
		Scale:    vec3{X: 1, Y: 1, Z: 1},
		Size:     vec3{X: 1, Y: 1, Z: 1},
		Color:    mustParseColor("#84a59d"),
		GridSize: 12,
		GridStep: 1,
	}
}

func fillCommonNodeAttrs(target *sceneNode, node xmlNode) {
	if target == nil {
		return
	}
	target.ID = attrString(node, "id", target.ID)
	target.Position = attrVec3(node, target.Position)
	target.Rotation = vec3{
		X: attrFloat(node, "rx", target.Rotation.X),
		Y: attrFloat(node, "ry", target.Rotation.Y),
		Z: attrFloat(node, "rz", target.Rotation.Z),
	}
	target.Scale = vec3{
		X: attrFloat(node, "sx", target.Scale.X),
		Y: attrFloat(node, "sy", target.Scale.Y),
		Z: attrFloat(node, "sz", target.Scale.Z),
	}
	target.Color = attrColor(node, "color", target.Color)
	target.Visible = attrBool(node, "visible", target.Visible)
	target.Solid = attrBool(node, "solid", target.Solid)
	target.Static = attrBool(node, "static", target.Static)
	target.Mass = attrFloat(node, "mass", target.Mass)
	target.Friction = attrFloat(node, "friction", target.Friction)
	target.Bounciness = attrFloat(node, "bounce", target.Bounciness)
}

func normalizeTag(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func attrString(node xmlNode, name string, fallback string) string {
	value, ok := findAttr(node, name)
	if !ok {
		return fallback
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func attrFloat(node xmlNode, name string, fallback float32) float32 {
	value, ok := findAttr(node, name)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 32)
	if err != nil {
		return fallback
	}
	return float32(parsed)
}

func attrInt(node xmlNode, name string, fallback int) int {
	value, ok := findAttr(node, name)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func attrBool(node xmlNode, name string, fallback bool) bool {
	value, ok := findAttr(node, name)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func attrColor(node xmlNode, name string, fallback rgbColor) rgbColor {
	value, ok := findAttr(node, name)
	if !ok {
		return fallback
	}
	color, ok := parseColor(value)
	if !ok {
		return fallback
	}
	return color
}

func attrVec3(node xmlNode, fallback vec3) vec3 {
	return vec3{
		X: attrFloat(node, "x", fallback.X),
		Y: attrFloat(node, "y", fallback.Y),
		Z: attrFloat(node, "z", fallback.Z),
	}
}

func findAttr(node xmlNode, name string) (string, bool) {
	for _, attr := range node.Attrs {
		if strings.EqualFold(attr.Name, name) {
			return attr.Value, true
		}
	}
	return "", false
}

func parseXMLDocument(input string) (xmlNode, error) {
	parser := xmlParser{input: input}
	for {
		parser.skipIgnored()
		if parser.eof() {
			return xmlNode{}, fmt.Errorf("empty document")
		}
		if parser.peek() != '<' {
			parser.pos++
			continue
		}
		return parser.parseElement()
	}
}

func (parser *xmlParser) parseElement() (xmlNode, error) {
	if !parser.consume("<") {
		return xmlNode{}, parser.errorf("expected <")
	}
	if parser.consume("/") {
		return xmlNode{}, parser.errorf("unexpected closing tag")
	}
	name := parser.readName()
	if name == "" {
		return xmlNode{}, parser.errorf("missing tag name")
	}
	node := xmlNode{Name: name}
	for {
		parser.skipSpaces()
		switch {
		case parser.consume("/>"):
			return node, nil
		case parser.consume(">"):
			for {
				parser.skipIgnored()
				if parser.consume("</") {
					endName := parser.readName()
					if !strings.EqualFold(endName, name) {
						return xmlNode{}, parser.errorf("closing tag </%s> does not match <%s>", endName, name)
					}
					parser.skipSpaces()
					if !parser.consume(">") {
						return xmlNode{}, parser.errorf("missing > after </%s>", endName)
					}
					return node, nil
				}
				if parser.eof() {
					return xmlNode{}, parser.errorf("unexpected end of document inside <%s>", name)
				}
				if parser.peek() != '<' {
					parser.pos++
					continue
				}
				child, err := parser.parseElement()
				if err != nil {
					return xmlNode{}, err
				}
				node.Children = append(node.Children, child)
			}
		default:
			attrName := parser.readName()
			if attrName == "" {
				return xmlNode{}, parser.errorf("invalid attribute in <%s>", name)
			}
			parser.skipSpaces()
			if !parser.consume("=") {
				return xmlNode{}, parser.errorf("missing = after attribute %s", attrName)
			}
			parser.skipSpaces()
			value, err := parser.readAttributeValue()
			if err != nil {
				return xmlNode{}, err
			}
			node.Attrs = append(node.Attrs, xmlAttr{Name: attrName, Value: value})
		}
	}
}

func (parser *xmlParser) readAttributeValue() (string, error) {
	if parser.eof() {
		return "", parser.errorf("unexpected end of attribute")
	}
	quote := parser.peek()
	if quote != '"' && quote != '\'' {
		return "", parser.errorf("attribute value must be quoted")
	}
	parser.pos++
	start := parser.pos
	for !parser.eof() && parser.peek() != quote {
		parser.pos++
	}
	if parser.eof() {
		return "", parser.errorf("unterminated attribute value")
	}
	value := parser.input[start:parser.pos]
	parser.pos++
	return xmlUnescape(value), nil
}

func (parser *xmlParser) readName() string {
	start := parser.pos
	for !parser.eof() {
		ch := parser.peek()
		if !isXMLNameChar(ch) {
			break
		}
		parser.pos++
	}
	return parser.input[start:parser.pos]
}

func isXMLNameChar(ch byte) bool {
	switch {
	case ch >= 'a' && ch <= 'z':
		return true
	case ch >= 'A' && ch <= 'Z':
		return true
	case ch >= '0' && ch <= '9':
		return true
	case ch == '_' || ch == '-' || ch == ':' || ch == '.':
		return true
	default:
		return false
	}
}

func xmlUnescape(value string) string {
	replacer := strings.NewReplacer(
		"&quot;", "\"",
		"&apos;", "'",
		"&lt;", "<",
		"&gt;", ">",
		"&amp;", "&",
	)
	return replacer.Replace(value)
}

func (parser *xmlParser) skipIgnored() {
	for {
		parser.skipSpaces()
		switch {
		case parser.consume("<?"):
			parser.skipUntil("?>")
		case parser.consume("<!--"):
			parser.skipUntil("-->")
		default:
			return
		}
	}
}

func (parser *xmlParser) skipUntil(marker string) {
	index := strings.Index(parser.input[parser.pos:], marker)
	if index < 0 {
		parser.pos = len(parser.input)
		return
	}
	parser.pos += index + len(marker)
}

func (parser *xmlParser) skipSpaces() {
	for !parser.eof() {
		switch parser.peek() {
		case ' ', '\t', '\r', '\n':
			parser.pos++
		default:
			return
		}
	}
}

func (parser *xmlParser) consume(token string) bool {
	if strings.HasPrefix(parser.input[parser.pos:], token) {
		parser.pos += len(token)
		return true
	}
	return false
}

func (parser *xmlParser) peek() byte {
	if parser.eof() {
		return 0
	}
	return parser.input[parser.pos]
}

func (parser *xmlParser) eof() bool {
	return parser.pos >= len(parser.input)
}

func (parser *xmlParser) errorf(format string, args ...interface{}) error {
	return fmt.Errorf("%s at byte %d", fmt.Sprintf(format, args...), parser.pos)
}

func parseColor(value string) (rgbColor, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return rgbColor{}, false
	}
	if strings.HasPrefix(text, "#") {
		text = text[1:]
	}
	if len(text) == 3 {
		text = strings.Repeat(string(text[0]), 2) + strings.Repeat(string(text[1]), 2) + strings.Repeat(string(text[2]), 2)
	}
	if len(text) != 6 {
		return rgbColor{}, false
	}
	parsed, err := strconv.ParseUint(text, 16, 32)
	if err != nil {
		return rgbColor{}, false
	}
	return rgbColor{
		R: uint8(parsed >> 16),
		G: uint8((parsed >> 8) & 0xFF),
		B: uint8(parsed & 0xFF),
	}, true
}

func mustParseColor(value string) rgbColor {
	color, ok := parseColor(value)
	if !ok {
		return rgbColor{}
	}
	return color
}

func (color rgbColor) toKOS() kos.Color {
	return kos.Color(uint32(color.R)<<16 | uint32(color.G)<<8 | uint32(color.B))
}

func (color rgbColor) withShade(factor float32) rgbColor {
	return rgbColor{
		R: scaleColor(color.R, factor),
		G: scaleColor(color.G, factor),
		B: scaleColor(color.B, factor),
	}
}

func scaleColor(value uint8, factor float32) uint8 {
	scaled := int(float32(value) * factor)
	if scaled < 0 {
		scaled = 0
	}
	if scaled > 255 {
		scaled = 255
	}
	return uint8(scaled)
}

func countDrawables(node sceneNode) int {
	count := 0
	switch node.Kind {
	case "cube", "grid":
		count++
	}
	for _, child := range node.Children {
		count += countDrawables(child)
	}
	return count
}

func flattenColliders(node sceneNode, parentPos vec3, parentScale vec3) []collider {
	worldPos := vec3{
		X: parentPos.X + node.Position.X,
		Y: parentPos.Y + node.Position.Y,
		Z: parentPos.Z + node.Position.Z,
	}
	worldScale := vec3{
		X: parentScale.X * nonZero(node.Scale.X),
		Y: parentScale.Y * nonZero(node.Scale.Y),
		Z: parentScale.Z * nonZero(node.Scale.Z),
	}
	var colliders []collider
	if node.Kind == "cube" && node.Solid && node.Visible {
		size := vec3{
			X: abs32(node.Size.X * worldScale.X),
			Y: abs32(node.Size.Y * worldScale.Y),
			Z: abs32(node.Size.Z * worldScale.Z),
		}
		half := vec3{X: size.X * 0.5, Y: size.Y * 0.5, Z: size.Z * 0.5}
		colliders = append(colliders, collider{
			ID:       node.ID,
			Min:      vec3{X: worldPos.X - half.X, Y: worldPos.Y - half.Y, Z: worldPos.Z - half.Z},
			Max:      vec3{X: worldPos.X + half.X, Y: worldPos.Y + half.Y, Z: worldPos.Z + half.Z},
			Mass:     node.Mass,
			Friction: node.Friction,
			Static:   node.Static,
		})
	}
	for _, child := range node.Children {
		colliders = append(colliders, flattenColliders(child, worldPos, worldScale)...)
	}
	return colliders
}

func nonZero(value float32) float32 {
	if value == 0 {
		return 1
	}
	return value
}

func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

func (app *sceneApp) run() {
	defer app.shutdown()
	app.redrawFull()
	for {
		event := kos.WaitEventFor(2)
		switch event {
		case kos.EventNone:
			if app.sceneLoaded {
				app.step()
			}
		case kos.EventRedraw:
			app.syncWindowInfo()
			app.redrawFull()
		case kos.EventMouse:
			app.handleMouse()
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				return
			}
		case kos.EventKey:
			if app.handleKey(kos.ReadKey()) {
				return
			}
		}
	}
}

func (app *sceneApp) handleKey(key kos.KeyEvent) bool {
	switch {
	case key.Code == keyEscape || key.ScanCode == scanEscape:
		if app.mouseCaptured {
			app.releaseMouseCapture()
			app.presentFooter()
			return false
		}
		return true
	case key.Code == keySpace:
		app.jump()
	case key.Code == keyReloadLow || key.Code == keyReloadHigh:
		app.reloadScene()
		app.redrawFull()
	case key.Code == keyWireLow || key.Code == keyWireHigh:
		app.wireframe = !app.wireframe
		app.presentHeader()
		app.renderScene()
	case key.Code == keyForwardLow || key.Code == keyForwardHigh || key.ScanCode == scanUp:
		app.moveRelative(app.scene.Player.MoveSpeed, 0)
	case key.Code == keyBackLow || key.Code == keyBackHigh || key.ScanCode == scanDown:
		app.moveRelative(-app.scene.Player.MoveSpeed, 0)
	case key.Code == keyStrafeLeftLow || key.Code == keyStrafeLeftHigh:
		app.moveRelative(0, -app.scene.Player.MoveSpeed)
	case key.Code == keyStrafeRightLow || key.Code == keyStrafeRightHigh:
		app.moveRelative(0, app.scene.Player.MoveSpeed)
	case key.Code == keyLeftLow || key.Code == keyLeftHigh || key.ScanCode == scanLeft:
		app.scene.Player.Yaw -= app.scene.Player.TurnSpeed
		app.presentFooter()
		app.renderScene()
	case key.Code == keyRightLow || key.Code == keyRightHigh || key.ScanCode == scanRight:
		app.scene.Player.Yaw += app.scene.Player.TurnSpeed
		app.presentFooter()
		app.renderScene()
	}
	return false
}

func (app *sceneApp) handleMouse() {
	if !app.sceneLoaded {
		return
	}
	if app.mouseCaptured {
		app.applyMouseLook()
		return
	}
	buttons := kos.MouseButtons()
	if !buttons.LeftPressed {
		return
	}
	pos := kos.MouseWindowPosition()
	if !app.viewWindowRect.Contains(pos.X, pos.Y) {
		return
	}
	app.captureMouse()
}

func (app *sceneApp) ensureMouseCursor() bool {
	if app.mouseCursor != 0 {
		return true
	}
	image := make([]byte, transparentCursorBytes)
	app.mouseCursor = kos.LoadCursorARGB(image, 0, 0)
	return app.mouseCursor != 0
}

func (app *sceneApp) captureMouse() {
	if app.mouseCaptured {
		return
	}
	if app.ensureMouseCursor() {
		kos.SetCursor(app.mouseCursor)
	}
	app.mouseCaptured = true
	app.centerMousePointer()
	app.presentFooter()
}

func (app *sceneApp) releaseMouseCapture() {
	if !app.mouseCaptured {
		return
	}
	app.mouseCaptured = false
	kos.RestoreDefaultCursor()
}

func (app *sceneApp) shutdown() {
	app.releaseMouseCapture()
	if app.mouseCursor != 0 {
		kos.DeleteCursor(app.mouseCursor)
		app.mouseCursor = 0
	}
	if app.prevEventMask != 0 {
		kos.SwapEventMask(app.prevEventMask)
		app.prevEventMask = 0
	}
}

func (app *sceneApp) applyMouseLook() {
	centerX, centerY := app.viewportCenterWindow()
	pos := kos.MouseWindowPosition()
	deltaX := pos.X - centerX
	deltaY := pos.Y - centerY
	if deltaX == 0 && deltaY == 0 {
		return
	}
	app.scene.Player.Yaw += float32(deltaX) * mouseLookDegreesPerPixel
	app.scene.Player.Pitch -= float32(deltaY) * mouseLookDegreesPerPixel
	app.scene.Player.Pitch = clampFloat32(app.scene.Player.Pitch, -75, 75)
	app.centerMousePointer()
	app.presentFooter()
	app.renderScene()
}

func (app *sceneApp) moveRelative(forward float32, strafe float32) {
	if !app.sceneLoaded {
		return
	}
	radians := float32(app.scene.Player.Yaw * float32(math.Pi) / 180.0)
	forwardVector := vec3{
		X: float32(math.Sin(float64(radians))),
		Y: 0,
		Z: -float32(math.Cos(float64(radians))),
	}
	rightVector := vec3{
		X: float32(math.Cos(float64(radians))),
		Y: 0,
		Z: float32(math.Sin(float64(radians))),
	}
	delta := vec3{
		X: forwardVector.X*forward + rightVector.X*strafe,
		Y: 0,
		Z: forwardVector.Z*forward + rightVector.Z*strafe,
	}
	app.tryMove(delta)
}

func (app *sceneApp) tryMove(delta vec3) {
	current := app.scene.Player.Position
	nextX := vec3{X: current.X + delta.X, Y: current.Y, Z: current.Z}
	if !app.collides(nextX) {
		current.X = nextX.X
	}
	nextZ := vec3{X: current.X, Y: current.Y, Z: current.Z + delta.Z}
	if !app.collides(nextZ) {
		current.Z = nextZ.Z
	}
	app.scene.Player.Position = current
	app.presentFooter()
	app.renderScene()
}

func (app *sceneApp) collides(position vec3) bool {
	playerMin := vec3{
		X: position.X - app.scene.Player.Radius,
		Y: position.Y,
		Z: position.Z - app.scene.Player.Radius,
	}
	playerMax := vec3{
		X: position.X + app.scene.Player.Radius,
		Y: position.Y + app.scene.Player.Height,
		Z: position.Z + app.scene.Player.Radius,
	}
	for _, collider := range app.scene.Colliders {
		if aabbOverlap(playerMin, playerMax, collider.Min, collider.Max) {
			return true
		}
	}
	return false
}

func (app *sceneApp) jump() {
	if !app.sceneLoaded || !app.scene.Player.OnGround {
		return
	}
	app.scene.Player.OnGround = false
	app.scene.Player.VerticalVelocity = app.scene.Player.JumpSpeed
	app.lastStepNS = kos.UptimeNanoseconds()
	app.presentFooter()
	app.renderScene()
}

func (app *sceneApp) updateVerticalPhysics(deltaSeconds float32) bool {
	if !app.sceneLoaded {
		return false
	}
	player := &app.scene.Player
	if player.OnGround {
		if player.Position.Y != player.BaseY {
			player.Position.Y = player.BaseY
			return true
		}
		return false
	}
	player.VerticalVelocity -= player.Gravity * deltaSeconds
	nextY := player.Position.Y + player.VerticalVelocity*deltaSeconds
	if nextY <= player.BaseY {
		player.Position.Y = player.BaseY
		player.VerticalVelocity = 0
		player.OnGround = true
		return true
	}
	player.Position.Y = nextY
	return true
}

func aabbOverlap(minA vec3, maxA vec3, minB vec3, maxB vec3) bool {
	if maxA.X <= minB.X || minA.X >= maxB.X {
		return false
	}
	if maxA.Y <= minB.Y || minA.Y >= maxB.Y {
		return false
	}
	if maxA.Z <= minB.Z || minA.Z >= maxB.Z {
		return false
	}
	return true
}

func (app *sceneApp) step() {
	now := kos.UptimeNanoseconds()
	if app.lastStepNS == 0 {
		app.lastStepNS = now
		app.renderScene()
		return
	}
	deltaSeconds := float32(now-app.lastStepNS) / 1000000000.0
	app.lastStepNS = now
	if deltaSeconds < 0 {
		deltaSeconds = 0
	}
	if deltaSeconds > 0.05 {
		deltaSeconds = 0.05
	}
	if app.updateVerticalPhysics(deltaSeconds) {
		app.presentFooter()
	}
	app.renderScene()
}

func (app *sceneApp) viewportCenterWindow() (int, int) {
	return app.viewWindowRect.X + app.viewWindowRect.Width/2, app.viewWindowRect.Y + app.viewWindowRect.Height/2
}

func (app *sceneApp) viewportCenterScreen() (int, int) {
	centerX, centerY := app.viewportCenterWindow()
	info, _, ok := kos.ReadCurrentThreadInfo()
	if ok {
		return info.WindowPosition.X + centerX, info.WindowPosition.Y + centerY
	}
	return app.presenter.X + centerX, app.presenter.Y + centerY
}

func (app *sceneApp) centerMousePointer() {
	screenX, screenY := app.viewportCenterScreen()
	kos.SetMousePointerPosition(screenX, screenY)
}

func clampFloat32(value float32, min float32, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (app *sceneApp) syncWindowInfo() {
	info, _, ok := kos.ReadCurrentThreadInfo()
	if ok && info.WindowSize.X > 0 && info.WindowSize.Y > 0 {
		if info.WindowSize.X != app.presenter.Width || info.WindowSize.Y != app.presenter.Height {
			app.presenter.SetSize(info.WindowSize.X, info.WindowSize.Y)
		}
		app.presenter.X = info.WindowPosition.X
		app.presenter.Y = info.WindowPosition.Y
	}
	client := app.presenter.Client
	if app.canvas == nil {
		app.canvas = surface.NewBuffer(client.Width, client.Height)
	} else if app.canvas.Width() != client.Width || app.canvas.Height() != client.Height {
		app.canvas.Resize(client.Width, client.Height)
	}

	app.headerRect = surface.Rect{X: 0, Y: 0, Width: client.Width, Height: headerHeight}
	app.footerRect = surface.Rect{X: 0, Y: client.Height - footerHeight, Width: client.Width, Height: footerHeight}
	app.frameRect = surface.Rect{
		X:      viewInset,
		Y:      app.headerRect.Height + 10,
		Width:  client.Width - viewInset*2,
		Height: client.Height - app.headerRect.Height - app.footerRect.Height - 20,
	}
	if app.frameRect.Width < 64 {
		app.frameRect.Width = 64
	}
	if app.frameRect.Height < 64 {
		app.frameRect.Height = 64
	}
	app.viewRect = surface.Rect{
		X:      app.frameRect.X + 12,
		Y:      app.frameRect.Y + 12,
		Width:  app.frameRect.Width - 24,
		Height: app.frameRect.Height - 24,
	}
	if app.viewRect.Width < 1 {
		app.viewRect.Width = 1
	}
	if app.viewRect.Height < 1 {
		app.viewRect.Height = 1
	}
	app.viewWindowRect = surface.Rect{
		X:      client.X + app.viewRect.X,
		Y:      client.Y + app.viewRect.Y,
		Width:  app.viewRect.Width,
		Height: app.viewRect.Height,
	}
	if app.mouseCaptured {
		app.centerMousePointer()
	}
}

func (app *sceneApp) redrawFull() {
	app.drawChrome()
	app.presenter.PresentFull(app.canvas)
	app.renderScene()
}

func (app *sceneApp) presentHeader() {
	if app.canvas == nil {
		return
	}
	app.canvas.FillRect(app.headerRect.X, app.headerRect.Y, app.headerRect.Width, app.headerRect.Height, colorCard)
	app.drawHeader()
	app.presenter.PresentRect(app.canvas, app.headerRect)
}

func (app *sceneApp) presentFooter() {
	if app.canvas == nil {
		return
	}
	app.canvas.FillRect(app.footerRect.X, app.footerRect.Y, app.footerRect.Width, app.footerRect.Height, colorCard)
	app.drawFooter()
	app.presenter.PresentRect(app.canvas, app.footerRect)
}

func (app *sceneApp) drawChrome() {
	if app.canvas == nil {
		return
	}
	bounds := app.canvas.Bounds()
	app.canvas.FillRectGradient(0, 0, bounds.Width, bounds.Height, surface.Gradient{
		From:      colorPanelTop,
		To:        colorPanelBottom,
		Direction: surface.GradientVertical,
	})
	app.canvas.FillRect(0, 0, bounds.Width, headerHeight, colorCard)
	app.canvas.FillRect(0, app.footerRect.Y, bounds.Width, footerHeight, colorCard)
	app.canvas.DrawShadowRounded(app.frameRect, surface.Shadow{
		OffsetX: 0,
		OffsetY: 2,
		Blur:    4,
		Color:   0x000000,
		Alpha:   88,
	}, uniformRadii(panelRadius))
	app.canvas.FillRoundedRect(app.frameRect.X, app.frameRect.Y, app.frameRect.Width, app.frameRect.Height, uniformRadii(panelRadius), colorCard)
	app.canvas.StrokeRoundedRectWidth(app.frameRect.X, app.frameRect.Y, app.frameRect.Width, app.frameRect.Height, uniformRadii(panelRadius), 1, colorBorder)
	app.canvas.FillRect(app.viewRect.X, app.viewRect.Y, app.viewRect.Width, app.viewRect.Height, app.scene.Background.toKOS())
	app.drawHeader()
	app.drawFooter()
	if !app.sceneLoaded {
		app.drawErrorView()
	}
}

func uniformRadii(radius int) surface.CornerRadii {
	return surface.CornerRadii{
		TopLeft:     radius,
		TopRight:    radius,
		BottomRight: radius,
		BottomLeft:  radius,
	}
}

func (app *sceneApp) drawHeader() {
	title := "Tagix xml3D Engine"
	if app.sceneLoaded && app.scene.Title != "" {
		title = title + " / " + app.scene.Title
	}
	app.canvas.DrawText(18, 16, colorText, title)
	app.canvas.DrawText(18, 36, colorMuted, trimText(app.scene.Path, 78))
	mode := "fill"
	if app.wireframe {
		mode = "wireframe"
	}
	status := fmt.Sprintf("objects=%d colliders=%d mode=%s", app.scene.Drawables, len(app.scene.Colliders), mode)
	app.canvas.DrawText(app.headerRect.Width-8-len(status)*8, 16, colorAccent, status)
}

func (app *sceneApp) drawFooter() {
	left := "LMB capture  W/S move  A/D turn  Q/E strafe  Space jump  T wireframe  R reload  Esc release/exit"
	app.canvas.DrawText(18, app.footerRect.Y+16, colorMuted, left)
	position := fmt.Sprintf(
		"player x=%.2f y=%.2f z=%.2f yaw=%.1f pitch=%.1f hp=%d jump=%s mouse=%s",
		app.scene.Player.Position.X,
		app.scene.Player.Position.Y,
		app.scene.Player.Position.Z,
		app.scene.Player.Yaw,
		app.scene.Player.Pitch,
		app.scene.Player.HP,
		jumpStateText(app.scene.Player.OnGround),
		mouseCaptureText(app.mouseCaptured),
	)
	app.canvas.DrawText(18, app.footerRect.Y+38, colorText, position)
	if app.lastError != "" {
		app.canvas.DrawText(18, app.footerRect.Y+58, colorError, trimText(app.lastError, 110))
		return
	}
	info := fmt.Sprintf(
		"camera fov=%.0f near=%.2f far=%.0f speed=%.2f turn=%.1f",
		app.scene.Camera.FOV,
		app.scene.Camera.Near,
		app.scene.Camera.Far,
		app.scene.Player.MoveSpeed,
		app.scene.Player.TurnSpeed,
	)
	app.canvas.DrawText(18, app.footerRect.Y+58, colorWarn, info)
}

func jumpStateText(onGround bool) string {
	if onGround {
		return "ready"
	}
	return "air"
}

func mouseCaptureText(captured bool) string {
	if captured {
		return "captured"
	}
	return "free"
}

func trimText(value string, limit int) string {
	if limit <= 3 || len(value) <= limit {
		return value
	}
	return value[:limit-3] + "..."
}

func (app *sceneApp) drawErrorView() {
	lines := []string{
		"Scene file was not loaded.",
		"Place demo.xml in the current directory or pass a file path as the first argument.",
		"Supported tags: <scene>, <camera>, <player>, <group>, <cube>, <grid>.",
	}
	y := app.viewRect.Y + 24
	for index, line := range lines {
		color := colorMuted
		if index == 0 {
			color = colorError
		}
		app.canvas.DrawText(app.viewRect.X+18, y, color, line)
		y += 20
	}
}

func (app *sceneApp) renderScene() {
	if !app.sceneLoaded {
		return
	}
	if app.layer.Render(app.viewWindowRect, func(gl *kos.TinyGL, ctx *kos.TinyGLContext) {
		app.drawScene(gl)
	}) {
		return
	}
	app.sceneLoaded = false
	app.lastError = "tinygl load/context init failed"
	app.drawChrome()
	app.presenter.PresentFull(app.canvas)
}

func (app *sceneApp) drawScene(gl *kos.TinyGL) {
	r, g, b := colorToFloat(app.scene.Background)
	gl.ClearColor(r, g, b, 1)
	gl.Clear(glColorBufferBit | glDepthBufferBit)
	gl.Enable(glDepthTest)
	gl.Enable(glCullFace)
	gl.CullFace(glBack)
	gl.ShadeModel(glSmooth)
	if app.wireframe {
		gl.PolygonMode(glFrontAndBack, glLineMode)
	} else {
		gl.PolygonMode(glFrontAndBack, glFillMode)
	}

	gl.MatrixMode(glProjection)
	gl.LoadMatrix(app.perspectiveMatrix(app.viewRect.Width, app.viewRect.Height))
	gl.MatrixMode(glModelview)
	gl.LoadIdentity()

	eyeY := app.scene.Player.Position.Y + app.scene.Camera.Height
	gl.Rotatef(-app.scene.Player.Pitch, 1, 0, 0)
	gl.Rotatef(app.scene.Player.Yaw, 0, 1, 0)
	gl.Translatef(-app.scene.Player.Position.X, -eyeY, -app.scene.Player.Position.Z)

	for _, child := range app.scene.Root.Children {
		app.drawNode(gl, child)
	}
	gl.Flush()
}

func (app *sceneApp) drawNode(gl *kos.TinyGL, node sceneNode) {
	if !node.Visible {
		return
	}
	gl.PushMatrix()
	gl.Translatef(node.Position.X, node.Position.Y, node.Position.Z)
	if node.Rotation.X != 0 {
		gl.Rotatef(node.Rotation.X, 1, 0, 0)
	}
	if node.Rotation.Y != 0 {
		gl.Rotatef(node.Rotation.Y, 0, 1, 0)
	}
	if node.Rotation.Z != 0 {
		gl.Rotatef(node.Rotation.Z, 0, 0, 1)
	}
	if node.Scale.X != 1 || node.Scale.Y != 1 || node.Scale.Z != 1 {
		gl.Scalef(nonZero(node.Scale.X), nonZero(node.Scale.Y), nonZero(node.Scale.Z))
	}

	switch node.Kind {
	case "cube":
		gl.Begin(glTriangles)
		drawCube(gl, node.Size, node.Color)
		gl.End()
	case "grid":
		gl.Begin(glLines)
		drawGrid(gl, node.GridSize, node.GridStep, node.Color)
		gl.End()
	}
	for _, child := range node.Children {
		app.drawNode(gl, child)
	}
	gl.PopMatrix()
}

func (app *sceneApp) perspectiveMatrix(width int, height int) []float32 {
	if width <= 0 || height <= 0 {
		copy(app.projection[:], []float32{
			1, 0, 0, 0,
			0, 1, 0, 0,
			0, 0, -1, -1,
			0, 0, -0.2, 0,
		})
		return app.projection[:]
	}
	aspect := float32(width) / float32(height)
	fovRadians := float32(app.scene.Camera.FOV * float32(math.Pi) / 180.0)
	near := app.scene.Camera.Near
	far := app.scene.Camera.Far
	if near <= 0 {
		near = 0.2
	}
	if far <= near {
		far = near + 64
	}
	f := float32(1.0 / math.Tan(float64(fovRadians*0.5)))
	app.projection[0] = f / aspect
	app.projection[1] = 0
	app.projection[2] = 0
	app.projection[3] = 0
	app.projection[4] = 0
	app.projection[5] = f
	app.projection[6] = 0
	app.projection[7] = 0
	app.projection[8] = 0
	app.projection[9] = 0
	app.projection[10] = (far + near) / (near - far)
	app.projection[11] = -1
	app.projection[12] = 0
	app.projection[13] = 0
	app.projection[14] = (2 * far * near) / (near - far)
	app.projection[15] = 0
	return app.projection[:]
}

func colorToFloat(color rgbColor) (float32, float32, float32) {
	return float32(color.R) / 255, float32(color.G) / 255, float32(color.B) / 255
}

func drawGrid(gl *kos.TinyGL, size float32, step float32, color rgbColor) {
	if size <= 0 || step <= 0 {
		return
	}
	lineColor(gl, color)
	count := int(size / step)
	extent := float32(count) * step
	for index := -count; index <= count; index++ {
		offset := float32(index) * step
		gl.Vertex3f(-extent, 0, offset)
		gl.Vertex3f(extent, 0, offset)
		gl.Vertex3f(offset, 0, -extent)
		gl.Vertex3f(offset, 0, extent)
	}
}

func lineColor(gl *kos.TinyGL, color rgbColor) {
	gl.Color3ub(color.R, color.G, color.B)
}

func drawCube(gl *kos.TinyGL, size vec3, color rgbColor) {
	halfX := size.X * 0.5
	halfY := size.Y * 0.5
	halfZ := size.Z * 0.5
	front := color.withShade(1.00)
	back := color.withShade(0.62)
	left := color.withShade(0.74)
	right := color.withShade(0.88)
	top := color.withShade(1.12)
	bottom := color.withShade(0.56)

	face(gl, front, 0, 0, 1,
		vec3{-halfX, -halfY, halfZ},
		vec3{halfX, -halfY, halfZ},
		vec3{halfX, halfY, halfZ},
		vec3{-halfX, halfY, halfZ},
	)
	face(gl, back, 0, 0, -1,
		vec3{halfX, -halfY, -halfZ},
		vec3{-halfX, -halfY, -halfZ},
		vec3{-halfX, halfY, -halfZ},
		vec3{halfX, halfY, -halfZ},
	)
	face(gl, left, -1, 0, 0,
		vec3{-halfX, -halfY, -halfZ},
		vec3{-halfX, -halfY, halfZ},
		vec3{-halfX, halfY, halfZ},
		vec3{-halfX, halfY, -halfZ},
	)
	face(gl, right, 1, 0, 0,
		vec3{halfX, -halfY, halfZ},
		vec3{halfX, -halfY, -halfZ},
		vec3{halfX, halfY, -halfZ},
		vec3{halfX, halfY, halfZ},
	)
	face(gl, top, 0, 1, 0,
		vec3{-halfX, halfY, halfZ},
		vec3{halfX, halfY, halfZ},
		vec3{halfX, halfY, -halfZ},
		vec3{-halfX, halfY, -halfZ},
	)
	face(gl, bottom, 0, -1, 0,
		vec3{-halfX, -halfY, -halfZ},
		vec3{halfX, -halfY, -halfZ},
		vec3{halfX, -halfY, halfZ},
		vec3{-halfX, -halfY, halfZ},
	)
}

func face(gl *kos.TinyGL, color rgbColor, nx float32, ny float32, nz float32, a vec3, b vec3, c vec3, d vec3) {
	gl.Color3ub(color.R, color.G, color.B)
	gl.Normal3f(nx, ny, nz)
	gl.Vertex3f(a.X, a.Y, a.Z)
	gl.Vertex3f(b.X, b.Y, b.Z)
	gl.Vertex3f(c.X, c.Y, c.Z)
	gl.Vertex3f(a.X, a.Y, a.Z)
	gl.Vertex3f(c.X, c.Y, c.Z)
	gl.Vertex3f(d.X, d.Y, d.Z)
}
