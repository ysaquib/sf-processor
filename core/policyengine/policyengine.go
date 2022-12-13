//
// Copyright (C) 2020 IBM Corporation.
//
// Authors:
// Frederico Araujo <frederico.araujo@ibm.com>
// Teryl Taylor <terylt@ibm.com>
// Andreas Schade <san@zurich.ibm.com>
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

// Package policyengine implements a plugin for a rules engine for telemetry records.
package policyengine

import (
	"sync"

	"github.com/sysflow-telemetry/sf-apis/go/logger"
	"github.com/sysflow-telemetry/sf-apis/go/plugins"
)

const (
	pluginName  string = "policyengine"
	channelName string = "eventchan"
)

// PolicyEngine defines a driver for the Policy Engine plugin.
type PolicyEngine struct {
	// pi            *engine.PolicyInterpreter
	// outCh         []chan *engine.Record
	// config        engine.Config
	// policyMonitor monitor.PolicyMonitor
}

// NewPolicyEngine constructs a new Policy Engine plugin.
func NewPolicyEngine() plugins.SFProcessor {
	return new(PolicyEngine)
}

// GetName returns the plugin name.
func (s *PolicyEngine) GetName() string {
	return pluginName
}

// NewEventChan creates a new event record channel instance.
func NewEventChan(size int) interface{} {
	// return &engine.RecordChannel{In: make(chan *engine.Record, size)}
	return nil
}

// Register registers plugin to plugin cache.
func (s *PolicyEngine) Register(pc plugins.SFPluginCache) {
	// pc.AddProcessor(pluginName, NewPolicyEngine)
	// pc.AddChannel(channelName, NewEventChan)
}

// Init initializes the plugin.
func (s *PolicyEngine) Init(conf map[string]interface{}) (err error) {
	// s.config, _ = engine.CreateConfig(conf) // no err check, assuming defaults

	// if s.config.Mode == engine.EnrichMode {
	// 	logger.Trace.Println("Setting policy engine in 'enrich' mode")
	// 	if s.config.PoliciesPath == sfgo.Zeros.String {
	// 		return
	// 	}
	// } else {
	// 	logger.Trace.Println("Setting policy engine in 'alert' mode")
	// 	if s.config.PoliciesPath == sfgo.Zeros.String {
	// 		return errors.New("configuration attribute 'policies' missing from policy engine plugin settings")
	// 	}
	// }

	// if s.config.Monitor == engine.NoneType {
	// 	// s.pi, err = s.createPolicyInterpreter()
	// 	// if err != nil {
	// 	// 	logger.Error.Printf("Unable to compile local policies from directory %s, %v", s.config.PoliciesPath, err)
	// 	// 	return
	// 	// }
	// } else {
	// 	s.policyMonitor, err = monitor.NewPolicyMonitor(s.config, s.out)
	// 	if err != nil {
	// 		logger.Error.Printf("Unable to load policy monitor %s, %v", s.config.Monitor.String(), err)
	// 		return
	// 	}
	// 	select {
	// 	// case s.pi = <-s.policyMonitor.GetInterpreterChan():
	// 	// 	logger.Info.Printf("Loaded policy engine from policy monitor %s.", s.config.Monitor.String())
	// 	// 	s.pi.StartWorkers()
	// 	default:
	// 		logger.Error.Printf("No policy engine available for plugin. Please check error logs for details.")
	// 		return errors.New("no policy engine available for plugin")
	// 	}
	// 	s.policyMonitor.StartMonitor()
	// }
	return
}

// Process implements the main loop of the plugin.
// Records are processed concurrently. The number of concurrent threads is controlled by s.config.Concurrency.
func (s *PolicyEngine) Process(ch []interface{}, wg *sync.WaitGroup) {
	// if len(ch) != 1 {
	// 	logger.Error.Println("Policy Engine only supports a single input channel at this time")
	// 	return
	// }
	// in := ch[0].(*flattener.FlatChannel).In
	// defer wg.Done()
	// logger.Trace.Println("Starting policy engine with capacity: ", cap(in))

	// // set start and expiration time for checking for new policy interpreter
	// start := time.Now()
	// expiration := start.Add(s.config.MonitorInterval)

	// for {
	// 	if fc, ok := <-in; ok {
	// 		if s.pi == nil {
	// 			s.out(engine.NewRecord(*fc))
	// 			continue
	// 		}
	// 		if s.policyMonitor != nil {
	// 			now := time.Now()
	// 			// check if another policy interpreter has been compiled (only happens when there are changes to the policy directory)
	// 			if now.After(expiration) {
	// 				select {
	// 				// case pi := <-s.policyMonitor.GetInterpreterChan():
	// 				// 	logger.Info.Println("Updated policy interpreter in main policy engine thread.")
	// 				// 	// stop workers from old policy interpreter before assigning new one
	// 				// 	s.pi.StopWorkers()
	// 				// 	pi.StartWorkers()
	// 				// 	s.pi = pi
	// 				default:
	// 				}
	// 				expiration = now.Add(s.config.MonitorInterval)
	// 			}
	// 		}
	// 		// Process record in interpreter's worker pool
	// 		// s.pi.ProcessAsync(engine.NewRecord(*fc))
	// 	} else {
	// 		logger.Trace.Println("Input channel closed. Shutting down.")
	// 		break
	// 	}
	// }
}

// Creates a policy interpreter from configuration.
// func (s *PolicyEngine) createPolicyInterpreter() (*engine.PolicyInterpreter, error) {
// dir := s.config.PoliciesPath
// logger.Info.Println("Loading policies from: ", dir)
// paths, err := ioutils.ListFilePaths(dir, ".yaml")
// if err != nil {
// 	return nil, err
// }
// if len(paths) == 0 {
// 	return nil, errors.New("no policy files with extension .yaml found in path: " + dir)
// }
// logger.Info.Println("Creating policy interpreter")
// // pi := engine.NewPolicyInterpreter(s.config, s.out)
// // err = pi.Compile(paths...)
// if err != nil {
// 	return nil, err
// }
// // pi.StartWorkers()
// return pi, nil
// }

// out sends a record to every output channel in the plugin.
// func (s *PolicyEngine) out(r *engine.Record) {
// 	// for _, c := range s.outCh {
// 	// 	c <- r
// 	// }
// }

// SetOutChan sets the output channel of the plugin.
func (s *PolicyEngine) SetOutChan(ch []interface{}) {
	// for _, c := range ch {
	// 	s.outCh = append(s.outCh, (c.(*engine.RecordChannel)).In)
	// }
}

// Cleanup clean up the plugin resources.
func (s *PolicyEngine) Cleanup() {
	logger.Trace.Println("Exiting ", pluginName)
	// if s.pi != nil {
	// 	s.pi.StopWorkers()
	// }
	// if s.outCh != nil {
	// 	for _, c := range s.outCh {
	// 		close(c)
	// 	}
	// }
	// if s.policyMonitor != nil {
	// 	s.policyMonitor.StopMonitor()
	// }
}
