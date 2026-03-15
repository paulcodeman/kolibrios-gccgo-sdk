#include "textflag.h"

TEXT ·Sleep(SB),NOSPLIT,$0-4
	MOVL	delay+0(FP), BX
	MOVL	$5, AX
	INT	$0x40
	RET

TEXT ·GetTime(SB),NOSPLIT,$0-4
	MOVL	$3, AX
	INT	$0x40
	MOVL	AX, ret+0(FP)
	RET

TEXT ·GetDate(SB),NOSPLIT,$0-4
	MOVL	$29, AX
	INT	$0x40
	MOVL	AX, ret+0(FP)
	RET

TEXT ·GetTimeCounter(SB),NOSPLIT,$0-4
	MOVL	$26, AX
	MOVL	$9, BX
	INT	$0x40
	MOVL	AX, ret+0(FP)
	RET

TEXT ·Event(SB),NOSPLIT,$0-4
	MOVL	$10, AX
	INT	$0x40
	MOVL	AX, ret+0(FP)
	RET

TEXT ·CheckEvent(SB),NOSPLIT,$0-4
	MOVL	$11, AX
	INT	$0x40
	MOVL	AX, ret+0(FP)
	RET

TEXT ·WaitEventTimeout(SB),NOSPLIT,$0-8
	MOVL	$23, AX
	MOVL	timeout+0(FP), BX
	INT	$0x40
	MOVL	AX, ret+4(FP)
	RET

TEXT ·SetEventMask(SB),NOSPLIT,$0-8
	MOVL	$40, AX
	MOVL	mask+0(FP), BX
	INT	$0x40
	MOVL	AX, ret+4(FP)
	RET

TEXT ·SetPortsRaw(SB),NOSPLIT,$0-16
	MOVL	$46, AX
	MOVL	mode+0(FP), BX
	MOVL	start+4(FP), CX
	MOVL	end+8(FP), DX
	INT	$0x40
	MOVL	AX, ret+12(FP)
	RET

TEXT ·SystemShutdown(SB),NOSPLIT,$0-8
	MOVL	$18, AX
	MOVL	$9, BX
	MOVL	mode+0(FP), CX
	INT	$0x40
	MOVL	AX, ret+4(FP)
	RET

TEXT ·FileSystemEncoded(SB),NOSPLIT,$0-12
	MOVL	$80, AX
	MOVL	request+0(FP), BX
	INT	$0x40
	MOVL	secondary+4(FP), DX
	TESTL	DX, DX
	JZ	fileSystemEncodedDone
	MOVL	BX, 0(DX)
fileSystemEncodedDone:
	MOVL	AX, ret+8(FP)
	RET

TEXT ·GetCurrentFolderRaw(SB),NOSPLIT,$0-16
	MOVL	$30, AX
	MOVL	$5, BX
	MOVL	buffer+0(FP), CX
	MOVL	size+4(FP), DX
	MOVL	encoding+8(FP), SI
	INT	$0x40
	MOVL	AX, ret+12(FP)
	RET

TEXT ·GetButtonID(SB),NOSPLIT,$0-4
	MOVL	$17, AX
	INT	$0x40
	CMPL	AX, $1
	JE	getButtonIDNone
	SHRL	$8, AX
	JMP	getButtonIDDone
getButtonIDNone:
	MOVL	$-1, AX
getButtonIDDone:
	MOVL	AX, ret+0(FP)
	RET

TEXT ·CreateButton(SB),NOSPLIT,$0-24
	MOVL	$8, AX
	MOVL	x+0(FP), BX
	SHLL	$16, BX
	ORL	width+8(FP), BX
	MOVL	y+4(FP), CX
	SHLL	$16, CX
	ORL	height+12(FP), CX
	MOVL	id+16(FP), DX
	MOVL	color+20(FP), SI
	INT	$0x40
	RET

TEXT ·ExitRaw(SB),NOSPLIT,$0-0
	MOVL	$-1, AX
	INT	$0x40
	RET

TEXT ·Redraw(SB),NOSPLIT,$0-4
	MOVL	$12, AX
	MOVL	mode+0(FP), BX
	INT	$0x40
	RET

TEXT ·windowRaw(SB),NOSPLIT,$0-20
	MOVL	x+0(FP), BX
	SHLL	$16, BX
	ORL	width+8(FP), BX
	MOVL	y+4(FP), CX
	SHLL	$16, CX
	ORL	height+12(FP), CX
	MOVL	$0x14FFFFFF, DX
	MOVL	$0x808899FF, SI
	MOVL	title+16(FP), DI
	XORL	AX, AX
	INT	$0x40
	RET

TEXT ·writeTextRaw(SB),NOSPLIT,$0-20
	MOVL	x+0(FP), BX
	SHLL	$16, BX
	MOVW	y+4(FP), BX
	MOVL	color+8(FP), CX
	ANDL	$0x00FFFFFF, CX
	ORL	$0x30000000, CX
	MOVL	text+12(FP), DX
	MOVL	textLen+16(FP), SI
	MOVL	$4, AX
	INT	$0x40
	RET

TEXT ·DrawLine(SB),NOSPLIT,$0-20
	MOVL	x1+0(FP), BX
	SHLL	$16, BX
	ORL	x2+8(FP), BX
	MOVL	y1+4(FP), CX
	SHLL	$16, CX
	ORL	y2+12(FP), CX
	MOVL	color+16(FP), DX
	MOVL	$38, AX
	INT	$0x40
	RET

TEXT ·DrawBar(SB),NOSPLIT,$0-20
	MOVL	x+0(FP), BX
	SHLL	$16, BX
	ORL	width+8(FP), BX
	MOVL	y+4(FP), CX
	SHLL	$16, CX
	ORL	height+12(FP), CX
	MOVL	color+16(FP), DX
	MOVL	$13, AX
	INT	$0x40
	RET
