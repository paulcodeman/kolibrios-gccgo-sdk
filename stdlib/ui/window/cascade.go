package ui

import "kos"

// WindowCascadeStep controls the per-window offset in pixels.
// Set to 0 to disable cascading.
var WindowCascadeStep = 20

// WindowCascadeSteps controls how many steps are used before reset.
// Set to 0 to disable reset.
var WindowCascadeSteps = 10

var windowCascadeIndex int

func nextWindowCascadeOffset(x int, y int, width int, height int) int {
	if WindowCascadeStep <= 0 {
		return 0
	}
	offset := windowCascadeIndex * WindowCascadeStep
	windowCascadeIndex++
	if WindowCascadeSteps > 0 && windowCascadeIndex > WindowCascadeSteps {
		windowCascadeIndex = 0
	}
	screenW, screenH := kos.ScreenSize()
	if screenW > 0 && screenH > 0 {
		if x+offset+width > screenW || y+offset+height > screenH {
			offset = 0
			windowCascadeIndex = 1
		}
	}
	return offset
}
