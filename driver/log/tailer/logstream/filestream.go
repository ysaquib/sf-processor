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
	"bytes"
	"context"
	"errors"
	"expvar"
	"io"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/sysflow-telemetry/sf-apis/go/logger"
	"github.com/sysflow-telemetry/sf-processor/driver/log/logline"
	"github.com/sysflow-telemetry/sf-processor/driver/log/waker"
)

// fileTruncates counts the truncations of a file stream.
var fileTruncates = expvar.NewMap("file_truncates_total")

// fileStream streams log lines from a regular file on the file system.  These
// log files are appended to by another process, and are either rotated or
// truncated by that (or yet another) process.  Rotation implies that a new
// inode with the same name has been created, the old file descriptor will be
// valid until EOF at which point it's considered completed.  A truncation means
// the same file descriptor is used but the file offset will be reset to 0.
// The latter is potentially lossy, if the last logs are not read before truncation
// occurs.  When an EOF is read, the goroutine tests for both truncation and inode
// change and resets or spins off a new goroutine and closes itself down.  The shared
// context is used for cancellation.
type fileStream struct {
	ctx   context.Context
	lines chan<- *logline.LogLine

	pathname string // Given name for the underlying file on the filesystem

	mu           sync.RWMutex // protects following fields.
	lastReadTime time.Time    // Last time a log line was read from this file
	completed    bool         // The filestream is completed and can no longer be used.

	stopOnce sync.Once     // Ensure stopChan only closed once.
	stopChan chan struct{} // Close to start graceful shutdown.
}

// newFileStream creates a new log stream from a regular file.
func newFileStream(ctx context.Context, wg *sync.WaitGroup, waker waker.Waker, pathname string, fi os.FileInfo, lines chan<- *logline.LogLine, streamFromStart bool) (LogStream, error) {
	fs := &fileStream{ctx: ctx, pathname: pathname, lastReadTime: time.Now(), lines: lines, stopChan: make(chan struct{})}
	if err := fs.stream(ctx, wg, waker, fi, streamFromStart); err != nil {
		return nil, err
	}
	return fs, nil
}

func (fs *fileStream) LastReadTime() time.Time {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.lastReadTime
}

