// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package errdare

import (
	"testing"
	"time"

	"github.com/mpvl/errdare/errtest"
)

// The CloudStorage challenge: open the client, reader, and writer and copy the
// contents of the reader to the writer. Any error while copying the contents
// should result in a non-nil error being passed to the Writer's CloseWithError
// method.
//
// A simplistic, but incorrect, implementation is:
//
//  func TestCloudStorage(t *testing.T) {
//  	errdare.RunCloudStorage(t, nil, func(tc *CloudStorage) error {
//  		c, err := tc.NewClient()
//  		if err != nil {
//  			return err
//  		}
//  		defer c.Close()
//
//  		r, err := tc.NewReader()
//  		if err != nil {
//  			return err
//  		}
//  		defer r.Close()
//
//  		w := tc.NewWriter()
//  		defer func() { w.CloseWithError(err) }()
//
//  		_, err = tc.Copy(w, r)
//  		return err
//  	})
//  }
//
type CloudStorage struct {
	s *errtest.Simulation
}

// RunCloudStorage runs the CloudStorage dare as a test.
func RunCloudStorage(t *testing.T, cfg *errtest.Config, f func(t *CloudStorage) error) {
	errtest.Run(t, cfg, func(s *errtest.Simulation) error {
		return mustCall(s, f(&CloudStorage{s}), "copy")
	})
}

// NewClient returns a client that must be closed. The error of the close may
// be ignored.
func (c *CloudStorage) NewClient() (Client, error) {
	v, err := ve(c.s, "client")
	v.closeOpts = append(v.closeOpts, errtest.IgnoreError())
	return v, err
}

// NewReader returns a reader. The caller must call Close on the reader.
func (c *CloudStorage) NewReader() (Reader, error) {
	return ve(c.s, "reader")
}

// NewWriter returns a writer. The caller must call CloseWithError with a
// non-nil value if there was any error.
func (c *CloudStorage) NewWriter(client Client) Writer {
	require(c.s, client, "client")
	v := v(c.s, "writer")
	v.closeOpts = append(v.closeOpts, errtest.NoError())
	return v
}

// Copy takes a Reader and Writer and reports any error.
func (c *CloudStorage) Copy(w Writer, r Reader) (n int, err error) {
	require(c.s, r, "reader")
	require(c.s, w, "writer")
	return 0, e(c.s, "copy")
}

// The PipeConvert challenge: given a reader, wrap the reader in a scanner
// and copy the result of repeated scans into a newly created pipe.
// Return the reader passing it to Wait().
//
// Source: https://github.com/cmars/represent/blob/e19ef73980e42e849cd026150516a5cdb9f827bc/pkg/represent/eol.go
//
// An almost correct implementation is:
//
//  func TestPipeConvert(t *testing.T) {
//  	RunPipeConvert(t, skip, func(t *PipeConvert, r Reader) error {
//  		pipeReader, pipeWriter := t.Pipe()
//  		go func() {
//  			var err error
//  			defer func() { pipeWriter.CloseWithError(err) }()
//  			scanner := t.NewScanner(r)
//  			for t.Scan(scanner) {
//  				err = t.WriteScanned(pipeWriter, scanner)
//  				if err != nil {
//  					return
//  				}
//  			}
//  			err = t.ScanErr(scanner)
//  		}()
//  		return t.Wait(pipeReader)
//  	})
//  }
//
type PipeConvert struct {
	s       *errtest.Simulation
	didScan bool
	err     chan error
}

// RunPipeConvert runs the PipeConvert dare as a test.
func RunPipeConvert(t *testing.T, cfg *errtest.Config, f func(t *PipeConvert, r Reader) error) {
	errtest.Run(t, cfg, func(s *errtest.Simulation) error {
		tc := &PipeConvert{
			s:   s,
			err: make(chan error, 1),
		}
		r := v(tc.s, "reader", errtest.NoClose())
		return mustCall(tc.s, f(tc, r), "wait", "writeScanned")
	})
}

