//
// Copyright (C) 2022 IBM Corporation.
//
// Authors:
// Frederico Araujo <frederico.araujo@ibm.com>
// Teryl Taylor <terylt@ibm.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package waker provides an interface for a routine waker.
// Adapted from https://github.com/google/mtail/tree/main/internal
package waker

import (
	"context"
	"sync"

	"github.com/sysflow-telemetry/sf-apis/go/logger"
)

// A testWaker is used to manually signal to idle routines it's time to look for new work.
type testWaker struct {
	Waker

	ctx context.Context

	n int

	wakeeReady chan struct{}
	wakeeDone  chan struct{}
	wait       chan struct{}

	mu   sync.Mutex // protects following fields
	wake chan struct{}
}

// WakeFunc describes a function used by tests to trigger a wakeup of blocked idle goroutines under test.  It takes as first parameter the number of goroutines to await before returning to the caller.
type WakeFunc func(int)

// NewTest creates a new Waker to be used in tests, returning it and a function to trigger a wakeup.  The constructor parameter says how many wakees are expected in the first pass.
func NewTest(ctx context.Context, n int) (Waker, WakeFunc) {
	t := &testWaker{
		ctx:        ctx,
		n:          n,
		wakeeReady: make(chan struct{}),
		wakeeDone:  make(chan struct{}),
		wait:       make(chan struct{}),
		wake:       make(chan struct{}),
	}
	initDone := make(chan struct{})
	go func() {
		defer close(initDone)
		for i := 0; i < t.n; i++ {
			<-t.wakeeDone
		}
	}()
	wakeFunc := func(after int) {
		<-initDone
		logger.Info.Println("TestWaker yielding to Wakee")
		for i := 0; i < t.n; i++ {
			t.wait <- struct{}{}
		}
		logger.Info.Printf("waiting for %d wakees to get the wake chan", t.n)
		for i := 0; i < t.n; i++ {
			<-t.wakeeReady
		}
		t.broadcastWakeAndReset()
		// Now wakeFunc blocks here
		logger.Info.Printf("waiting for %d wakees to return to Wake", after)
		for i := 0; i < after; i++ {
			<-t.wakeeDone
		}
		t.n = after
		logger.Info.Println("Wakee yielding to TestWaker")
	}
	return t, wakeFunc
}

// Wake satisfies the Waker interface.
func (t *testWaker) Wake() (w <-chan struct{}) {
	t.mu.Lock()
	w = t.wake
	t.mu.Unlock()
	logger.Info.Println("waiting for wakeup on chan ", w)
	// Background this so we can return the wake channel.
	// The wakeFunc won't close the channel until this completes.
	go func() {
		// Signal we've reentered Wake.  wakeFunc can't return until we do this.
		select {
		case <-t.ctx.Done():
			return
		case t.wakeeDone <- struct{}{}:
		}
		// Block wakees here until a subsequent wakeFunc is called.
		select {
		case <-t.ctx.Done():
			return
		case <-t.wait:
		}
		// Signal we've got the wake chan, telling wakeFunc it can now issue a broadcast.
		select {
		case <-t.ctx.Done():
			return
		case t.wakeeReady <- struct{}{}:
		}
	}()
	return
}

func (t *testWaker) broadcastWakeAndReset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	logger.Info.Printf("broadcasting wake to chan %p", t.wake)
	close(t.wake)
	t.wake = make(chan struct{})
	logger.Info.Println("wake channel reset")
}

// alwaysWaker never blocks the wakee.
type alwaysWaker struct {
	wake chan struct{}
}

func NewTestAlways() Waker {
	w := &alwaysWaker{
		wake: make(chan struct{}),
	}
	close(w.wake)
	return w
}

func (w *alwaysWaker) Wake() <-chan struct{} {
	return w.wake
}
