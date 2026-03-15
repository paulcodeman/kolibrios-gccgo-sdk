package kos

import (
	"math"
	"unsafe"
)

const TinyGLDLLPath = "/sys/lib/tinygl.obj"

const tinyGLContextSize = 28

const (
	tinyGLContextOffsetGLContext = 0
	tinyGLContextOffsetX         = 20
	tinyGLContextOffsetY         = 24
)

type TinyGLContext struct {
	raw [tinyGLContextSize]byte
}

type TinyGL struct {
	table              DLLExportTable
	makeCurrentProc    DLLProc
	swapBuffersProc    DLLProc
	enableProc         DLLProc
	disableProc        DLLProc
	shadeModelProc     DLLProc
	cullFaceProc       DLLProc
	polygonModeProc    DLLProc
	beginProc          DLLProc
	endProc            DLLProc
	vertex2fProc       DLLProc
	vertex3fProc       DLLProc
	vertex4fProc       DLLProc
	vertex2fvProc      DLLProc
	vertex3fvProc      DLLProc
	vertex4fvProc      DLLProc
	color3fProc        DLLProc
	color4fProc        DLLProc
	color3fvProc       DLLProc
	color4fvProc       DLLProc
	color3ubProc       DLLProc
	normal3fProc       DLLProc
	normal3fvProc      DLLProc
	texCoord1fProc     DLLProc
	texCoord2fProc     DLLProc
	texCoord3fProc     DLLProc
	texCoord4fProc     DLLProc
	texCoord1fvProc    DLLProc
	texCoord2fvProc    DLLProc
	texCoord3fvProc    DLLProc
	texCoord4fvProc    DLLProc
	edgeFlagProc       DLLProc
	matrixModeProc     DLLProc
	loadMatrixfProc    DLLProc
	loadIdentityProc   DLLProc
	multMatrixfProc    DLLProc
	pushMatrixProc     DLLProc
	popMatrixProc      DLLProc
	rotatefProc        DLLProc
	translatefProc     DLLProc
	scalefProc         DLLProc
	viewportProc       DLLProc
	genListsProc       DLLProc
	isListProc         DLLProc
	newListProc        DLLProc
	endListProc        DLLProc
	callListProc       DLLProc
	clearProc          DLLProc
	clearColorProc     DLLProc
	renderModeProc     DLLProc
	selectBufferProc   DLLProc
	initNamesProc      DLLProc
	pushNameProc       DLLProc
	popNameProc        DLLProc
	loadNameProc       DLLProc
	genTexturesProc    DLLProc
	deleteTexturesProc DLLProc
	bindTextureProc    DLLProc
	texEnviProc        DLLProc
	texParameteriProc  DLLProc
	pixelStoreiProc    DLLProc
	materialfvProc     DLLProc
	materialfProc      DLLProc
	colorMaterialProc  DLLProc
	lightfvProc        DLLProc
	lightfProc         DLLProc
	lightModeliProc    DLLProc
	lightModelfvProc   DLLProc
	flushProc          DLLProc
	hintProc           DLLProc
	getIntegervProc    DLLProc
	getFloatvProc      DLLProc
	ready              bool
}

func LoadTinyGLDLL() DLLExportTable {
	return LoadDLLFile(TinyGLDLLPath)
}

func LoadTinyGL() (TinyGL, bool) {
	return LoadTinyGLFromDLL(LoadTinyGLDLL())
}

