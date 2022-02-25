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

// Package syslog implements pluggable drivers for syslog ingestion.
package syslog

import (
	"fmt"
	"testing"

	"github.com/influxdata/go-syslog/rfc5424"
)

func TestSyslog(t *testing.T) {
	i := []byte(`<165>4 2018-10-11T22:14:15.003Z mymach.it e - 1 [ex@32473 iut="3"] An application event log entry...`)
	p := rfc5424.NewParser()
	best := true
	m, _ := p.Parse(i, &best)

	fmt.Printf("%v\n", *m.Message())
}

func TestSyslog2(t *testing.T) {
	i := []byte(`type=INTEGRITY_RULE msg=audit(1645464963.927:39061): file="/var/lib/dpkg/tmp.ci/preinst" hash="sha256:f8696a98c8ae7b0cccfd7051d3c3442e307160c605808afefe3e0c17fd244eb8" ppid=31395 pid=31439 auid=4294967295 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=4294967295 comm="dpkg" exe="/usr/bin/dpkg"`)
	p := rfc5424.NewParser()
	best := true
	m, _ := p.Parse(i, &best)

	fmt.Printf("%v\n", *m.Message())
}
