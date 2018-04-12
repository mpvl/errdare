// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errtest

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

func TestSimulation(t *testing.T) {
	count := 0
	testCases := []struct {
		desc   string
		config *Config
		err    *simError
		count  int
		f      func(s *Simulation) error
		errs   string
	}{{
		desc:  "only successful run",
		count: 1,
		f: func(s *Simulation) error {
			s.Open("reader", NoError(), NoPanic(), NoClose())
			return nil
		},
	}, {
		desc:  "one succesful and one error run",
		count: 2,
		f: func(s *Simulation) error {
			return s.Open("reader", NoPanic(), NoClose())
		},
	}, {
		desc:  "success, error, panic",
		count: 3,
		f: func(s *Simulation) error {
			return s.Open("reader", NoClose())
		},
	}, {
		desc:  "success, error, and panic with Close, ignore error",
		count: 5,
		f: func(s *Simulation) error {
			err := s.Open("reader")
			if err != nil {
				return err
			}
			defer s.Close("reader", IgnoreError())
			return nil
		},
	}, {
		desc:  "fail to ignore error",
		count: 3,
		f: func(s *Simulation) (err error) {
			return s.Open("reader", IgnoreError())
		},
		errs: "1:simulation did not return the correct error: got reader: Error; want <nil>\n",
	}, {
		desc:  "fail to ignore an error in close",
		count: 5,
		f: func(s *Simulation) (err error) {
			err = s.Open("reader")
			if err != nil {
				return err
			}
			defer func() {
				errClose := s.Close("reader", IgnoreError())
				if errClose != nil && err == nil {
					err = errClose
				}
			}()
			return nil
		},
		errs: "1:simulation did not return the correct error: got reader.close: Error; want <nil>\n",
	}, {
		desc:  "success, error, and panic with Close",
		count: 5,
		f: func(s *Simulation) (err error) {
			err = s.Open("reader")
			if err != nil {
				return err
			}
			defer func() {
				errClose := s.Close("reader")
				if errClose != nil && err == nil {
					err = errClose
				}
			}()
			return nil
		},
	}, {
		desc:  "incorrect error returned",
		count: 7,
		f: func(s *Simulation) (err error) {
			err = s.Open("reader")
			return s.Open("writer")
		},
		errs: `3:simulation did not return the correct error: got <nil>; want reader: Error
4:simulation did not return the correct error: got writer: Error; want reader: Error
`,
	}, {
		desc:  "incorrect return from Close",
		count: 5,
		f: func(s *Simulation) (err error) {
			err = s.Open("reader")
			if err != nil {
				return err
			}
			defer s.Close("reader")
			return nil
		},
		errs: "1:simulation did not return the correct error: got <nil>; want reader.close: Error\n",
	}, {
		desc:   "closed in incorrect order",
		config: Pedantic,
		count:  1,
		f: func(s *Simulation) (err error) {
			s.Open("o1", NoError(), NoPanic())
			s.Open("o2", NoError(), NoPanic())
			s.Close("o1", NoError(), NoPanic())
			s.Close("o2", NoError(), NoPanic())
			return nil
		},
		errs: `0:"o2" closed in wrong order (expected "o1")
`,
	}, {
		desc:  "closed twice",
		count: 1,
		f: func(s *Simulation) (err error) {
			s.Open("o1", NoError(), NoPanic())
			s.Close("o1", NoError(), NoPanic())
			s.Close("o1", NoError(), NoPanic())
			return nil
		},
		errs: `0:"o1" was already closed or should not be closed
`,
	}, {
		desc:  "disallowed close",
		count: 1,
		f: func(s *Simulation) (err error) {
			s.Open("o1", NoError(), NoPanic(), NoClose())
			s.Close("o1", NoError(), NoPanic())
			return nil
		},
		errs: `0:"o1" was already closed or should not be closed
`,
	}, {
		desc:  "unmatched close",
		count: 1,
		f: func(s *Simulation) (err error) {
			s.Close("o2", NoError(), NoPanic())
			return nil
		},
		errs: `0:unmatched close "o2"
`,
	}, {
		desc:   "closed with wrong error",
		config: Pedantic,
		count:  5,
		f: func(s *Simulation) (err error) {
			s.Open("o1", NoError(), NoPanic())
			defer func() {
				s.CloseWithError("o1", err, NoError(), NoPanic())
			}()

			if err := s.Open("o2"); err == nil {
				s.Close("o2")
			}
			return nil
		},
		errs: `1:close of "o1" with wrong error: got <nil>; want o2.close: Error
1:simulation did not return the correct error: got <nil>; want o2.close: Error
2:close of "o1" with wrong error: got <nil>; want o2.close: Panic
3:close of "o1" with wrong error: got <nil>; want o2: Error
3:simulation did not return the correct error: got <nil>; want o2: Error
4:close of "o1" with wrong error: got <nil>; want o2: Panic
`,
	}, {
		desc:  "duplicate entry",
		count: 1,
		f: func(s *Simulation) (err error) {
			s.Open("reader", NoError(), NoPanic())
			s.Open("reader", NoError(), NoPanic())
			return nil
		},
		errs: `0:statement "reader" was already executed
`,
	}, {
		desc:  "non-deterministic simulation",
		count: 3,
		f: func(s *Simulation) (err error) {
			if count != 2 {
				return s.Open("reader")
			}
			return s.Open("writer")
		},
		errs: `1:non-deterministic simulation at "writer"
`,
	}, {
		desc:  "unexpected panic",
		count: 1,
		f: func(s *Simulation) (err error) {
			panic(simError{})
		},
		errs: "0:simulation panicked unexpectedly\n",
	}}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			count = 0
			errs := ""
			Run(t, nil, func(s *Simulation) error {
				s.fatalf = func(format string, args ...interface{}) {

					format = strconv.Itoa(count-1) + ":" + format + "\n"
					errs += fmt.Sprintf(format, args...)
				}
				count++
				err := tc.f(s)
				return err
			})
			if count != tc.count {
				t.Errorf("count: got %d; want %d", count, tc.count)
			}
			if !reflect.DeepEqual(errs, tc.errs) {
				t.Errorf("sim errors:\ngot:\n%swant:\n%s", errs, tc.errs)
			}
		})
	}
}
