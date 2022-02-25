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
	"os"

	"github.com/sysflow-telemetry/sf-apis/go/logger"
	"github.com/sysflow-telemetry/sf-apis/go/plugins"
)

const (
	syslogDriverName = "syslog"
)

// SyslogDriver is a driver for reading and parsing syslog data
type SyslogDriver struct {
	pipeline plugins.SFPipeline
	file     *os.File
}

// NewSyslogDriver creates a new syslog driver object
func NewSyslogDriver() plugins.SFDriver {
	return &SyslogDriver{}
}

// GetName returns the driver name.
func (s *SyslogDriver) GetName() string {
	return syslogDriverName
}

// Register registers driver to plugin cache
func (s *SyslogDriver) Register(pc plugins.SFPluginCache) {
	pc.AddDriver(syslogDriverName, NewSyslogDriver)
}

// Init initializes the file driver with the pipeline
func (s *SyslogDriver) Init(pipeline plugins.SFPipeline) error {
	s.pipeline = pipeline
	return nil
}

// Run runs the file driver
func (s *SyslogDriver) Run(path string, running *bool) error {
	channel := s.pipeline.GetRootChannel()
	sfChannel := channel.(*plugins.SFChannel)
	records := sfChannel.In

	logger.Trace.Println("Loading file: ", path)

	// sfobjcvter := converter.NewSFObjectConverter()

	// files, err := getFiles(path)
	// if err != nil {
	// 	logger.Error.Println("Files error: ", err)
	// 	return err
	// }
	// for _, fn := range files {
	// 	logger.Trace.Println("Loading file: " + fn)
	// 	s.file, err = os.Open(fn)
	// 	if err != nil {
	// 		logger.Error.Println("File open error: ", err)
	// 		return err
	// 	}
	// 	reader := bufio.NewReader(s.file)
	// 	sreader, err := goavro.NewOCFReader(reader)
	// 	if err != nil {
	// 		logger.Error.Println("Reader error: ", err)
	// 		return err
	// 	}
	// 	for sreader.Scan() {
	// 		if !*running {
	// 			break
	// 		}
	// 		datum, err := sreader.Read()
	// 		if err != nil {
	// 			logger.Error.Println("Datum reading error: ", err)
	// 			break
	// 		}
	// 		records <- sfobjcvter.ConvertToSysFlow(datum)
	// 	}
	// 	s.file.Close()
	// 	if !*running {
	// 		break
	// 	}
	// }
	logger.Trace.Println("Closing main channel")
	close(records)
	s.pipeline.Wait()
	return nil
}

// Cleanup tears down the driver resources.
func (s *SyslogDriver) Cleanup() {
	logger.Trace.Println("Exiting ", syslogDriverName)
	if s.file != nil {
		s.file.Close()
	}
}
