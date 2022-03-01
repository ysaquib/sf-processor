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

// Package log implements pluggable drivers for log ingestion.
package log

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sysflow-telemetry/sf-apis/go/logger"
	"github.com/sysflow-telemetry/sf-processor/driver/log/logline"
	"github.com/sysflow-telemetry/sf-processor/driver/log/tailer"
	"github.com/sysflow-telemetry/sf-processor/driver/log/testutil"
	"github.com/sysflow-telemetry/sf-processor/driver/log/waker"
)

func TestMain(m *testing.M) {
	logger.InitLoggers(logger.TRACE)
	os.Exit(m.Run())
}

func makeTestTail(t *testing.T, options ...tailer.Option) (*tailer.Tailer, chan *logline.LogLine, func(int), string, func()) {
	t.Helper()
	tmpDir := testutil.TestTempDir(t)

	// ctx, cancel := context.WithCancel(context.Background())
	ctx := context.Background()
	lines := make(chan *logline.LogLine, 1000) // 5 loglines ought to be enough for any test
	var wg sync.WaitGroup
	waker, awaken := waker.NewTest(ctx, 1)
	options = append(options, tailer.LogPatterns([]string{tmpDir}), tailer.LogstreamPollWaker(waker))
	ta, err := tailer.New(ctx, &wg, lines, options...)
	testutil.FatalIfErr(t, err)
	// return ta, lines, awaken, tmpDir, func() { cancel(); wg.Wait() }
	return ta, lines, awaken, tmpDir, func() {}
}

func TestAuditUpdateTail(t *testing.T) {
	ta, lines, awaken, dir, stop := makeTestTail(t)

	logfile := filepath.Join(dir, "audit.log")
	f := testutil.TestOpenFile(t, logfile)
	defer f.Close()

	testutil.FatalIfErr(t, ta.TailPath(logfile))
	awaken(1)

	i := []byte(`type=INTEGRITY_RULE msg=audit(1645464963.927:39061): file="/var/lib/dpkg/tmp.ci/preinst" hash="sha256:f8696a98c8ae7b0cccfd7051d3c3442e307160c605808afefe3e0c17fd244eb8" ppid=31395 pid=31439 auid=4294967295 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=4294967295 comm="dpkg" exe="/usr/bin/dpkg"`)
	testutil.WriteString(t, f, string(i))
	awaken(1)

	stop()

	received := testutil.LinesReceived(lines)
	for _, l := range received {
		logger.Info.Printf("%v", l.Line)
	}
}

func TestAuditUpdatePollTail(t *testing.T) {
	ta, lines, awaken, dir, _ := makeTestTail(t)

	logfile := filepath.Join(dir, "audit.log")
	f := testutil.TestOpenFile(t, logfile)
	defer f.Close()

	testutil.FatalIfErr(t, ta.TailPath(logfile))
	awaken(1)

	i := []byte(`type=INTEGRITY_RULE msg=audit(1645464963.927:39061): file="/var/lib/dpkg/tmp.ci/preinst" hash="sha256:f8696a98c8ae7b0cccfd7051d3c3442e307160c605808afefe3e0c17fd244eb8" ppid=31395 pid=31439 auid=4294967295 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts0 ses=4294967295 comm="dpkg" exe="/usr/bin/dpkg"\n`)
	testutil.WriteString(t, f, string(i))
	awaken(1)

	for {
		select {
		case l, ok := <-lines:
			if ok {
				logger.Info.Printf("%v", l.Line)
			}
		default:
			awaken(1)
			time.Sleep(5 * time.Second)
			// cancel()
		}
	}
}