// Wait must be called on the Reader returned from Pipe.
func (p *PipeConvert) Wait(r Reader) error {
	require(p.s, r, "pipeReader")
	select {
	case err := <-p.err:
		return err
	case <-time.After(10 * time.Millisecond):
	}
	return r.Close()
}

type pipeWriter struct {
	*value
	p *PipeConvert
}

func (w *pipeWriter) Close() error {
	w.p.err <- w.p.s.Close("pipeWriter", errtest.NoError(), errtest.NoPanic())
	return nil
}

func (w *pipeWriter) CloseWithError(err error) error {
	w.p.s.CloseWithError("pipeWriter", err, errtest.NoError(), errtest.NoPanic())
	w.p.err <- err
	return nil
}

// Pipe returns a Reader and Writer. The Writer must be closed upon completion.
// It must be closed with CloseWithError and a non-nil error if any error
// occurs. The Reader must be passed to Wait to await completion.
func (p *PipeConvert) Pipe() (Reader, Writer) {
	pr := v(p.s, "pipeReader")
	pw := v(p.s, "pipeWriter")
	return pr, &pipeWriter{pw, p}
}

// NewScanner returns a new Scanner that readers from the Reader passed to the
// test.
func (p *PipeConvert) NewScanner(r Reader) Value {
	require(p.s, r, "reader")
	do(p.s, "scanner")
	return key("scanner")
}

// Scan must be called with the Scanner created from NewScanner until it returs
// false.
func (p *PipeConvert) Scan(scanner Value) bool {
	require(p.s, scanner, "scanner")
	if p.didScan {
		return false
	}
	do(p.s, "scan")
	p.didScan = true
	return true
}

// WriteScanned must be called after each successful call to Scan.
func (p *PipeConvert) WriteScanned(w Writer, scanner Value) error {
	require(p.s, w, "pipeWriter")
	require(p.s, scanner, "scanner")
	return e(p.s, "writeScanned")
}

// ScanErr must be called after the last call to Scan.
func (p *PipeConvert) ScanErr(scan Value) error {
	return e(p.s, "scanErr")
}

// The TrickyCatch challenge: create a writer, wrap it, and write something
// to it. If any error occurs during writing or wrapping, the original writer
// should be called with CloseWithError. Any error encountered should be
// returned.
//
// A simple, but incorrect implementation is:
//
//  func TestTrickyCatch(t *testing.T) {
//  	RunTrickyCatch(t, skip, func(t *TrickyCatch) (err error) {
//  		w, err := t.NewWriter()
//  		if err != nil {
//  			return err
//  		}
//  		defer func() { w.CloseWithError(err) }() // Close may return error, even if err is not
//
//  		ww, err := t.NewWrapper(w)
//  		if err != nil {
//  			return err
//  		}
//  		defer ww.Close() // must catch error, but may also panic.
//
//  		err = t.WriteSomething(ww)
//  		return err
//  	})
//  }
type TrickyCatch struct {
	s *errtest.Simulation
}

func RunTrickyCatch(t *testing.T, cfg *errtest.Config, f func(t *TrickyCatch) error) {
	errtest.Run(t, cfg, func(s *errtest.Simulation) error {
		return mustCall(s, f(&TrickyCatch{s}), "write")
	})
}

// NewWriter returns a Writer. It must be closed with CloseWithError and a
// non-nil error if any error occurred.
func (t *TrickyCatch) NewWriter() (Writer, error) {
	return ve(t.s, "writer")
}

// NewWrapper returns a Writer, given the Writer returned by NewWriter. It must
// be Closed and the error returned by the close must be observed.
func (t *TrickyCatch) NewWrapper(w Writer) (Writer, error) {
	require(t.s, w, "writer")
	return ve(t.s, "wrapper") // no error close
}

// WriteSomething writes something to the Writer returned by NewWrapper.
// It may return an error.
func (t *TrickyCatch) WriteSomething(w Writer) error {
	require(t.s, w, "wrapper")
	return e(t.s, "writeSomething")
}