func LoadTinyGLFromDLL(table DLLExportTable) (TinyGL, bool) {
	lib := TinyGL{
		table:              table,
		makeCurrentProc:    LookupDLLExportAny(table, "kosglMakeCurrent"),
		swapBuffersProc:    LookupDLLExportAny(table, "kosglSwapBuffers"),
		enableProc:         LookupDLLExportAny(table, "glEnable"),
		disableProc:        LookupDLLExportAny(table, "glDisable"),
		shadeModelProc:     LookupDLLExportAny(table, "glShadeModel"),
		cullFaceProc:       LookupDLLExportAny(table, "glCullFace"),
		polygonModeProc:    LookupDLLExportAny(table, "glPolygonMode"),
		beginProc:          LookupDLLExportAny(table, "glBegin"),
		endProc:            LookupDLLExportAny(table, "glEnd"),
		vertex2fProc:       LookupDLLExportAny(table, "glVertex2f"),
		vertex3fProc:       LookupDLLExportAny(table, "glVertex3f"),
		vertex4fProc:       LookupDLLExportAny(table, "glVertex4f"),
		vertex2fvProc:      LookupDLLExportAny(table, "glVertex2fv"),
		vertex3fvProc:      LookupDLLExportAny(table, "glVertex3fv"),
		vertex4fvProc:      LookupDLLExportAny(table, "glVertex4fv"),
		color3fProc:        LookupDLLExportAny(table, "glColor3f"),
		color4fProc:        LookupDLLExportAny(table, "glColor4f"),
		color3fvProc:       LookupDLLExportAny(table, "glColor3fv"),
		color4fvProc:       LookupDLLExportAny(table, "glColor4fv"),
		color3ubProc:       LookupDLLExportAny(table, "glColor3ub"),
		normal3fProc:       LookupDLLExportAny(table, "glNormal3f"),
		normal3fvProc:      LookupDLLExportAny(table, "glNormal3fv"),
		texCoord1fProc:     LookupDLLExportAny(table, "glTexCoord1f"),
		texCoord2fProc:     LookupDLLExportAny(table, "glTexCoord2f"),
		texCoord3fProc:     LookupDLLExportAny(table, "glTexCoord3f"),
		texCoord4fProc:     LookupDLLExportAny(table, "glTexCoord4f"),
		texCoord1fvProc:    LookupDLLExportAny(table, "glTexCoord1fv"),
		texCoord2fvProc:    LookupDLLExportAny(table, "glTexCoord2fv"),
		texCoord3fvProc:    LookupDLLExportAny(table, "glTexCoord3fv"),
		texCoord4fvProc:    LookupDLLExportAny(table, "glTexCoord4fv"),
		edgeFlagProc:       LookupDLLExportAny(table, "glEdgeFlag"),
		matrixModeProc:     LookupDLLExportAny(table, "glMatrixMode"),
		loadMatrixfProc:    LookupDLLExportAny(table, "glLoadMatrixf"),
		loadIdentityProc:   LookupDLLExportAny(table, "glLoadIdentity"),
		multMatrixfProc:    LookupDLLExportAny(table, "glMultMatrixf"),
		pushMatrixProc:     LookupDLLExportAny(table, "glPushMatrix"),
		popMatrixProc:      LookupDLLExportAny(table, "glPopMatrix"),
		rotatefProc:        LookupDLLExportAny(table, "glRotatef"),
		translatefProc:     LookupDLLExportAny(table, "glTranslatef"),
		scalefProc:         LookupDLLExportAny(table, "glScalef"),
		viewportProc:       LookupDLLExportAny(table, "glViewport"),
		genListsProc:       LookupDLLExportAny(table, "glGenLists"),
		isListProc:         LookupDLLExportAny(table, "glIsList"),
		newListProc:        LookupDLLExportAny(table, "glNewList"),
		endListProc:        LookupDLLExportAny(table, "glEndList"),
		callListProc:       LookupDLLExportAny(table, "glCallList"),
		clearProc:          LookupDLLExportAny(table, "glClear"),
		clearColorProc:     LookupDLLExportAny(table, "glClearColor"),
		renderModeProc:     LookupDLLExportAny(table, "glRenderMode"),
		selectBufferProc:   LookupDLLExportAny(table, "glSelectBuffer"),
		initNamesProc:      LookupDLLExportAny(table, "glInitNames"),
		pushNameProc:       LookupDLLExportAny(table, "glPushName"),
		popNameProc:        LookupDLLExportAny(table, "glPopName"),
		loadNameProc:       LookupDLLExportAny(table, "glLoadName"),
		genTexturesProc:    LookupDLLExportAny(table, "glGenTextures"),
		deleteTexturesProc: LookupDLLExportAny(table, "glDeleteTextures"),
		bindTextureProc:    LookupDLLExportAny(table, "glBindTexture"),
		texEnviProc:        LookupDLLExportAny(table, "glTexEnvi"),
		texParameteriProc:  LookupDLLExportAny(table, "glTexParameteri"),
		pixelStoreiProc:    LookupDLLExportAny(table, "glPixelStorei"),
		materialfvProc:     LookupDLLExportAny(table, "glMaterialfv"),
		materialfProc:      LookupDLLExportAny(table, "glMaterialf"),
		colorMaterialProc:  LookupDLLExportAny(table, "glColorMaterial"),
		lightfvProc:        LookupDLLExportAny(table, "glLightfv"),
		lightfProc:         LookupDLLExportAny(table, "glLightf"),
		lightModeliProc:    LookupDLLExportAny(table, "glLightModeli"),
		lightModelfvProc:   LookupDLLExportAny(table, "glLightModelfv"),
		flushProc:          LookupDLLExportAny(table, "glFlush"),
		hintProc:           LookupDLLExportAny(table, "glHint"),
		getIntegervProc:    LookupDLLExportAny(table, "glGetIntegerv"),
		getFloatvProc:      LookupDLLExportAny(table, "glGetFloatv"),
		ready:              true,
	}
	if !lib.Valid() {
		return TinyGL{}, false
	}

	initProc := LookupDLLExportAny(table, "lib_init")
	if initProc.Valid() {
		InitDLLLibraryRaw(uint32(initProc))
	}

	return lib, true
}

