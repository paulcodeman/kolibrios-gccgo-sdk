// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build kolibrios

package runtime

type _pid_t int32

type _sigset_t [1]uint32

type _siginfo_t struct{}

type _sigaction struct{}

const (
	_SIGURG            = 0
	_NSIG              = 0
	_sizeof_ucontext_t = 0
)

//go:nosplit
func getpid() _pid_t {
	return 0
}

//go:nosplit
func sigsave(p *sigset) {
	_ = p
}

//go:nosplit
func msigrestore(sigmask sigset) {
	_ = sigmask
}

//go:nosplit
//go:nowritebarrierrec
func clearSignalHandlers() {
}

//go:nosplit
func sigblock(exiting bool) {
	_ = exiting
}

func signame(sig uint32) string {
	_ = sig
	return ""
}

func crash() {
	abort()
}

func initsig(preinit bool) {
	_ = preinit
}

func setProcessCPUProfiler(hz int32) {
	_ = hz
}

func setThreadCPUProfiler(hz int32) {
	_ = hz
}

func sigdisable(sig uint32) {
	_ = sig
}

func sigenable(sig uint32) {
	_ = sig
}

func sigignore(sig uint32) {
	_ = sig
}

const preemptMSupported = false

func preemptM(mp *m) {
	_ = mp
}
