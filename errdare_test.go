// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errdare

import (
	"flag"

	"github.com/mpvl/errdare/errtest"
)

var (
	enableDare = flag.Bool("dare", dareOn,
		"enable testsing of dares, which includes failing tests")

	panicOrder = flag.Bool("panic_order", false,
		"require the first panic to be passed to an error referenced in defer")

	closeOnPanic = flag.Bool("panic_close", false,
		"require closes to be called in case of panic")

	pedantic = flag.Bool("pedantic", false,
		"strictest interpretation; overrides all other flags except wrapping")
)

func config() *errtest.Config {
	if *pedantic {
		return errtest.Pedantic
	}
	c := &errtest.Config{
		RequireCloseOnPanic: *closeOnPanic,
		IgnorePanicOrder:    !*panicOrder,
	}
	return c
}

func dareConfig() *errtest.Config {
	c := config()
	c.SkipErrors = !*enableDare
	return c
}