func (lib TinyGL) Valid() bool {
	return lib.table != 0 && lib.makeCurrentProc.Valid() && lib.swapBuffersProc.Valid()
}

func (lib TinyGL) ExportTable() DLLExportTable {
	return lib.table
}

func (lib TinyGL) Ready() bool {
	return lib.ready
}

func (lib TinyGL) MakeCurrent(x0 int, y0 int, width int, height int, ctx *TinyGLContext) bool {
	if !lib.ready || ctx == nil || !lib.makeCurrentProc.Valid() {
		return false
	}

	return CallStdcall5Raw(uint32(lib.makeCurrentProc), uint32(x0), uint32(y0), uint32(width), uint32(height), ctx.address()) != 0
}

func (lib TinyGL) SwapBuffers() bool {
	if !lib.ready || !lib.swapBuffersProc.Valid() {
		return false
	}

	CallStdcall0Raw(uint32(lib.swapBuffersProc))
	return true
}

func (lib TinyGL) Enable(capability int) bool {
	if !lib.ready || !lib.enableProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.enableProc), uint32(capability))
	return true
}

func (lib TinyGL) Disable(capability int) bool {
	if !lib.ready || !lib.disableProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.disableProc), uint32(capability))
	return true
}

func (lib TinyGL) ShadeModel(mode int) bool {
	if !lib.ready || !lib.shadeModelProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.shadeModelProc), uint32(mode))
	return true
}

func (lib TinyGL) CullFace(mode int) bool {
	if !lib.ready || !lib.cullFaceProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.cullFaceProc), uint32(mode))
	return true
}

func (lib TinyGL) PolygonMode(face int, mode int) bool {
	if !lib.ready || !lib.polygonModeProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.polygonModeProc), uint32(face), uint32(mode))
	return true
}

func (lib TinyGL) Begin(mode int) bool {
	if !lib.ready || !lib.beginProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.beginProc), uint32(mode))
	return true
}

func (lib TinyGL) End() bool {
	if !lib.ready || !lib.endProc.Valid() {
		return false
	}

	CallStdcall0Raw(uint32(lib.endProc))
	return true
}

func (lib TinyGL) Vertex2f(x float32, y float32) bool {
	if !lib.ready || !lib.vertex2fProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.vertex2fProc), float32Bits(x), float32Bits(y))
	return true
}

