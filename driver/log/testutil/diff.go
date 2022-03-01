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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Diff(a, b interface{}, opts ...cmp.Option) string {
	return cmp.Diff(a, b, opts...)
}

func IgnoreUnexported(types ...interface{}) cmp.Option {
	return cmpopts.IgnoreUnexported(types...)
}

func AllowUnexported(types ...interface{}) cmp.Option {
	return cmp.AllowUnexported(types...)
}

func IgnoreFields(typ interface{}, names ...string) cmp.Option {
	return cmpopts.IgnoreFields(typ, names...)
}

func SortSlices(lessFunc interface{}) cmp.Option {
	return cmpopts.SortSlices(lessFunc)
}

// ExpectNoDiff tests to see if the two interfaces have no diff.
// If there is no diff, the retrun value is true.
// If there is a diff, it is logged to tb and an error is flagged, and the return value is false.
func ExpectNoDiff(tb testing.TB, a, b interface{}, opts ...cmp.Option) bool {
	tb.Helper()
	if diff := Diff(a, b, opts...); diff != "" {
		tb.Errorf("Unexpected diff, -want +got:\n%s", diff)
		tb.Logf("expected:\n%#v", a)
		tb.Logf("received:\n%#v", b)
		return false
	}
	return true
}
