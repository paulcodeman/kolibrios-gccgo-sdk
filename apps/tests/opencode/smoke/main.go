// This exercises OpenCode's unchanged LSP protocol package, not the full CLI.
package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"math"
	"reflect"
	"regression/consumer"
	"regression/owner"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/opencode-ai/opencode/internal/lsp/protocol"
	"github.com/rivo/uniseg"
	"kos"
)

//go:embed resources/*.txt
var embeddedResources embed.FS

//go:embed "resources/space file.txt"
var embeddedText string

func main() {
	kos.DebugString("OPENCODE_SMOKE_START")
	console, ok := kos.OpenConsole("OpenCode port smoke test")
	if !ok {
		kos.DebugString("opencode-smoke: console unavailable")
		return
	}
	defer console.Close()
	embedded, embeddedErr := embeddedResources.ReadFile("resources/space file.txt")
	entries, directoryErr := embeddedResources.ReadDir("resources")
	if embeddedErr != nil || directoryErr != nil || len(entries) != 1 || string(embedded) != "original embedded resource\n" || embeddedText != string(embedded) {
		panic("original embed resources or directory traversal failed")
	}
	kos.DebugString("EMBED_OK")
	if owner.Buffer(1).String() != "buffer" {
		panic("generic zero literal of an imported struct failed")
	}
	embeddedBox := consumer.NewEmbedded()
	var embeddedMethods interface{ Get() int } = embeddedBox
	if embeddedBox.Get() != 19 || embeddedBox.Box.Get() != 19 || embeddedMethods.Get() != 19 || consumer.NewEmbeddedValue().Get() != 23 {
		panic("embedded generic field or method set failed")
	}
	embeddedField := reflect.TypeOf(embeddedBox).Field(0)
	if embeddedField.Name != "Box" || !embeddedField.Anonymous || embeddedField.PkgPath != "" || embeddedField.Tag.Get("json") != "embedded" {
		panic("embedded generic reflection failed")
	}
	genericEmbedded := owner.MakeEmbedded(29)
	privateEmbedded := owner.MakePrivateEmbedded(31)
	if genericEmbedded.Get() != 29 || genericEmbedded.Box.Get() != 29 || privateEmbedded.Get() != 31 {
		panic("generic or private embedded field promotion failed")
	}
	privateEmbeddedField := reflect.TypeOf(privateEmbedded).Field(0)
	if privateEmbeddedField.Name != "privateBox" || !privateEmbeddedField.Anonymous || privateEmbeddedField.PkgPath != "regression/owner" {
		panic("private embedded generic visibility failed")
	}
	kos.DebugString("GENERIC_EMBEDDING_OK")
	sliceBacking := []int{10, 20, 30, 40}
	reflectedSlice := reflect.ValueOf(sliceBacking).Slice(1, 3)
	if reflectedSlice.Len() != 2 || reflectedSlice.Cap() != 3 {
		panic("reflected slice bounds changed")
	}
	reflectedSlice.Index(0).SetInt(25)
	if sliceBacking[1] != 25 || reflect.ValueOf(sliceBacking).Slice3(1, 2, 3).Cap() != 2 {
		panic("reflected slice backing storage or three-index capacity changed")
	}
	arrayBacking := [3]int{3, 4, 5}
	reflectedArray := reflect.ValueOf(&arrayBacking).Elem()
	reflectedArray.Slice(1, 3).Index(0).SetInt(7)
	if reflectedArray.Cap() != 3 || arrayBacking[1] != 7 || reflect.ValueOf("port").Slice(1, 3).String() != "or" {
		panic("reflected array or string slicing failed")
	}
	reflectedChannel := make(chan int, 3)
	reflectedChannel <- 1
	if reflect.ValueOf(reflectedChannel).Len() != 1 || reflect.ValueOf(reflectedChannel).Cap() != 3 || reflect.ValueOf((chan int)(nil)).Len() != 0 {
		panic("reflected channel length or capacity failed")
	}
	if !reflect.ValueOf(float32(0)).OverflowFloat(math.MaxFloat64) || reflect.ValueOf(float32(0)).OverflowFloat(math.Inf(1)) {
		panic("reflected floating-point overflow check failed")
	}
	privateSlice := reflect.ValueOf(struct{ items []int }{[]int{1, 2}}).Field(0).Slice(0, 1)
	if privateSlice.CanInterface() || privateSlice.Index(0).CanSet() {
		panic("slicing removed private-field restrictions")
	}
	for _, invalid := range []func(){
		func() { reflect.ValueOf(2).Slice(0, 1) },
		func() { reflect.ValueOf([1]int{}).Slice(0, 1) },
		func() { reflect.ValueOf(sliceBacking).Slice(-1, 2) },
		func() { reflect.ValueOf(sliceBacking).Slice3(0, 3, 2) },
		func() { reflect.ValueOf("port").Slice3(0, 1, 2) },
		func() { reflect.ValueOf("port").Slice(1, 10) },
		func() { reflect.ValueOf(2).Cap() },
		func() { reflect.ValueOf(2).OverflowFloat(0) },
	} {
		if !func() (panicked bool) {
			defer func() { panicked = recover() != nil }()
			invalid()
			return false
		}() {
			panic("invalid reflection slicing or kind did not panic")
		}
	}
	kos.DebugString("REFLECT_SLICE_OK")
	var splitValues []string
	strings.SplitSeq("a,b,c", ",")(func(value string) bool {
		splitValues = append(splitValues, value)
		return len(splitValues) < 2
	})
	if len(splitValues) != 2 || splitValues[0] != "a" || splitValues[1] != "b" {
		panic("original Go 1.24 SplitSeq or early stop failed")
	}
	var runeValues []string
	strings.SplitSeq("é界", "")(func(value string) bool {
		runeValues = append(runeValues, value)
		return true
	})
	if len(runeValues) != 2 || runeValues[0] != "é" || runeValues[1] != "界" {
		panic("original Go 1.24 SplitSeq Unicode splitting failed")
	}
	var lineValues []string
	strings.Lines("one\ntwo")(func(value string) bool {
		lineValues = append(lineValues, value)
		return true
	})
	if len(lineValues) != 2 || lineValues[0] != "one\n" || lineValues[1] != "two" {
		panic("original Go 1.24 Lines failed")
	}
	kos.DebugString("STRINGS_ITERATOR_OK")
	started, cleaned := false, false
	next, stop := iter.Pull(func(yield func(int) bool) {
		started = true
		defer func() { cleaned = true }()
		for _, value := range []int{4, 5, 6} {
			if !yield(value) {
				return
			}
		}
	})
	if started {
		panic("pull iterator ran before first next")
	}
	for _, expected := range []int{4, 5, 6} {
		value, ok := next()
		if !ok || value != expected {
			panic("pull iterator lost a value")
		}
	}
	if value, ok := next(); ok || value != 0 || !cleaned {
		panic("pull iterator exhaustion or cleanup failed")
	}
	stop()
	stop()
	if value, ok := next(); ok || value != 0 {
		panic("pull iterator restarted after exhaustion")
	}
	cleaned = false
	next, stop = iter.Pull(func(yield func(int) bool) {
		defer func() { cleaned = true }()
		if yield(9) {
			panic("stopped pull iterator continued")
		}
	})
	if value, ok := next(); value != 9 || !ok {
		panic("early stop iterator lost its first value")
	}
	stop()
	if !cleaned {
		panic("stopped pull iterator did not run defers")
	}
	started = false
	next, stop = iter.Pull(func(yield func(int) bool) { started = true })
	stop()
	if value, ok := next(); started || ok || value != 0 {
		panic("stop before next ran the sequence")
	}
	nextPair, stopPair := iter.Pull2(func(yield func(string, int) bool) {
		for _, value := range []int{7, 8} {
			if !yield("pair", value) {
				return
			}
		}
	})
	for _, expected := range []int{7, 8} {
		key, value, ok := nextPair()
		if !ok || key != "pair" || value != expected {
			panic("pull pair iterator failed")
		}
	}
	if key, value, ok := nextPair(); ok || key != "" || value != 0 {
		panic("pull pair iterator did not exhaust")
	}
	stopPair()
	panicValue := errors.New("iterator panic")
	next, stop = iter.Pull(func(yield func(int) bool) { panic(panicValue) })
	if caught := func() (caught any) {
		defer func() { caught = recover() }()
		next()
		return nil
	}(); caught != panicValue {
		panic("pull iterator did not propagate its panic")
	}
	stop()
	iteratorExited := make(chan bool, 1)
	go func() {
		defer func() { iteratorExited <- recover() == nil }()
		goexitNext, _ := iter.Pull(func(yield func(int) bool) { runtime.Goexit() })
		goexitNext()
		iteratorExited <- false
	}()
	if !<-iteratorExited {
		panic("pull iterator did not propagate Goexit")
	}
	runtime.LockOSThread()
	next, stop = iter.Pull(func(yield func(int) bool) { yield(12) })
	if value, ok := next(); !ok || value != 12 {
		panic("thread-locked pull iterator failed")
	}
	stop()
	runtime.UnlockOSThread()
	nestedNext, nestedStop := iter.Pull(func(yield func(int) bool) {
		innerNext, innerStop := iter.Pull(func(innerYield func(int) bool) {
			for _, value := range []int{3, 5} {
				if !innerYield(value) {
					return
				}
			}
		})
		defer innerStop()
		for {
			value, ok := innerNext()
			if !ok || !yield(value*2) {
				return
			}
		}
	})
	for _, expected := range []int{6, 10} {
		value, ok := nestedNext()
		if !ok || value != expected {
			panic("nested iterator coroutine exchange failed")
		}
	}
	nestedStop()
	stopPanic := errors.New("iterator panic during stop")
	next, stop = iter.Pull(func(yield func(int) bool) {
		yield(1)
		panic(stopPanic)
	})
	next()
	if caught := func() (caught any) {
		defer func() { caught = recover() }()
		stop()
		return nil
	}(); caught != stopPanic {
		panic("pull stop did not propagate a panic")
	}
	kos.DebugString("ITER_PULL_OK")
	formatTime := time.Date(2024, time.February, 29, 13, 14, 15, 123400000, time.FixedZone("EET", 2*60*60))
	for _, check := range []struct{ layout, expected string }{
		{time.RFC1123Z, "Thu, 29 Feb 2024 13:14:15 +0200"},
		{time.RFC3339, "2024-02-29T13:14:15+02:00"},
		{time.RFC3339Nano, "2024-02-29T13:14:15.1234+02:00"},
		{time.RFC822, "29 Feb 24 13:14 EET"},
		{time.ANSIC, "Thu Feb 29 13:14:15 2024"},
		{time.DateOnly, "2024-02-29"},
		{"2006-002", "2024-060"},
		{"Monday January 2 3:04PM", "Thursday February 29 1:14PM"},
	} {
		if actual := formatTime.Format(check.layout); actual != check.expected || string(formatTime.AppendFormat([]byte("prefix:"), check.layout)) != "prefix:"+check.expected {
			panic("original time layout formatting failed: " + actual)
		}
	}
	for _, check := range []struct{ layout, value string }{
		{time.RFC1123Z, "Thu, 29 Feb 2024 13:14:15 +0200"},
		{time.RFC3339Nano, "2024-02-29T13:14:15.1234+02:00"},
	} {
		parsed, parseErr := time.Parse(check.layout, check.value)
		if parseErr != nil || parsed.Unix() != formatTime.Unix() || parsed.YearDay() != 60 {
			panic("original time parsing failed")
		}
	}
	if parsed, parseErr := time.Parse(time.RFC3339Nano, "2024-02-29T13:14:15.1234+02:00"); parseErr != nil || parsed.Nanosecond() != 123400000 {
		panic("original fractional time parsing failed")
	}
	if parsed, parseErr := time.ParseInLocation(time.DateTime, "2024-02-29 13:14:15", time.FixedZone("EET", 2*60*60)); parseErr != nil || parsed.Unix() != formatTime.Unix() {
		panic("original time parsing in a fixed location failed")
	}
	if _, parseErr := time.Parse(time.DateOnly, "2023-02-29"); parseErr == nil {
		panic("invalid calendar date accepted")
	}
	if duration, parseErr := time.ParseDuration("-2562047h47m16.854775808s"); parseErr != nil || duration != time.Duration(-1<<63) {
		panic("original duration parser rejected its negative bound")
	}
	if _, parseErr := time.ParseDuration("2562047h47m16.854775808s"); parseErr == nil {
		panic("duration parser overflow accepted")
	}
	kos.DebugString("TIME_FORMAT_OK")
	var crossPackageBox owner.Box[int] = consumer.New().Box
	dynamicBox, dynamicOK := consumer.Interface().(owner.Box[int])
	if crossPackageBox.Get() != 7 || !dynamicOK || dynamicBox.Get() != 11 || owner.Touch(5) != 7 || owner.Stored() != 6 {
		panic("generic package identity or private state failed")
	}
	boxType := reflect.TypeOf(dynamicBox)
	if boxType.Name() != "Box[int]" || boxType.PkgPath() != "regression/owner" {
		panic("generic reflection identity failed")
	}
	kos.DebugString("GENERIC_IDENTITY_OK")
	backingText := []byte{'p', 'o', 'r', 't'}
	aliasedText := unsafe.String(unsafe.SliceData(backingText), len(backingText))
	if aliasedText != "port" || unsafe.StringData(aliasedText) != &backingText[0] {
		panic("unsafe string backing storage failed")
	}
	kos.DebugString("UNSAFE_STRING_OK")
	if time.Tick(0) != nil || time.Tick(-time.Second) != nil {
		panic("non-positive Tick must return nil")
	}
	ticker := time.NewTicker(30 * time.Millisecond)
	select {
	case <-ticker.C:
	case <-time.After(time.Second):
		panic("ticker did not fire")
	}
	time.Sleep(100 * time.Millisecond)
	if len(ticker.C) != 1 {
		panic("ticker did not buffer and drop slow receiver ticks")
	}
	ticker.Stop()
	<-ticker.C
	time.Sleep(60 * time.Millisecond)
	select {
	case <-ticker.C:
		panic("stopped ticker fired or closed its channel")
	default:
	}
	ticker.Reset(40 * time.Millisecond)
	select {
	case <-ticker.C:
	case <-time.After(time.Second):
		panic("ticker Reset did not restart")
	}
	ticker.Stop()
	if func() (panicked bool) {
		defer func() { panicked = recover() != nil }()
		time.NewTicker(0)
		return false
	}() == false {
		panic("NewTicker accepted a non-positive period")
	}
	kos.DebugString("TICKER_OK")
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	logger.Info("port", slog.String("unicode", "e\u0301"), slog.Int("count", 2))
	kos.DebugString("SLOG_JSON=" + logOutput.String())
	var event map[string]any
	logDecodeErr := json.Unmarshal(logOutput.Bytes(), &event)
	if logDecodeErr != nil || event["msg"] != "port" || event["unicode"] != "e\u0301" {
		panic("upstream slog JSON handler failed")
	}
	kos.DebugString("SLOG_DECODE_OK")
	type jsonKey string
	var namedKeys map[jsonKey]int
	if err := json.Unmarshal([]byte(`{"count":2}`), &namedKeys); err != nil || namedKeys["count"] != 2 {
		panic("JSON named map keys failed")
	}
	runes := reflect.ValueOf("e\u0301").Convert(reflect.TypeOf([]rune(nil)))
	if runes.Len() != 2 || runes.Convert(reflect.TypeOf("")).String() != "e\u0301" || reflect.ValueOf(int64(257)).Convert(reflect.TypeOf(uint8(0))).Uint() != 1 {
		panic("reflect conversion rules failed")
	}
	sourceArray := []int{1, 2}
	arrayValue := reflect.ValueOf(sourceArray).Convert(reflect.TypeOf([2]int{}))
	sourceArray[0] = 9
	if arrayValue.Index(0).Int() != 1 || reflect.ValueOf(sourceArray[:1]).CanConvert(reflect.TypeOf([2]int{})) {
		panic("reflect slice-to-array copy failed")
	}
	kos.DebugString("REFLECT_CONVERSION_OK")
	originalMap := map[string]int{"one": 1, "two": 2}
	clonedMap := maps.Clone(originalMap)
	clonedMap["one"] = 9
	if originalMap["one"] != 1 || clonedMap["one"] != 9 || !maps.Equal(originalMap, map[string]int{"one": 1, "two": 2}) || maps.Clone(map[string]int(nil)) != nil {
		panic("upstream map cloning failed")
	}
	nanKeys := make(map[float64]int)
	nanKeys[math.NaN()] = 1
	nanKeys[math.NaN()] = 2
	nanCopy := maps.Clone(nanKeys)
	if len(nanCopy) != 2 {
		panic("map clone lost non-reflexive keys")
	}
	kos.DebugString("MAP_CLONE_OK")
	cause := errors.New("provider request canceled")
	parent, cancelParent := context.WithCancelCause(context.Background())
	child, cancelChild := context.WithCancelCause(parent)
	cancelParent(cause)
	cancelChild(errors.New("later cancellation"))
	if child.Err() != context.Canceled || context.Cause(child) != cause {
		panic("context cancellation cause propagation failed")
	}
	kos.DebugString("CONTEXT_CAUSE_OK")
	if context.WithoutCancel(child).Done() != nil || context.Cause(context.WithoutCancel(child)) != nil {
		panic("context WithoutCancel failed")
	}
	callbackDone := make(chan bool, 1)
	callbackCtx, cancelCallback := context.WithCancel(context.Background())
	stopCallback := context.AfterFunc(callbackCtx, func() { callbackDone <- true })
	cancelCallback()
	if stopCallback() {
		panic("context AfterFunc did not start")
	}
	select {
	case <-callbackDone:
	case <-time.After(time.Second):
		panic("context AfterFunc failed")
	}
	kos.DebugString("CONTEXT_CALLBACK_OK")
	exited := make(chan bool, 1)
	go func() {
		defer func() { exited <- recover() == nil }()
		runtime.Goexit()
		panic("Goexit returned")
	}()
	select {
	case nilRecover := <-exited:
		if !nilRecover {
			panic("Goexit was recoverable")
		}
	case <-time.After(time.Second):
		panic("Goexit skipped defer")
	}
	kos.DebugString("GOEXIT_OK")
	if uniseg.GraphemeClusterCount("e\u0301👨‍👩‍👧‍👦") != 2 || uniseg.StringWidth("👨‍👩‍👧‍👦") != 2 {
		panic("upstream generic Unicode segmentation failed")
	}
	graphemes := uniseg.NewGraphemes("e\u0301x")
	if !graphemes.Next() || graphemes.Str() != "e\u0301" || !graphemes.Next() || graphemes.Str() != "x" || graphemes.Next() {
		panic("upstream Unicode iterator methods failed")
	}
	backing := []int{1, 2, 3}
	clear(backing[:2])
	if backing[0] != 0 || backing[1] != 0 || backing[2] != 3 {
		panic("slice clear failed")
	}
	values := map[string]int{"one": 1, "two": 2}
	clear(values)
	if len(values) != 0 {
		panic("map clear failed")
	}
	callbacks := make([]func() int, 0)
	for i := range 3 {
		callbacks = append(callbacks, func() int { return i })
	}
	if callbacks[0]() != 0 || callbacks[2]() != 2 {
		panic("integer range closure failed")
	}
	const minimum = min(1, 1000)
	var small int8 = minimum
	if small != 1 || min("b", "a") != "a" || max(3, 9, 1) != 9 {
		panic("min/max constant failed")
	}
	x, y := 4, -7
	if min(x, y) != y || max(x, y) != x {
		panic("min/max integer failed")
	}
	positive, negative := float64(0), math.Copysign(0, -1)
	if !math.Signbit(min(positive, negative)) || math.Signbit(max(negative, positive)) {
		panic("min/max signed zero failed")
	}
	nan := math.NaN()
	if !math.IsNaN(min(1, nan)) || !math.IsNaN(max(nan, 1)) {
		panic("min/max NaN failed")
	}
	uri, err := protocol.ParseDocumentUri("file:///sys/settings/opencode.json")
	if err != nil || uri.Path() != "/sys/settings/opencode.json" {
		panic("OpenCode DocumentUri round trip failed")
	}
	escapedURI, err := protocol.ParseDocumentUri("file:///sys/My%20File.go")
	if err != nil || escapedURI.Path() != "/sys/My File.go" {
		panic("OpenCode escaped URI failed")
	}
	position := protocol.Position{Line: 7, Character: 12}
	data, err := json.Marshal(position)
	if err != nil {
		panic(err)
	}
	var decoded protocol.Position
	if err := json.Unmarshal(data, &decoded); err != nil {
		panic(err)
	}
	if decoded != position {
		panic("OpenCode Position JSON round trip failed")
	}
	fmt.Println("PASS: upstream OpenCode LSP URI and JSON")
	kos.DebugString("OPENCODE_SMOKE_PASS")
	kos.SleepSeconds(15)
}