func (lib TinyGL) Vertex3f(x float32, y float32, z float32) bool {
	if !lib.ready || !lib.vertex3fProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.vertex3fProc), float32Bits(x), float32Bits(y), float32Bits(z))
	return true
}

func (lib TinyGL) Vertex4f(x float32, y float32, z float32, w float32) bool {
	if !lib.ready || !lib.vertex4fProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.vertex4fProc), float32Bits(x), float32Bits(y), float32Bits(z), float32Bits(w))
	return true
}

func (lib TinyGL) Vertex2fv(values []float32) bool {
	if !lib.ready || !lib.vertex2fvProc.Valid() || len(values) < 2 {
		return false
	}

	CallStdcall1Raw(uint32(lib.vertex2fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) Vertex3fv(values []float32) bool {
	if !lib.ready || !lib.vertex3fvProc.Valid() || len(values) < 3 {
		return false
	}

	CallStdcall1Raw(uint32(lib.vertex3fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) Vertex4fv(values []float32) bool {
	if !lib.ready || !lib.vertex4fvProc.Valid() || len(values) < 4 {
		return false
	}

	CallStdcall1Raw(uint32(lib.vertex4fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) Color3f(r float32, g float32, b float32) bool {
	if !lib.ready || !lib.color3fProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.color3fProc), float32Bits(r), float32Bits(g), float32Bits(b))
	return true
}

func (lib TinyGL) Color4f(r float32, g float32, b float32, a float32) bool {
	if !lib.ready || !lib.color4fProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.color4fProc), float32Bits(r), float32Bits(g), float32Bits(b), float32Bits(a))
	return true
}

func (lib TinyGL) Color3fv(values []float32) bool {
	if !lib.ready || !lib.color3fvProc.Valid() || len(values) < 3 {
		return false
	}

	CallStdcall1Raw(uint32(lib.color3fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) Color4fv(values []float32) bool {
	if !lib.ready || !lib.color4fvProc.Valid() || len(values) < 4 {
		return false
	}

	CallStdcall1Raw(uint32(lib.color4fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) Color3ub(r uint8, g uint8, b uint8) bool {
	if !lib.ready || !lib.color3ubProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.color3ubProc), uint32(r), uint32(g), uint32(b))
	return true
}

func (lib TinyGL) Normal3f(x float32, y float32, z float32) bool {
	if !lib.ready || !lib.normal3fProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.normal3fProc), float32Bits(x), float32Bits(y), float32Bits(z))
	return true
}

func (lib TinyGL) Normal3fv(values []float32) bool {
	if !lib.ready || !lib.normal3fvProc.Valid() || len(values) < 3 {
		return false
	}

	CallStdcall1Raw(uint32(lib.normal3fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) TexCoord1f(s float32) bool {
	if !lib.ready || !lib.texCoord1fProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.texCoord1fProc), float32Bits(s))
	return true
}

func (lib TinyGL) TexCoord2f(s float32, t float32) bool {
	if !lib.ready || !lib.texCoord2fProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.texCoord2fProc), float32Bits(s), float32Bits(t))
	return true
}

func (lib TinyGL) TexCoord3f(s float32, t float32, r float32) bool {
	if !lib.ready || !lib.texCoord3fProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.texCoord3fProc), float32Bits(s), float32Bits(t), float32Bits(r))
	return true
}

func (lib TinyGL) TexCoord4f(s float32, t float32, r float32, q float32) bool {
	if !lib.ready || !lib.texCoord4fProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.texCoord4fProc), float32Bits(s), float32Bits(t), float32Bits(r), float32Bits(q))
	return true
}

