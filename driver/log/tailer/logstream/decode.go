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
	"expvar"
	"unicode/utf8"

	"github.com/sysflow-telemetry/sf-apis/go/logger"
	"github.com/sysflow-telemetry/sf-processor/driver/log/logline"
)

// logLines counts the number of lines read per log file.
var logLines = expvar.NewMap("log_lines_total")

// decodeAndSend transforms the byte array `b` into unicode in `partial`, sending to the llp as each newline is decoded.
func decodeAndSend(ctx context.Context, lines chan<- *logline.LogLine, pathname string, n int, b []byte, partial *bytes.Buffer) {
	var (
		r     rune
		width int
	)
	for i := 0; i < len(b) && i < n; i += width {
		r, width = utf8.DecodeRune(b[i:])
		// Most file-based log sources will end with \n on Unixlike systems.
		// On Windows they appear to be both \r\n.  syslog disallows \r (and \t
		// and others) and writes them escaped, per syslog(7).  [RFC
		// 3164](https://www.ietf.org/rfc/rfc3164.txt) disallows newlines in
		// the message: "The MSG part of the syslog packet MUST contain visible
		// (printing) characters."  So for now let's assume that a \r only
		// occurs at the end of a line anyway, and we can just eat it.
		switch {
		case r == '\r':
			// nom
		case r == '\n':
			sendLine(ctx, pathname, partial, lines)
		default:
			partial.WriteRune(r)
		}
	}
}

func sendLine(ctx context.Context, pathname string, partial *bytes.Buffer, lines chan<- *logline.LogLine) {
	logger.Trace.Printf("sendline")
	logLines.Add(pathname, 1)
	lines <- logline.New(ctx, pathname, partial.String())
	partial.Reset()
}
