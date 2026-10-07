//go:build !kolibrios

package signal

// Defined by the upstream runtime on platforms supporting asynchronous signals.
func signalWaitUntilIdle()