func (lib TinyGL) TexCoord1fv(values []float32) bool {
	if !lib.ready || !lib.texCoord1fvProc.Valid() || len(values) < 1 {
		return false
	}

	CallStdcall1Raw(uint32(lib.texCoord1fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) TexCoord2fv(values []float32) bool {
	if !lib.ready || !lib.texCoord2fvProc.Valid() || len(values) < 2 {
		return false
	}

	CallStdcall1Raw(uint32(lib.texCoord2fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) TexCoord3fv(values []float32) bool {
	if !lib.ready || !lib.texCoord3fvProc.Valid() || len(values) < 3 {
		return false
	}

	CallStdcall1Raw(uint32(lib.texCoord3fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) TexCoord4fv(values []float32) bool {
	if !lib.ready || !lib.texCoord4fvProc.Valid() || len(values) < 4 {
		return false
	}

	CallStdcall1Raw(uint32(lib.texCoord4fvProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) EdgeFlag(flag bool) bool {
	if !lib.ready || !lib.edgeFlagProc.Valid() {
		return false
	}

	value := uint32(0)
	if flag {
		value = 1
	}
	CallStdcall1Raw(uint32(lib.edgeFlagProc), value)
	return true
}

func (lib TinyGL) MatrixMode(mode int) bool {
	if !lib.ready || !lib.matrixModeProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.matrixModeProc), uint32(mode))
	return true
}

func (lib TinyGL) LoadMatrix(values []float32) bool {
	if !lib.ready || !lib.loadMatrixfProc.Valid() || len(values) < 16 {
		return false
	}

	CallStdcall1Raw(uint32(lib.loadMatrixfProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) LoadIdentity() bool {
	if !lib.ready || !lib.loadIdentityProc.Valid() {
		return false
	}

	CallStdcall0Raw(uint32(lib.loadIdentityProc))
	return true
}

func (lib TinyGL) MultMatrix(values []float32) bool {
	if !lib.ready || !lib.multMatrixfProc.Valid() || len(values) < 16 {
		return false
	}

	CallStdcall1Raw(uint32(lib.multMatrixfProc), float32SliceAddress(values))
	return true
}

func (lib TinyGL) PushMatrix() bool {
	if !lib.ready || !lib.pushMatrixProc.Valid() {
		return false
	}

	CallStdcall0Raw(uint32(lib.pushMatrixProc))
	return true
}

func (lib TinyGL) PopMatrix() bool {
	if !lib.ready || !lib.popMatrixProc.Valid() {
		return false
	}

	CallStdcall0Raw(uint32(lib.popMatrixProc))
	return true
}

func (lib TinyGL) Rotatef(angle float32, x float32, y float32, z float32) bool {
	if !lib.ready || !lib.rotatefProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.rotatefProc), float32Bits(angle), float32Bits(x), float32Bits(y), float32Bits(z))
	return true
}

func (lib TinyGL) Translatef(x float32, y float32, z float32) bool {
	if !lib.ready || !lib.translatefProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.translatefProc), float32Bits(x), float32Bits(y), float32Bits(z))
	return true
}

func (lib TinyGL) Scalef(x float32, y float32, z float32) bool {
	if !lib.ready || !lib.scalefProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.scalefProc), float32Bits(x), float32Bits(y), float32Bits(z))
	return true
}

func (lib TinyGL) Viewport(x int, y int, width int, height int) bool {
	if !lib.ready || !lib.viewportProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.viewportProc), uint32(x), uint32(y), uint32(width), uint32(height))
	return true
}

func (lib TinyGL) GenLists(rangeCount int) uint32 {
	if !lib.ready || !lib.genListsProc.Valid() {
		return 0
	}

	return CallStdcall1Raw(uint32(lib.genListsProc), uint32(rangeCount))
}

func (lib TinyGL) IsList(listID uint32) bool {
	if !lib.ready || !lib.isListProc.Valid() {
		return false
	}

	return CallStdcall1Raw(uint32(lib.isListProc), listID) != 0
}

func (lib TinyGL) NewList(listID uint32, mode int) bool {
	if !lib.ready || !lib.newListProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.newListProc), listID, uint32(mode))
	return true
}

func (lib TinyGL) EndList() bool {
	if !lib.ready || !lib.endListProc.Valid() {
		return false
	}

	CallStdcall0Raw(uint32(lib.endListProc))
	return true
}

