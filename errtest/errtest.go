// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errtest

import (
	"fmt"
	"testing"
)

// A Config is used to configure a simulation.
type Config struct {
	IgnorePanicOrder    bool
	RequireCloseOnPanic bool

	SkipErrors bool // call Skip on testing.T for any error it encounters.
}

// These Config values are some common values
var (
	Pedantic *Config = &Config{
		RequireCloseOnPanic: true,
	}

	Relaxed *Config = &Config{
		IgnorePanicOrder: true,
	}

	SkipErrors *Config = &Config{
		SkipErrors: true,
	}
)

// Another case: error friendly, like EOF, but panic may override this.
// in general: override friendly error with mean error.

type mode int

const (
	modeNoError mode = iota
	modeError
	modePanic
)

func (m mode) String() string {
	return map[mode]string{
		modeNoError: "NoError",
		modePanic:   "Panic",
		modeError:   "Error",
	}[m]
}

type simError struct {
	mode mode
	key  string
}

type fatalError struct {
	error
}

func (e simError) Error() string { return fmt.Sprintf("%s: %s", e.key, e.mode) }

// An Option configures a simulation.
type Option func(*options)

type options struct {
	frame

	noError bool
	noPanic bool
}

func NoClose() Option {
	return func(o *options) { o.noClose = true }
}

func NoError() Option {
	return func(o *options) { o.noError = true }
}

func NoPanic() Option {
	return func(o *options) { o.noPanic = true }
}

func IgnoreError() Option {
	return func(o *options) { o.ignoreError = true }
}

// func OnClose(f func(err error)) Option {
// 	return func(fr *frame) { fr.onClose = f }
// }

type frame struct {
	key         string
	modes       []mode
	modeIndex   int
	noClose     bool
	ignoreError bool
	// onClose   func(err error)
}

type Simulation struct {
	testT  *testing.T
	fatalf func(format string, args ...interface{})
	config *Config

	runIndex int
	run      []frame

	// mustErr is the error that must be returned by the simulation function.
	// This is always nil or a simError.
	mustErr error
}

func (s *Simulation) ignorePanicOrder() bool {
	if s.config == nil {
		return false
	}
	return s.config.IgnorePanicOrder
}

func (s *Simulation) skipErrors() bool {
	if s.config == nil {
		return false
	}
	return s.config.SkipErrors
}

// Run runs simulations by repeatedly calling s until all possible scenarios of
// a simulation are covered.
func Run(t *testing.T, config *Config, f func(s *Simulation) error) {
	sim := &Simulation{
		config: config,
	}
	runSim(t, sim, f)
	for sim.incRun() {
		runSim(t, sim, f)
	}
}

func isPanic(err error) bool {
	if err == nil {
		return false
	}
	return err.(simError).mode == modePanic
}

func runSim(t *testing.T, s *Simulation, f func(s *Simulation) error) {
	t.Run("", func(t *testing.T) {
		s.runIndex = 0
		s.mustErr = nil
		s.testT = t
		s.fatalf = t.Fatalf
		var err error
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(simError); !ok {
					if !s.config.IgnorePanicOrder {
						panic(r)
					}
					err = simError{mode: modePanic, key: "user"}
				}
				// TODO: be pedantic and check that we have the right kind of
				// panic?
				if s.mustErr == nil || !isPanic(s.mustErr) {
					s.Fatalf("simulation panicked unexpectedly")
				}
			}
			if err != s.mustErr {
				if s.mustErr == nil || !isPanic(s.mustErr) {
					s.Fatalf("simulation did not return the correct error: got %v; want %v", err, s.mustErr)
				}
			}
		}()
		err = f(s)
	})
}

func (s *Simulation) incRun() bool {
	for len(s.run) > 0 {
		p := len(s.run) - 1
		s.run[p].modeIndex++
		if s.run[p].modeIndex != len(s.run[p].modes) {
			return true
		}
		s.run = s.run[:p]
	}
	return false
}

func (s *Simulation) setMustError(m mode, key string) error {
	err := simError{m, key}
	if s.mustErr == nil {
		s.mustErr = err
	} else if e := s.mustErr.(simError); m == modePanic && e.mode != modePanic {
		s.mustErr = err
	}
	return err
}

func (s *Simulation) Fatalf(format string, args ...interface{}) {
	if s.skipErrors() {
		s.testT.Logf(format, args...)
	} else {
		s.fatalf(format, args...)
	}
	s.testT.SkipNow()
}

func (s *Simulation) Open(key string, opts ...Option) error {
	o := options{
		frame: frame{key: key},
	}
	for _, fn := range opts {
		fn(&o)
	}
	o.modes = append(o.modes, modeNoError)
	if !o.noError {
		o.modes = append(o.modes, modeError)
	}
	if !o.noPanic {
		o.modes = append(o.modes, modePanic)
	}
	if s.runIndex == len(s.run) {
		// New entry. Ensure that a statement with this key wasn't already
		// executed.
		for _, f := range s.run {
			if f.key == key {
				s.Fatalf("statement %q was already executed", key)
				return nil
			}
		}
		s.run = append(s.run, o.frame)
	} else {
		// Simulation of a variation of a previous run. Expect the same key as
		// before.
		if s.run[s.runIndex].key != key {
			s.Fatalf("non-deterministic simulation at %q", key)
			return nil
		}
		o.frame.modeIndex = s.run[s.runIndex].modeIndex
		s.run[s.runIndex] = o.frame
	}
	defer func() { s.runIndex++ }()
	switch f := s.run[s.runIndex]; f.modes[f.modeIndex] {
	case modeError:
		s.run[s.runIndex].noClose = true
		if !f.ignoreError {
			s.setMustError(modeError, key)
		}
		// fmt.Println(key, "errr")
		return simError{modeError, key}
	case modePanic:
		// fmt.Println(key, "panic")
		s.run[s.runIndex].noClose = true
		panic(s.setMustError(modePanic, key))
	}
	// fmt.Println(key, "success")
	return nil
}

func (s *Simulation) Close(key string, opts ...Option) error {
	return s.CloseWithError(key, s.mustErr, opts...)
}

func (s *Simulation) CloseWithError(key string, err error, opts ...Option) error {
	p := len(s.run) - 1
	for ; p >= 0; p-- {
		f := s.run[p]
		if !f.noClose {
			s.run[p].noClose = true
			if f.key != key {
				s.Fatalf("%q closed in wrong order (expected %q)", f.key, key)
				return nil
			}
			if err != s.mustErr {
				if !s.ignorePanicOrder() || !isPanic(err) || !isPanic(s.mustErr) {
					s.Fatalf("close of %q with wrong error: got %v; want %v", key, err, s.mustErr)
					return nil
				}
			}
			return s.Open(key+".close", append(opts, NoClose())...)
		}
		if f.key == key {
			s.Fatalf("%q was already closed or should not be closed", key)
			return nil
		}
	}
	s.Fatalf("unmatched close %q", key)
	return nil
}
