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

// Package falco implements a rules engine based on falco rules.
package falco

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/sysflow-telemetry/sf-processor/core/policyengine/engine"
	"github.ibm.com/sysflow/sf-processor-filehasher/core/policyengine/engine"
)

// True defines a functional predicate that always returns true.
var True = engine.Criterion[*Record]{func(r *Record) (bool, error) { return true, nil }}

// False defines a functional predicate that always returns false.
var False = engine.Criterion[*Record]{func(r *Record) (bool, error) { return false, nil }}

type Operations struct {
}

// All derives the conjuctive clause of all predicates in a slice of predicates.
func (op Operations) All(criteria []engine.Criterion[*Record]) engine.Criterion[*Record] {
	all := True
	for _, c := range criteria {
		all = all.And(c)
	}
	return all
}

// Any derives the disjuntive clause of all predicates in a slice of predicates.
func (op Operations) Any(criteria []engine.Criterion[*Record]) engine.Criterion[*Record] {
	any := False
	for _, c := range criteria {
		any = any.Or(c)
	}
	return any
}

// Exists creates a criterion for an existential predicate.
func (op Operations) Exists(attr string) engine.Criterion[*Record] {
	m := Mapper.Map(attr)
	p := func(r *Record) bool { return !reflect.ValueOf(m(r)).IsZero() }
	return engine.Criterion[*Record]{p}
}

// Eq creates a criterion for an equality predicate.
func (op Operations) Eq(lattr string, rattr string) engine.Criterion[*Record] {
	ml := Mapper.MapStr(lattr)
	mr := Mapper.MapStr(rattr)
	p := func(r *Record) bool { return eval(ml(r), mr(r), ops.eq) }
	return engine.Criterion[*Record]{p}
}

// NEq creates a criterion for an inequality predicate.
func (op Operations) NEq(lattr string, rattr string) engine.Criterion[*Record] {
	return op.Eq(lattr, rattr).Not()
}

// Ge creates a criterion for a greater-or-equal predicate.
func (op Operations) Ge(lattr string, rattr string) engine.Criterion[*Record] {
	ml := Mapper.MapInt(lattr)
	mr := Mapper.MapInt(rattr)
	p := func(r *engine.Record) bool { return ml(r) >= mr(r) }
	return engine.Criterion[*Record]{p}
}

// Gt creates a criterion for a greater-than predicate.
func (op Operations) Gt(lattr string, rattr string) engine.Criterion[*Record] {
	ml := Mapper.MapInt(lattr)
	mr := Mapper.MapInt(rattr)
	p := func(r *engine.Record) bool { return ml(r) > mr(r) }
	return engine.Criterion[*Record]{p}
}

// Le creates a criterion for a lower-or-equal predicate.
func (op Operations) Le(lattr string, rattr string) engine.Criterion[*Record] {
	return op.Gt(lattr, rattr).Not()
}

// Lt creates a criterion for a lower-than predicate.
func (op Operations) Lt(lattr string, rattr string) engine.Criterion[*Record] {
	return op.Ge(lattr, rattr).Not()
}

// StartsWith creates a criterion for a starts-with predicate.
func (op Operations) StartsWith(lattr string, rattr string) engine.Criterion[*Record] {
	ml := Mapper.MapStr(lattr)
	mr := Mapper.MapStr(rattr)
	p := func(r *engine.Record) bool { return eval(ml(r), mr(r), ops.startswith) }
	return engine.Criterion[*Record]{p}
}

// EndsWith creates a criterion for a ends-with predicate.
func (op Operations) EndsWith(lattr string, rattr string) engine.Criterion[*Record] {
	ml := Mapper.MapStr(lattr)
	mr := Mapper.MapStr(rattr)
	p := func(r *engine.Record) bool { return eval(ml(r), mr(r), ops.endswith) }
	return engine.Criterion[*Record]{p}
}

// Contains creates a criterion for a contains predicate.
func Contains(lattr string, rattr string) Criterion {
	ml := Mapper.MapStr(lattr)
	mr := Mapper.MapStr(rattr)
	p := func(r *engine.Record) bool { return eval(ml(r), mr(r), ops.contains) }
	return Criterion{p}
}

// IContains creates a criterion for a case-insensitive contains predicate.
func IContains(lattr string, rattr string) Criterion {
	ml := Mapper.MapStr(lattr)
	mr := Mapper.MapStr(rattr)
	p := func(r *engine.Record) bool { return eval(ml(r), mr(r), ops.icontains) }
	return Criterion{p}
}

// In creates a criterion for a list-inclusion predicate.
func In(attr string, list []string) Criterion {
	m := Mapper.MapStr(attr)
	p := func(r *engine.Record) bool {
		for _, v := range list {
			if eval(m(r), v, ops.eq) {
				return true
			}
		}
		return false
	}
	return Criterion{p}
}

// PMatch creates a criterion for a list-pattern-matching predicate.
func PMatch(attr string, list []string) Criterion {
	m := Mapper.MapStr(attr)
	p := func(r *engine.Record) bool {
		for _, v := range list {
			if eval(m(r), v, ops.contains) {
				return true
			}
		}
		return false
	}
	return Criterion{p}
}

// operator type.
type operator func(string, string) bool

// operators struct.
type operators struct {
	eq         operator
	contains   operator
	icontains  operator
	startswith operator
	endswith   operator
}

// ops defines boolean comparison operators over strings.
var ops = operators{
	eq:         func(l string, r string) bool { return l == r },
	contains:   func(l string, r string) bool { return strings.Contains(l, r) },
	icontains:  func(l string, r string) bool { return strings.Contains(strings.ToLower(l), strings.ToLower(r)) },
	startswith: func(l string, r string) bool { return strings.HasPrefix(l, r) },
	endswith:   func(l string, r string) bool { return strings.HasSuffix(l, r) },
}

// Eval evaluates a boolean operator over two predicates.
func eval(l interface{}, r interface{}, op operator) bool {
	lattrs := strings.Split(fmt.Sprintf("%v", l), engine.LISTSEP)
	rattrs := strings.Split(fmt.Sprintf("%v", r), engine.LISTSEP)
	for _, lattr := range lattrs {
		for _, rattr := range rattrs {
			if op(lattr, rattr) {
				return true
			}
		}
	}
	return false
}