func (fs *fileStream) stream(ctx context.Context, wg *sync.WaitGroup, waker waker.Waker, fi os.FileInfo, streamFromStart bool) error {
	fd, err := os.OpenFile(fs.pathname, os.O_RDONLY, 0o600)
	if err != nil {
		logErrors.Add(fs.pathname, 1)
		return err
	}
	logOpens.Add(fs.pathname, 1)
	logger.Info.Printf("%v: opened new file", fd)
	if !streamFromStart {
		if _, err := fd.Seek(0, io.SeekEnd); err != nil {
			logErrors.Add(fs.pathname, 1)
			if err := fd.Close(); err != nil {
				logErrors.Add(fs.pathname, 1)
				logger.Info.Println(err)
			}
			return err
		}
		logger.Info.Printf("%v: seeked to end", fd)
	}
	b := make([]byte, defaultReadBufferSize)
	partial := bytes.NewBufferString("")
	started := make(chan struct{})
	var total int
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			logger.Info.Printf("%v: read total %d bytes from %s", fd, total, fs.pathname)
			logger.Info.Printf("%v: closing file descriptor", fd)
			if err := fd.Close(); err != nil {
				logErrors.Add(fs.pathname, 1)
				logger.Info.Println(err)
			}
			logCloses.Add(fs.pathname, 1)
		}()
		close(started)
		for {
			// Blocking read but regular files will return EOF straight away.
			count, err := fd.Read(b)
			logger.Info.Printf("%v: read %d bytes, err is %v", fd, count, err)

			if count > 0 {
				total += count
				logger.Info.Printf("%v: decode and send", fd)
				decodeAndSend(ctx, fs.lines, fs.pathname, count, b[:count], partial)
				fs.mu.Lock()
				fs.lastReadTime = time.Now()
				fs.mu.Unlock()
			}

			if err != nil && err != io.EOF {
				logErrors.Add(fs.pathname, 1)
				// TODO: This could be generalised to check for any retryable
				// errors, and end on unretriables; e.g. ESTALE looks
				// retryable.
				if errors.Is(err, syscall.ESTALE) {
					logger.Info.Printf("%v: reopening stream due to %s", fd, err)
					if nerr := fs.stream(ctx, wg, waker, fi, true); nerr != nil {
						logger.Info.Println(nerr)
					}
					// Close this stream.
					return
				}
				logger.Info.Println(err)
			}

			// If we have read no bytes and are at EOF, check for truncation and rotation.
			if err == io.EOF && count == 0 {
				logger.Info.Printf("%v: eof an no bytes", fd)
				// Both rotation and truncation need to stat, so check for
				// rotation first.  It is assumed that rotation is the more
				// common change pattern anyway.
				newfi, serr := os.Stat(fs.pathname)
				if serr != nil {
					logger.Info.Println(serr)
					// If this is a NotExist error, then we should wrap up this
					// goroutine. The Tailer will create a new logstream if the
					// file is in the middle of a rotation and gets recreated
					// in the next moment.  We can't rely on the Tailer to tell
					// us we're deleted because the tailer can only tell us to
					// Stop, which ends up causing us to race here against
					// detection of IsCompleted.
					if os.IsNotExist(serr) {
						logger.Info.Printf("%v: source no longer exists, exiting", fd)
						if partial.Len() > 0 {
							sendLine(ctx, fs.pathname, partial, fs.lines)
						}
						fs.mu.Lock()
						fs.completed = true
						fs.mu.Unlock()
						return
					}
					logErrors.Add(fs.pathname, 1)
					goto Sleep
				}
				if !os.SameFile(fi, newfi) {
					logger.Info.Printf("%v: adding a new file routine", fd)
					if err := fs.stream(ctx, wg, waker, newfi, true); err != nil {
						logger.Info.Println(err)
					}
					// We're at EOF so there's nothing left to read here.
					return
				}
				currentOffset, serr := fd.Seek(0, io.SeekCurrent)
				if serr != nil {
					logErrors.Add(fs.pathname, 1)
					logger.Info.Println(serr)
					continue
				}
				logger.Info.Printf("%v: current seek is %d", fd, currentOffset)
				logger.Info.Printf("%v: new size is %d", fd, newfi.Size())
				// We know that newfi is from the current file.  Truncation can
				// only be detected if the new file is currently shorter than
				// the current seek offset.  In test this can be a race, but in
				// production it's unlikely that a new file writes more bytes
				// than the previous after rotation in the time it takes for
				// mtail to notice.
				if newfi.Size() < currentOffset {
					logger.Info.Printf("%v: truncate? currentoffset is %d and size is %d", fd, currentOffset, newfi.Size())
					// About to lose all remaining data because of the truncate so flush the accumulator.
					if partial.Len() > 0 {
						sendLine(ctx, fs.pathname, partial, fs.lines)
					}
					p, serr := fd.Seek(0, io.SeekStart)
					if serr != nil {
						logErrors.Add(fs.pathname, 1)
						logger.Info.Println(serr)
					}
					logger.Info.Printf("%v: Seeked to %d", fd, p)
					fileTruncates.Add(fs.pathname, 1)
					continue
				}
			}

			// No error implies there is more to read in this file so go
			// straight back to read unless it looks like context is Done.
			if err == nil && ctx.Err() == nil {
				continue
			}

		Sleep:
			// If we get here it's because we've stalled.  First test to see if it's
			// time to exit.
			if err == io.EOF || ctx.Err() != nil {
				select {
				case <-fs.stopChan:
					logger.Info.Printf("%v: stream has been stopped, exiting", fd)
					if partial.Len() > 0 {
						sendLine(ctx, fs.pathname, partial, fs.lines)
					}
					fs.mu.Lock()
					fs.completed = true
					fs.mu.Unlock()
					return
				case <-ctx.Done():
					logger.Info.Printf("%v: stream has been cancelled, exiting", fd)
					if partial.Len() > 0 {
						sendLine(ctx, fs.pathname, partial, fs.lines)
					}
					fs.mu.Lock()
					fs.completed = true
					fs.mu.Unlock()
					return
				default:
					// keep going
				}
			}

			// Don't exit, instead yield and wait for a termination signal or
			// wakeup.
			logger.Info.Printf("%v: waiting", fd)
			select {
			case <-fs.stopChan:
				// We may have started waiting here when the stop signal
				// arrives, but since that wait the file may have been
				// written to.  The file is not technically yet at EOF so
				// we need to go back and try one more read.  We'll exit
				// the stream in the select stanza above.
				logger.Info.Printf("%v: Stopping after next read", fd)
			case <-ctx.Done():
				// Same for cancellation; this makes tests stable, but
				// could argue exiting immediately is less surprising.
				// Assumption is that this doesn't make a difference in
				// production.
				logger.Info.Printf("%v: Cancelled after next read", fd)
			case <-waker.Wake():
				// sleep until next Wake()
				logger.Info.Printf("%v: Wake received", fd)
			}
		}
	}()

	<-started
	return nil
}

func (fs *fileStream) IsComplete() bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.completed
}

// Stop implements the LogStream interface.
func (fs *fileStream) Stop() {
	fs.stopOnce.Do(func() {
		logger.Info.Println("signalling stop at next EOF")
		close(fs.stopChan)
	})
}
