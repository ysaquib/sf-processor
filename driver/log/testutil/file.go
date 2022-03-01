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

// Package testutil provides testing helpers.
// Adapted from https://github.com/google/mtail/tree/main/internal
package testutil

import (
	"io"
	"os"
	"testing"

	"github.com/sysflow-telemetry/sf-apis/go/logger"
)

func WriteString(tb testing.TB, f io.StringWriter, str string) int {
	tb.Helper()
	n, err := f.WriteString(str)
	FatalIfErr(tb, err)
	logger.Info.Printf("Wrote %d bytes", n)
	// If this is a regular file (not a pipe or other StringWriter) then ensure
	// it's flushed to disk, to guarantee the write happens-before this
	// returns.
	if v, ok := f.(*os.File); ok {
		fi, err := v.Stat()
		FatalIfErr(tb, err)
		if fi.Mode().IsRegular() {
			logger.Info.Printf("This is a regular file, doing a sync.")
			FatalIfErr(tb, v.Sync())
		}
	}
	return n
}
