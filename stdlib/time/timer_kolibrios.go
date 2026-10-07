// Copyright 2026 The KolibriOS gccgo SDK Authors.
// Use of this source code is governed by a BSD-style license.

package time

import (
	"kos"
	"sync"
)

// runtimeTimer is the libgo timer interface backed by the cooperative
// KolibriOS scheduler. Runtime-internal heap fields are replaced by a mutex
// and a generation counter; public Timer and Ticker algorithms are unchanged.
type runtimeTimer struct {
	when   int64
	period int64
	f      func(any, uintptr)
	arg    any
	seq    uintptr

	mu         sync.Mutex
	active     bool
	generation uint64
}

func runtimeNano() int64 { return int64(kos.UptimeNanoseconds()) }

func startTimer(t *runtimeTimer) {
	t.mu.Lock()
	t.active = true
	t.generation++
	generation := t.generation
	t.mu.Unlock()
	go runTimer(t, generation)
}

func stopTimer(t *runtimeTimer) bool {
	t.mu.Lock()
	active := t.active
	t.active = false
	t.generation++
	t.mu.Unlock()
	return active
}

func resetTimer(t *runtimeTimer, deadline int64) bool {
	t.mu.Lock()
	active := t.active
	t.when = deadline
	t.active = true
	t.generation++
	generation := t.generation
	t.mu.Unlock()
	go runTimer(t, generation)
	return active
}

func modTimer(t *runtimeTimer, deadline, period int64, f func(any, uintptr), arg any, seq uintptr) {
	t.mu.Lock()
	t.when, t.period, t.f, t.arg, t.seq = deadline, period, f, arg, seq
	t.active = true
	t.generation++
	generation := t.generation
	t.mu.Unlock()
	go runTimer(t, generation)
}

func runTimer(t *runtimeTimer, generation uint64) {
	for {
		t.mu.Lock()
		if !t.active || t.generation != generation {
			t.mu.Unlock()
			return
		}
		deadline := t.when
		t.mu.Unlock()
		Sleep(Duration(deadline - runtimeNano()))

		t.mu.Lock()
		if !t.active || t.generation != generation {
			t.mu.Unlock()
			return
		}
		now := runtimeNano()
		if now < t.when {
			t.mu.Unlock()
			continue
		}
		if t.period > 0 {
			// Match libgo runtime's phase-preserving rescheduling. Missed
			// intervals are skipped rather than emitted in a burst.
			delay := t.period - (now-t.when)%t.period
			t.when = now + delay
			if t.when < 0 {
				t.when = 1<<63 - 1
			}
		} else {
			t.active = false
		}
		// libgo uses sendTime (a nonblocking send) or goFunc (starts a
		// goroutine). Keeping the lock prevents firing a canceled generation.
		t.f(t.arg, t.seq)
		active := t.active
		t.mu.Unlock()
		if !active {
			return
		}
	}
}
