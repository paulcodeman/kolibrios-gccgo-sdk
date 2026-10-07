package main

import (
	"context"
	"go.opentelemetry.io/auto/sdk"
	"kos"
)

func main() {
	console, ok := kos.OpenConsole("OpenCode original OTel method tables")
	if !ok {
		panic("console initialization")
	}
	defer console.Exit(false)
	tracer := sdk.TracerProvider().Tracer("native")
	ctx, span := tracer.Start(context.Background(), "original package")
	if ctx == nil || span == nil || !span.IsRecording() {
		panic("original recording span")
	}
	span.SetName("native span")
	span.End()
	if span.IsRecording() {
		panic("ended span still recording")
	}
	console.WriteString("OpenCode original OTel private receiver/interface dispatch PASS\n")
	kos.DebugString("OPENCODE_OTEL_METHOD_TABLE_PASS")
}
