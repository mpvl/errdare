// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errdare

import (
	"testing"

	"github.com/mpvl/errc"
	"github.com/mpvl/errd"
)

func TestCloudStorageCorrect(t *testing.T) {
	RunCloudStorage(t, config(), func(t *CloudStorage) (err error) {
		c, err := t.NewClient()
		if err != nil {
			return err
		}
		defer c.Close()

		r, err := t.NewReader()
		if err != nil {
			return err
		}
		defer func() {
			if errC := r.Close(); err == nil {
				err = errC
			}
		}()

		// err = errors.New("panicking")
		// w := t.NewWriter(c)
		// defer func() { w.CloseWithError(err) }()
		w := t.NewWriter(c)
		defer func() {
			if r := recover(); r != nil {
				w.CloseWithError(r.(error))
				panic(r)
			}
			w.CloseWithError(err)
		}()

		_, err = t.Copy(w, r)
		return err
	})
}

func TestCloudStorageErrc(t *testing.T) {
	RunCloudStorage(t, config(), func(t *CloudStorage) (err error) {
		e := errc.Catch(&err)
		defer e.Handle()

		c, err := t.NewClient()
		e.Must(err)
		e.Defer(c.Close, errc.Discard)

		r, err := t.NewReader()
		e.Must(err)
		e.Defer(r.Close)

		w := t.NewWriter(c)
		e.Defer(w.CloseWithError)

		_, err = t.Copy(w, r)
		e.Must(err)
		return nil
	})
}

func TestCloudStorageErrd(t *testing.T) {
	RunCloudStorage(t, config(), func(t *CloudStorage) (err error) {
		return errd.Run(func(e *errd.E) {
			c, err := t.NewClient()
			e.Must(err)
			e.Defer(c.Close, errd.Discard)

			r, err := t.NewReader()
			e.Must(err)
			e.Defer(r.Close)

			w := t.NewWriter(c)
			e.Defer(w.CloseWithError)

			_, err = t.Copy(w, r)
			e.Must(err)
		})
	})
}

func TestPipeConvertCorrect(t *testing.T) {
	RunPipeConvert(t, config(), func(t *PipeConvert, r Reader) error {
		pipeReader, pipeWriter := t.Pipe()
		go func() {
			var err error
			defer func() {
				if r := recover(); r != nil {
					err = r.(error)
					pipeWriter.CloseWithError(err)
					// simulator does not support gobbling panic
					// panic(r)
					return
				}
				pipeWriter.CloseWithError(err)
			}()
			scanner := t.NewScanner(r)
			for t.Scan(scanner) {
				err = t.WriteScanned(pipeWriter, scanner)
				if err != nil {
					return
				}
			}
			err = t.ScanErr(scanner)
		}()
		return t.Wait(pipeReader)
	})
}

func TestPipeConvertErrd(t *testing.T) {
	RunPipeConvert(t, config(), func(t *PipeConvert, r Reader) error {
		pipeReader, pipeWriter := t.Pipe()
		GoErrd(func(e *errd.E) {
			e.Defer(pipeWriter.CloseWithError)
			scanner := t.NewScanner(r)
			for t.Scan(scanner) {
				e.Must(t.WriteScanned(pipeWriter, scanner))
			}
			e.Must(t.ScanErr(scanner))
		})
		return t.Wait(pipeReader)
	})
}

func GoErrd(f func(*errd.E)) {
	go func() {
		defer func() { recover() }()
		errd.Run(f)
	}()
}

func TestTrickyCatchCorrect(t *testing.T) {
	RunTrickyCatch(t, config(), func(t *TrickyCatch) (err error) {
		w, err := t.NewWriter()
		if err != nil {
			return err
		}
		isPanic := false
		defer func() {
			r := recover()
			if r != nil && !isPanic {
				err = r.(error)
				isPanic = true
			}
			if errC := w.CloseWithError(err); err == nil {
				err = errC
			}
			if isPanic {
				panic(err)
			}
		}()

		ww, err := t.NewWrapper(w)
		if err != nil {
			return err
		}
		defer func() {
			// No need to catch panic with ignorePanicOrder.
			if errC := ww.Close(); err == nil {
				err = errC
			}
		}()

		err = t.WriteSomething(ww)
		return err
	})
}

func TestTrickyCatchErrd(t *testing.T) {
	RunTrickyCatch(t, config(), func(t *TrickyCatch) (err error) {
		return errd.Run(func(e *errd.E) {
			w, err := t.NewWriter()
			e.Must(err)
			e.Defer(w.CloseWithError)

			ww, err := t.NewWrapper(w)
			e.Must(err)
			e.Defer(ww.Close)

			e.Must(t.WriteSomething(ww))
		})
	})
}

func TestTrickyCatchErrc(t *testing.T) {
	RunTrickyCatch(t, config(), func(t *TrickyCatch) (err error) {
		e := errc.Catch(&err)
		defer e.Handle()

		w, err := t.NewWriter()
		e.Must(err)
		e.Defer(w.CloseWithError)

		ww, err := t.NewWrapper(w)
		e.Must(err)
		e.Defer(ww.Close)

		return t.WriteSomething(ww)
	})
}
