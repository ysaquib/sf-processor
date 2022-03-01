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

// Package logline provides the data structure for a log line.
// Adapted from https://github.com/google/mtail/tree/main/internal
package logline

import "context"

// LogLine contains all the information about a line just read from a log.
type LogLine struct {
	Context context.Context

	Filename string // The log filename that this line was read from
	Line     string // The text of the log line itself up to the newline.
}

// New creates a new LogLine object.
func New(ctx context.Context, filename string, line string) *LogLine {
	return &LogLine{ctx, filename, line}
}
