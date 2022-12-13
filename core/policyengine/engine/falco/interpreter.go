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

// Package falco implements a rules engine based on falco rules.
package falco

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

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

	ops engine.Operations[R]

	// Parsed rule and filter object maps
	rules   []engine.Rule
	filters []engine.Filter

	// Accessory parsing maps
	lists     map[string][]string
	macroCtxs map[string]parser.IExpressionContext

	// Action Handler
	//ah *ActionHandler
}

// NewPolicyInterpreter constructs a new interpreter instance.
func NewPolicyInterpreter[R any](conf engine.Config, out func(*R)) *PolicyInterpreter[R] {
	pi := new(PolicyInterpreter[R])
	pi.ops = Operations{}
	pi.rules = make([]engine.Rule, 0)
	pi.filters = make([]engine.Filter, 0)
	pi.lists = make(map[string][]string)
	pi.macroCtxs = make(map[string]parser.IExpressionContext)
	pi.out = out
	//pi.ah = NewActionHandler(conf)
	return pi
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

// ExitList is called when production list is exited.
func (pi *PolicyInterpreter[R]) ExitPlist(ctx *parser.PlistContext) {
	logger.Trace.Println("Parsing list ", ctx.GetText())
	pi.lists[ctx.ID().GetText()] = pi.extractListFromItems(ctx.Items())
}

// ExitMacro is called when production macro is exited.
func (pi *PolicyInterpreter[R]) ExitPmacro(ctx *parser.PmacroContext) {
	logger.Trace.Println("Parsing macro ", ctx.GetText())
	pi.macroCtxs[ctx.ID().GetText()] = ctx.Expression()
}

// ExitFilter is called when production filter is exited.
func (pi *PolicyInterpreter[R]) ExitPfilter(ctx *parser.PfilterContext) {
	logger.Trace.Println("Parsing filter ", ctx.GetText())
	f := engine.Filter{
		Name:      ctx.ID().GetText(),
		condition: pi.visitExpression(ctx.Expression()),
		Enabled:   ctx.ENABLED() == nil || pi.getEnabledFlag(ctx.Enabled()),
	}
	pi.filters = append(pi.filters, f)
}

// ExitFilter is called when production filter is exited.
func (pi *PolicyInterpreter[R]) ExitPrule(ctx *parser.PruleContext) {
	logger.Trace.Println("Parsing rule ", ctx.GetText())
	r := engine.Rule{
		Name:      pi.getOffChannelText(ctx.Text(0)),
		Desc:      pi.getOffChannelText(ctx.Text(1)),
		condition: pi.visitExpression(ctx.Expression()),
		Actions:   pi.getActions(ctx),
		Tags:      pi.getTags(ctx),
		Priority:  pi.getPriority(ctx),
		Prefilter: pi.getPrefilter(ctx),
		Enabled:   ctx.ENABLED(0) == nil || pi.getEnabledFlag(ctx.Enabled(0)),
	}
	pi.rules = append(pi.rules, r)
}

func (pi *PolicyInterpreter[R]) getEnabledFlag(ctx parser.IEnabledContext) bool {
	flag := engine.trimBoundingQuotes(ctx.GetText())
	if b, err := strconv.ParseBool(flag); err == nil {
		return b
	}
	logger.Warn.Println("Unrecognized enabled flag: ", flag)
	return true
}

func (pi *PolicyInterpreter[R]) getOffChannelText(ctx parser.ITextContext) string {
	a := ctx.GetStart().GetStart()
	b := ctx.GetStop().GetStop()
	interval := antlr.Interval{Start: a, Stop: b}
	return ctx.GetStart().GetInputStream().GetTextFromInterval(&interval)
}

func (pi *PolicyInterpreter[R]) getTags(ctx *parser.PruleContext) []engine.EnrichmentTag {
	var tags = make([]engine.EnrichmentTag, 0)
	ictx := ctx.Tags(0)
	if ictx != nil {
		return append(tags, pi.extractTags(ictx))
	}
	return tags
}

func (pi *PolicyInterpreter[R]) getPrefilter(ctx *parser.PruleContext) []string {
	var pfs = make([]string, 0)
	ictx := ctx.Prefilter(0)
	if ictx != nil {
		return append(pfs, pi.extractList(ictx.GetText())...)
	}
	return pfs
}

func (pi *PolicyInterpreter[R]) getPriority(ctx *parser.PruleContext) engine.Priority {
	ictx := ctx.Severity(0)
	if ictx != nil {
		p := ictx.GetText()
		switch strings.ToLower(p) {
		case Low.String():
			return Low
		case Medium.String():
			return Medium
		case High.String():
			return High
		case FPriorityDebug:
			return Low
		case FPriorityInfo:
			return Low
		case FPriorityInformational:
			return Low
		case FPriorityNotice:
			return Low
		case FPriorityWarning:
			return Medium
		case FPriorityError:
			return High
		case FPriorityCritical:
			return High
		case FPriorityEmergency:
			return High
		default:
			logger.Warn.Printf("Unrecognized priority value %s. Deferring to %s\n", p, Low.String())
		}
	}
	return Low
}

func (pi *PolicyInterpreter[R]) getActions(ctx *parser.PruleContext) []string {
	var actions []string
	ictx := ctx.Actions(0)
	if ictx != nil {
		return append(actions, pi.extractActions(ictx)...)
	}
	return actions
}

func (pi *PolicyInterpreter[R]) extractList(str string) []string {
	var items []string
	for _, i := range strings.Split(itemsre.ReplaceAllString(str, "$2"), LISTSEP) {
		items = append(items, engine.trimBoundingQuotes(i))
	}
	return items
}

func (pi *PolicyInterpreter[R]) extractListFromItems(ctx parser.IItemsContext) []string {
	if ctx != nil {
		return pi.extractList(ctx.GetText())
	}
	return []string{}
}

func (pi *PolicyInterpreter[R]) extractTags(ctx parser.ITagsContext) []string {
	if ctx != nil {
		return pi.extractList(ctx.GetText())
	}
	return []string{}
}

func (pi *PolicyInterpreter[R]) extractActions(ctx parser.IActionsContext) []string {
	if ctx != nil {
		return pi.extractList(ctx.GetText())
	}
	return []string{}
}

func (pi *PolicyInterpreter[R]) extractListFromAtoms(ctxs []parser.IAtomContext) []string {
	s := []string{}
	for _, v := range ctxs {
		s = append(s, pi.reduceList(v.GetText())...)
	}
	return s
}

func (pi *PolicyInterpreter[R]) reduceList(sl string) []string {
	s := []string{}
	if l, ok := pi.lists[sl]; ok {
		for _, v := range l {
			s = append(s, pi.reduceList(v)...)
		}
	} else {
		s = append(s, engine.trimBoundingQuotes(sl))
	}
	return s
}

func (pi *PolicyInterpreter[R]) visitExpression(ctx parser.IExpressionContext) engine.Criterion[R] {
	orCtx := ctx.GetChild(0).(parser.IOr_expressionContext)
	orPreds := make([]engine.Criterion[R], 0)
	for _, andCtx := range orCtx.GetChildren() {
		if andCtx.GetChildCount() > 0 {
			andPreds := make([]engine.Criterion[R], 0)
			for _, termCtx := range andCtx.GetChildren() {
				t, isTermCtx := termCtx.(parser.ITermContext)
				if isTermCtx {
					c := pi.visitTerm(t)
					andPreds = append(andPreds, c)
				}
			}
			orPreds = append(orPreds, All(andPreds))
		}
	}
	return Any(orPreds)
}

func (pi *PolicyInterpreter[R]) visitTerm(ctx parser.ITermContext) engine.Criterion[R] {
	termCtx := ctx.(*parser.TermContext)
	if termCtx.Variable() != nil {
		if m, ok := pi.macroCtxs[termCtx.GetText()]; ok {
			return pi.visitExpression(m)
		}
		logger.Error.Println("Unrecognized reference ", termCtx.GetText())
	} else if termCtx.NOT() != nil {
		return pi.visitTerm(termCtx.GetChild(1).(parser.ITermContext)).Not()
	} else if opCtx, ok := termCtx.Unary_operator().(*parser.Unary_operatorContext); ok {
		lop := termCtx.Atom(0).(*parser.AtomContext).GetText()
		if opCtx.EXISTS() != nil {
			return Exists(lop)
		}
		logger.Error.Println("Unrecognized unary operator ", opCtx.GetText())
	} else if opCtx, ok := termCtx.Binary_operator().(*parser.Binary_operatorContext); ok {
		lop := termCtx.Atom(0).(*parser.AtomContext).GetText()
		rop := termCtx.Atom(1).(*parser.AtomContext).GetText()
		if opCtx.CONTAINS() != nil {
			return Contains(lop, rop)
		} else if opCtx.ICONTAINS() != nil {
			return IContains(lop, rop)
		} else if opCtx.STARTSWITH() != nil {
			return StartsWith(lop, rop)
		} else if opCtx.ENDSWITH() != nil {
			return EndsWith(lop, rop)
		} else if opCtx.EQ() != nil {
			return Eq(lop, rop)
		} else if opCtx.NEQ() != nil {
			return NEq(lop, rop)
		} else if opCtx.GT() != nil {
			return Gt(lop, rop)
		} else if opCtx.GE() != nil {
			return Ge(lop, rop)
		} else if opCtx.LT() != nil {
			return Lt(lop, rop)
		} else if opCtx.LE() != nil {
			return Le(lop, rop)
		}
		logger.Error.Println("Unrecognized binary operator ", opCtx.GetText())
	} else if termCtx.Expression() != nil {
		return pi.visitExpression(termCtx.Expression())
	} else if termCtx.IN() != nil {
		lop := termCtx.Atom(0).(*parser.AtomContext).GetText()
		rop := termCtx.AllAtom()[1:]
		return In(lop, pi.extractListFromAtoms(rop))
	} else if termCtx.PMATCH() != nil {
		lop := termCtx.Atom(0).(*parser.AtomContext).GetText()
		rop := termCtx.AllAtom()[1:]
		return PMatch(lop, pi.extractListFromAtoms(rop))
	} else {
		logger.Warn.Println("Unrecognized term ", termCtx.GetText())
	}
	return False
}
