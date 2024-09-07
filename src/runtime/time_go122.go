//go:build !go1.23

// Portions copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

// Time functions for Go 1.22 and below.

type puintptr uintptr

// Package time knows the layout of this structure.
// If this struct changes, adjust ../time/sleep.go:/runtimeTimer.
type timer struct {
	// If this timer is on a heap, which P's heap it is on.
	// puintptr rather than *p to match uintptr in the versions
	// of this struct defined in other packages.
	pp puintptr

	// Timer wakes up at when, and then at when+period, ... (period > 0 only)
	// each time calling f(arg, now) in the timer goroutine, so f must be
	// a well-behaved function and not block.
	//
	// when must be positive on an active timer.
	when   int64
	period int64
	f      func(any, uintptr)
	arg    any
	seq    uintptr

	// What to set the when field to in timerModifiedXX status.
	nextwhen int64

	// The status field holds one of the values below.
	status uint32
}

func (tim *timer) callCallback(delta int64) {
	tim.f(tim.arg, 0)
}

// Defined in the time package, implemented here in the runtime.
//
//go:linkname startTimer time.startTimer
func startTimer(tim *timer) {
	addTimer(&timerNode{
		timer:    tim,
		callback: timerCallback,
	})
	scheduleLog("adding timer")
}

//go:linkname stopTimer time.stopTimer
func stopTimer(tim *timer) bool {
	return removeTimer(tim)
}

//go:linkname resetTimer time.resetTimer
func resetTimer(tim *timer, when int64) bool {
	tim.when = when
	removed := removeTimer(tim)
	startTimer(tim)
	return removed
}
