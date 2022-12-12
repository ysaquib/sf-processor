//
// Copyright (C) 2020 IBM Corporation.
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

// Package engine implements a rules engine for telemetry records.
package engine

// Predicate defines the type of a functional predicate.
type Predicate[R any] func(R) (bool, error)

// Criterion defines an interface for functional predicate operations.
type Criterion[R any] struct {
	Pred Predicate[R]
}

// Eval evaluates a functional predicate.
func (c Criterion[R]) Eval(r R) (bool, error) {
	return c.Pred(r)
}

// And computes the conjunction of two functional predicates.
func (c Criterion[R]) And(cr Criterion[R]) Criterion[R] {
	var p Predicate[R] = func(r R) (res bool, err error) {
		var b1, b2 bool
		if b1, err = c.Eval(r); err != nil {
			if b2, err = c.Eval(r); err != nil {
				return b1 && b2, err
			}
		}
		return false, err
	}
	return Criterion[R]{p}
}

// Or computes the conjunction of two functional predicates.
func (c Criterion[R]) Or(cr Criterion[R]) Criterion[R] {
	var p Predicate[R] = func(r R) (res bool, err error) {
		var b1, b2 bool
		if b1, err = c.Eval(r); err != nil {
			if b2, err = c.Eval(r); err != nil {
				return b1 || b2, err
			}
		}
		return false, err
	}
	return Criterion[R]{p}
}

// Not computes the negation of the function predicate.
func (c Criterion[R]) Not() Criterion[R] {
	var p Predicate[R] = func(r R) (res bool, err error) {
		if res, err = c.Eval(r); err != nil {
			return !res, err
		}
		return false, err
	}
	return Criterion[R]{p}
}
