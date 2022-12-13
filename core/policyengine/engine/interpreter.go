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

// Package engine implements a rules engine for telemetry records.
package engine

import (
	"errors"
	"regexp"
	"sync"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/sysflow-telemetry/sf-apis/go/logger"
	"github.com/sysflow-telemetry/sf-processor/core/policyengine/lang/errorhandler"
	"github.com/sysflow-telemetry/sf-processor/core/policyengine/lang/parser"
)

// Regular expression for parsing lists.
var itemsre = regexp.MustCompile(`(^\[)(.*)(\]$?)`)

// PolicyInterpreter defines a rules engine for SysFlow data streams.
type PolicyInterpreter[R any] struct {
	*parser.BaseSfplListener

	// Mode of the policy interpreter
	mode Mode

	// Parsed rule and filter object maps
	rules   []Rule[R]
	filters []Filter[R]

	// Accessory parsing maps
	lists     map[string][]string
	macroCtxs map[string]parser.IExpressionContext

	// Worker channel and waitgroup
	workerCh chan *R
	wg       *sync.WaitGroup

	// Callback for sending records downstream
	out func(*R)

	// Worker pool size
	concurrency int

	// Action Handler
	//ah *ActionHandler
}

// NewPolicyInterpreter constructs a new interpreter instance.
func NewPolicyInterpreter[R any](conf Config, out func(*R)) *PolicyInterpreter[R] {
	pi := new(PolicyInterpreter[R])
	pi.mode = conf.Mode
	pi.concurrency = conf.Concurrency
	pi.rules = make([]Rule[R], 0)
	pi.filters = make([]Filter[R], 0)
	pi.lists = make(map[string][]string)
	pi.macroCtxs = make(map[string]parser.IExpressionContext)
	pi.out = out
	//pi.ah = NewActionHandler(conf)
	return pi
}

// StartWorkers creates the worker pool.
func (pi *PolicyInterpreter[R]) StartWorkers() {
	logger.Trace.Printf("Starting policy engine's thread pool with %d workers", pi.concurrency)
	pi.workerCh = make(chan *R, pi.concurrency)
	pi.wg = new(sync.WaitGroup)
	pi.wg.Add(pi.concurrency)
	for i := 0; i < pi.concurrency; i++ {
		go pi.worker()
	}
}

// StopWorkers stops the worker pool and waits for all tasks to finish.
func (pi *PolicyInterpreter[R]) StopWorkers() {
	logger.Trace.Println("Stopping policy engine's thread pool")
	close(pi.workerCh)
	pi.wg.Wait()
}

// Compile parses and interprets an input policy defined in path.
func (pi *PolicyInterpreter[R]) compile(path string) error {
	// Setup the input
	is, err := antlr.NewFileStream(path)
	if err != nil {
		logger.Error.Println("Error reading policy from path", path)
		return err
	}

	// Create the Lexer
	lexerErrors := &errorhandler.SfplErrorListener{}
	lexer := parser.NewSfplLexer(is)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrors)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	// Create the Parser
	parserErrors := &errorhandler.SfplErrorListener{}
	p := parser.NewSfplParser(stream)
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrors)

	// Pre-processing (to deal with usage before definitions of macros and lists)
	antlr.ParseTreeWalkerDefault.Walk(pi, p.Defs())
	p.GetInputStream().Seek(0)

	// Parse the policy
	antlr.ParseTreeWalkerDefault.Walk(pi, p.Policy())

	errFound := false
	if len(lexerErrors.Errors) > 0 {
		logger.Error.Printf("Lexer %d errors found\n", len(lexerErrors.Errors))
		for _, e := range lexerErrors.Errors {
			logger.Error.Println("\t", e.Error())
		}
		errFound = true
	}
	if len(parserErrors.Errors) > 0 {
		logger.Error.Printf("Parser %d errors found\n", len(parserErrors.Errors))
		for _, e := range parserErrors.Errors {
			logger.Error.Println("\t", e.Error())
		}
		errFound = true
	}

	if errFound {
		return errors.New("errors found during compilation of policies. check logs for detail")
	}

	return nil
}

// Compile parses and interprets a set of input policies defined in paths.
func (pi *PolicyInterpreter[R]) Compile(paths ...string) error {
	for _, path := range paths {
		logger.Trace.Println("Parsing policy file ", path)
		if err := pi.compile(path); err != nil {
			return err
		}
	}
	//pi.ah.CheckActions(pi.rules)
	return nil
}

// ProcessAsync queues the record for processing in the worker pool.
func (pi *PolicyInterpreter[R]) ProcessAsync(r *R) {
	pi.workerCh <- r
}

// Asynchronous worker thread: apply all compiled policies, enrich matching records, and send records downstream.
func (pi *PolicyInterpreter[R]) worker() {
	for {
		// Fetch record
		r, ok := <-pi.workerCh
		if !ok {
			logger.Trace.Println("Worker channel closed. Shutting down.")
			break
		}

		// Drop record if any drop rule applied.
		if pi.EvalFilters(r) {
			continue
		}

		// Enrich mode is non-blocking: Push record even if no rule matches
		match := (pi.mode == EnrichMode)

		// Apply rules
		for _, rule := range pi.rules {
			// if rule.Enabled && rule.isApplicable(r) && rule.condition.Eval(r) {
			if rule.Enabled && rule.condition.Eval(*r) {
				// r.Ctx.SetAlert(pi.mode == AlertMode)
				// r.Ctx.AddRule(rule)
				//pi.ah.HandleActions(rule, r)
				match = true
			}
		}

		// Push record if a rule matches (or if mode is enrich)
		if match && pi.out != nil {
			pi.out(r)
		}
	}
	pi.wg.Done()
}

// Process executes all compiled policies against record r.
func (pi *PolicyInterpreter[R]) Process(r *R) *R {
	// Drop record if any drop rule applies
	if pi.EvalFilters(r) {
		return nil
	}

	// Enrich mode is non-blocking: Push record even if no rule matches
	match := (pi.mode == EnrichMode)

	for _, rule := range pi.rules {
		// if rule.Enabled && rule.isApplicable(r) && rule.condition.Eval(r) {
		if rule.Enabled && rule.condition.Eval(*r) {
			// r.Ctx.SetAlert(pi.mode == AlertMode)
			// r.Ctx.AddRule(rule)
			//pi.ah.HandleActions(rule, r)
			match = true
		}
	}

	// Push record if a rule matched (or if we are in enrich mode)
	if match {
		return r
	}
	return nil
}

// EvalFilters executes compiled policy filters against record r.
func (pi *PolicyInterpreter[R]) EvalFilters(r *R) bool {
	for _, f := range pi.filters {
		if f.Enabled && f.condition.Eval(*r) {
			return true
		}
	}
	return false
}
