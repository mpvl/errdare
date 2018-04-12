// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package errdare

import (
	"io"

	"github.com/mpvl/errdare/errtest"
)

func require(s *errtest.Simulation, v Value, key string) {
	if v.key() != key {
		s.Fatalf("got %q; want %q", v.key(), key)
	}
}

func mustCall(s *errtest.Simulation, err error, keys ...string) error {
	return err
}

// Value is any value returned by a call.
type Value interface {
	key() string
}

type key string

func (k key) key() string { return string(k) }

// Client is a Value with a Close method.
type Client interface {
	Value
	io.Closer
}

// A Writer is a Value with a Close and CloseWithError method.
type Writer interface {
	Value
	io.Closer
	CloseWithError(err error) error
}

// A Reader is a Value with a Close method.
type Reader interface {
	Value
	io.Closer
}

// An Aborter is a Value with a Close and Abort method.
type Aborter interface {
	Value
	io.Closer
	Abort(err error)
}

type value struct {
	s         *errtest.Simulation
	keyStr    string
	closeOpts []errtest.Option
}

func ve(s *errtest.Simulation, key string, opts ...errtest.Option) (*value, error) {
	err := s.Open(key, opts...)
	return &value{s, key, nil}, err
}

func v(s *errtest.Simulation, key string, opts ...errtest.Option) *value {
	s.Open(key, append(opts, errtest.NoError())...)
	return &value{s, key, nil}
}

func e(s *errtest.Simulation, key string, opts ...errtest.Option) error {
	return s.Open(key, append(opts, errtest.NoClose())...)
}

func do(s *errtest.Simulation, key string, opts ...errtest.Option) {
	s.Open(key, append(opts, errtest.NoError(), errtest.NoClose())...)
}

func (v *value) key() string { return v.keyStr }

func (v *value) Close() error {
	return v.s.Close(v.key(), v.closeOpts...)
}

func (v *value) CloseWithError(err error) error {
	return v.s.CloseWithError(v.key(), err, v.closeOpts...)
}

func (v *value) Abort(err error) {
	v.s.Close(v.key(), v.closeOpts...)
}