func (lib TinyGL) CallList(listID uint32) bool {
	if !lib.ready || !lib.callListProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.callListProc), listID)
	return true
}

func (lib TinyGL) Clear(mask int) bool {
	if !lib.ready || !lib.clearProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.clearProc), uint32(mask))
	return true
}

func (lib TinyGL) ClearColor(r float32, g float32, b float32, a float32) bool {
	if !lib.ready || !lib.clearColorProc.Valid() {
		return false
	}

	CallStdcall4Raw(uint32(lib.clearColorProc), float32Bits(r), float32Bits(g), float32Bits(b), float32Bits(a))
	return true
}

func (lib TinyGL) RenderMode(mode int) int {
	if !lib.ready || !lib.renderModeProc.Valid() {
		return 0
	}

	return int(int32(CallStdcall1Raw(uint32(lib.renderModeProc), uint32(mode))))
}

func (lib TinyGL) SelectBuffer(buffer []uint32) bool {
	if !lib.ready || !lib.selectBufferProc.Valid() || len(buffer) == 0 {
		return false
	}

	CallStdcall2Raw(uint32(lib.selectBufferProc), uint32(len(buffer)), uint32SliceAddress(buffer))
	return true
}

func (lib TinyGL) InitNames() bool {
	if !lib.ready || !lib.initNamesProc.Valid() {
		return false
	}

	CallStdcall0Raw(uint32(lib.initNamesProc))
	return true
}

func (lib TinyGL) PushName(name uint32) bool {
	if !lib.ready || !lib.pushNameProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.pushNameProc), name)
	return true
}

func (lib TinyGL) PopName() bool {
	if !lib.ready || !lib.popNameProc.Valid() {
		return false
	}

	CallStdcall0Raw(uint32(lib.popNameProc))
	return true
}

func (lib TinyGL) LoadName(name uint32) bool {
	if !lib.ready || !lib.loadNameProc.Valid() {
		return false
	}

	CallStdcall1Raw(uint32(lib.loadNameProc), name)
	return true
}

func (lib TinyGL) GenTextures(textures []uint32) bool {
	if !lib.ready || !lib.genTexturesProc.Valid() || len(textures) == 0 {
		return false
	}

	CallStdcall2Raw(uint32(lib.genTexturesProc), uint32(len(textures)), uint32SliceAddress(textures))
	return true
}

func (lib TinyGL) DeleteTextures(textures []uint32) bool {
	if !lib.ready || !lib.deleteTexturesProc.Valid() || len(textures) == 0 {
		return false
	}

	CallStdcall2Raw(uint32(lib.deleteTexturesProc), uint32(len(textures)), uint32SliceAddress(textures))
	return true
}

func (lib TinyGL) BindTexture(target int, texture uint32) bool {
	if !lib.ready || !lib.bindTextureProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.bindTextureProc), uint32(target), texture)
	return true
}

func (lib TinyGL) TexEnvi(target int, pname int, param int) bool {
	if !lib.ready || !lib.texEnviProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.texEnviProc), uint32(target), uint32(pname), uint32(param))
	return true
}

func (lib TinyGL) TexParameteri(target int, pname int, param int) bool {
	if !lib.ready || !lib.texParameteriProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.texParameteriProc), uint32(target), uint32(pname), uint32(param))
	return true
}

func (lib TinyGL) PixelStorei(pname int, param int) bool {
	if !lib.ready || !lib.pixelStoreiProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.pixelStoreiProc), uint32(pname), uint32(param))
	return true
}

func (lib TinyGL) Materialfv(face int, pname int, params []float32) bool {
	if !lib.ready || !lib.materialfvProc.Valid() || len(params) == 0 {
		return false
	}

	CallStdcall3Raw(uint32(lib.materialfvProc), uint32(face), uint32(pname), float32SliceAddress(params))
	return true
}

func (lib TinyGL) Materialf(face int, pname int, param float32) bool {
	if !lib.ready || !lib.materialfProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.materialfProc), uint32(face), uint32(pname), float32Bits(param))
	return true
}

