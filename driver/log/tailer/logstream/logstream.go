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

// Package logstream provides an interface and implementations of log source
// streaming. Each log streaming implementation provides an abstraction that
// makes one pathname look like one perpetual source of logs, even though the
// underlying file objects might be truncated or rotated.
// Adapted from https://github.com/google/mtail/tree/main/internal
package logstream

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/sysflow-telemetry/sf-apis/go/logger"
	"github.com/sysflow-telemetry/sf-processor/driver/log/logline"
	"github.com/sysflow-telemetry/sf-processor/driver/log/waker"
)

var (
	// logErrors counts the IO errors encountered per log.
	logErrors = expvar.NewMap("log_errors_total")
	// logOpens counts the opens of new log file descriptors/sockets.
	logOpens = expvar.NewMap("log_opens_total")
	// logCloses counts the closes of old log file descriptors/sockets.
	logCloses = expvar.NewMap("log_closes_total")
)

// LogStream.
type LogStream interface {
	LastReadTime() time.Time // Return the time when the last log line was read from the source
	Stop()                   // Ask to gracefully stop the stream; e.g. stream keeps reading until EOF and then completes work.
	IsComplete() bool        // True if the logstream has completed work and cannot recover.  The caller should clean up this logstream, creating a new logstream on a pathname if necessary.
}

// defaultReadBufferSize the size of the buffer for reading bytes into.
const defaultReadBufferSize = 4096

var (
	ErrUnsupportedURLScheme = errors.New("unsupported URL scheme")
	ErrUnsupportedFileType  = errors.New("unsupported file type")
	ErrEmptySocketAddress   = errors.New("socket address cannot be empty, please provide a unix domain socket filename or host:port")
)

// New creates a LogStream from the file object located at the absolute path
// `pathname`.  The LogStream will watch `ctx` for a cancellation signal, and
// notify the `wg` when it is Done.  Log lines will be sent to the `lines`
// channel.  `seekToStart` is only used for testing and only works for regular
// files that can be seeked.
func New(ctx context.Context, wg *sync.WaitGroup, waker waker.Waker, pathname string, lines chan<- *logline.LogLine, oneShot bool) (LogStream, error) {
	u, err := url.Parse(pathname)
	if err != nil {
		return nil, err
	}
	logger.Info.Printf("Parsed url as %v", u)

	path := pathname
	switch u.Scheme {
	default:
		logger.Info.Printf("%v: %q in path pattern %q, treating as path", ErrUnsupportedURLScheme, u.Scheme, pathname)
	case "", "file":
		path = u.Path
	}
	fi, err := os.Stat(path)
	if err != nil {
		logErrors.Add(path, 1)
		return nil, err
	}
	switch m := fi.Mode(); {
	case m.IsRegular():
		return newFileStream(ctx, wg, waker, path, fi, lines, oneShot)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedFileType, pathname)
	}
}
