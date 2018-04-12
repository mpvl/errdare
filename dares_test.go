// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package errdare

import "testing"

const dareOn = false

func TestCloudStorage(t *testing.T) {
	RunCloudStorage(t, dareConfig(), func(t *CloudStorage) error {
		c, err := t.NewClient()
		if err != nil {
			return err
		}
		defer c.Close()

		r, err := t.NewReader()
		if err != nil {
			return err
		}
		defer r.Close()

		w := t.NewWriter(c)
		defer func() { w.CloseWithError(err) }()

		_, err = t.Copy(w, r)
		return err
	})
}

func TestPipeConvert(t *testing.T) {
	RunPipeConvert(t, dareConfig(), func(t *PipeConvert, r Reader) error {
		pipeReader, pipeWriter := t.Pipe()
		go func() {
			var err error
			defer func() { pipeWriter.CloseWithError(err) }()
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

func TestTrickyCatch(t *testing.T) {
	RunTrickyCatch(t, dareConfig(), func(t *TrickyCatch) (err error) {
		w, err := t.NewWriter()
		if err != nil {
			return err
		}
		defer func() { w.CloseWithError(err) }() // Close may return error, even if err is not

		ww, err := t.NewWrapper(w)
		if err != nil {
			return err
		}
		defer ww.Close() // must catch error, but may also panic.

		err = t.WriteSomething(ww)
		return err
	})
}
