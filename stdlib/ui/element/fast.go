package ui

// Fast render flags for profiling and stress tests.
// These are intended for temporary diagnostics to isolate costly features.

// FastNoShadows disables box and text shadows.
var FastNoShadows bool

// FastNoGradients disables gradient fills.
var FastNoGradients bool

// FastNoRadius disables rounded corners (treats them as zero radius).
var FastNoRadius bool

// FastNoText disables text rendering (layout is unchanged).
var FastNoText bool

// FastNoTextDraw disables the actual text output calls (layout is unchanged).
// Use for isolating syscall costs without skipping text layout logic.
var FastNoTextDraw bool

// FastNoTextCache disables cached text wrapping/layout (forces reflow each time).
var FastNoTextCache bool

// FastNoFontSmoothing disables system font smoothing (global setting).
var FastNoFontSmoothing = true

// FastNoTextShadow disables text shadow rendering.
var FastNoTextShadow bool

// FastNoBorders disables border rendering.
var FastNoBorders bool

// FastNoCache disables element render caching.
var FastNoCache bool
