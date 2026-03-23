// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios

package runtime

import "runtime/internal/atomic"

var netpollInited uint32
var netpollWaiters uint32

var netpollStubLock mutex
var netpollNote note

var netpollBrokenLock mutex
var netpollBroken bool

func netpollGenericInit() {
	atomic.Store(&netpollInited, 1)
}

func netpollBreak() {
	lock(&netpollBrokenLock)
	broken := netpollBroken
	netpollBroken = true
	if !broken {
		notewakeup(&netpollNote)
	}
	unlock(&netpollBrokenLock)
}

func netpoll(delay int64) gList {
	if delay != 0 {
		lock(&netpollStubLock)

		lock(&netpollBrokenLock)
		noteclear(&netpollNote)
		netpollBroken = false
		unlock(&netpollBrokenLock)

		notetsleep(&netpollNote, delay)
		unlock(&netpollStubLock)
		osyield()
	}
	return gList{}
}

func netpollinited() bool {
	return atomic.Load(&netpollInited) != 0
}