func (lib TinyGL) ColorMaterial(face int, mode int) bool {
	if !lib.ready || !lib.colorMaterialProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.colorMaterialProc), uint32(face), uint32(mode))
	return true
}

func (lib TinyGL) Lightfv(light int, pname int, params []float32) bool {
	if !lib.ready || !lib.lightfvProc.Valid() || len(params) == 0 {
		return false
	}

	CallStdcall3Raw(uint32(lib.lightfvProc), uint32(light), uint32(pname), float32SliceAddress(params))
	return true
}

func (lib TinyGL) Lightf(light int, pname int, param float32) bool {
	if !lib.ready || !lib.lightfProc.Valid() {
		return false
	}

	CallStdcall3Raw(uint32(lib.lightfProc), uint32(light), uint32(pname), float32Bits(param))
	return true
}

func (lib TinyGL) LightModeli(pname int, param int) bool {
	if !lib.ready || !lib.lightModeliProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.lightModeliProc), uint32(pname), uint32(param))
	return true
}

func (lib TinyGL) LightModelfv(pname int, params []float32) bool {
	if !lib.ready || !lib.lightModelfvProc.Valid() || len(params) == 0 {
		return false
	}

	CallStdcall2Raw(uint32(lib.lightModelfvProc), uint32(pname), float32SliceAddress(params))
	return true
}

func (lib TinyGL) Flush() bool {
	if !lib.ready || !lib.flushProc.Valid() {
		return false
	}

	CallStdcall0Raw(uint32(lib.flushProc))
	return true
}

func (lib TinyGL) Hint(target int, mode int) bool {
	if !lib.ready || !lib.hintProc.Valid() {
		return false
	}

	CallStdcall2Raw(uint32(lib.hintProc), uint32(target), uint32(mode))
	return true
}

func (lib TinyGL) GetIntegerv(pname int, values []int32) bool {
	if !lib.ready || !lib.getIntegervProc.Valid() || len(values) == 0 {
		return false
	}

	CallStdcall2Raw(uint32(lib.getIntegervProc), uint32(pname), int32SliceAddress(values))
	return true
}

func (lib TinyGL) GetFloatv(pname int, values []float32) bool {
	if !lib.ready || !lib.getFloatvProc.Valid() || len(values) == 0 {
		return false
	}

	CallStdcall2Raw(uint32(lib.getFloatvProc), uint32(pname), float32SliceAddress(values))
	return true
}

func (ctx *TinyGLContext) address() uint32 {
	if ctx == nil {
		return 0
	}

	return pointerValue(&ctx.raw[0])
}

func (ctx *TinyGLContext) Initialized() bool {
	if ctx == nil {
		return false
	}
	return littleEndianUint32(ctx.raw[:], tinyGLContextOffsetGLContext) != 0
}

func (ctx *TinyGLContext) Reset() {
	if ctx == nil {
		return
	}
	for i := range ctx.raw {
		ctx.raw[i] = 0
	}
}

func (ctx *TinyGLContext) SetPosition(x int, y int) {
	if ctx == nil {
		return
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	putUint32LE(ctx.raw[:], tinyGLContextOffsetX, uint32(x))
	putUint32LE(ctx.raw[:], tinyGLContextOffsetY, uint32(y))
}

func float32Bits(value float32) uint32 {
	return math.Float32bits(value)
}

func float32SliceAddress(values []float32) uint32 {
	if len(values) == 0 {
		return 0
	}

	return pointerValue((*byte)(unsafe.Pointer(&values[0])))
}

func int32SliceAddress(values []int32) uint32 {
	if len(values) == 0 {
		return 0
	}

	return pointerValue((*byte)(unsafe.Pointer(&values[0])))
}

func uint32SliceAddress(values []uint32) uint32 {
	if len(values) == 0 {
		return 0
	}

	return pointerValue((*byte)(unsafe.Pointer(&values[0])))
}
