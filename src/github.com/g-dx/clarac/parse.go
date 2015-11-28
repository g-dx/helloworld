package main

import (
	"errors"
	"fmt"
	"strings"
	"strconv"
	"github.com/g-dx/clarac/lex"
)

const (
	syntaxErrMsg = "%v:%d:%d: syntax error, Unexpected '%v', Expecting: '%v'"
	errRedeclaredMsg = "%v:%d:%d: error, '%v' redeclared"
	errUndefinedMsg = "%v:%d:%d: error, '%v' undefined"
	errNotFuncMsg = "%v:%d:%d: error, '%v' is not a function"
	errArgCountMsg = "%v:%d:%d: error, wrong argument count to call '%v'"
)

type Parser struct {
	pos    int
	tokens []*lex.Token
	errs   []error
	discard bool // Are we in "discard" mode?
	symtab SymTab
	extra []*Node
}

var errUnexpectedEof = errors.New("Unexpected EOF")

func NewParser(tokens []*lex.Token, extra []*Node) *Parser {
	// Add any symbols from predefined nodes
	symtab := NewSymtab()
	for _, n := range extra {
		symtab.Define(n.sym)
	}
	return &Parser{tokens : tokens, symtab: symtab, extra : extra}
}

func (p *Parser) Parse() (errs []error, root *Node) {

	// Setup handler to recover from unexpected EOF
	defer p.onUnexpectedEof(&errs)

	// Create root & add extra nodes
	root = &Node{op : opRoot}
	for _, n := range p.extra {
		root.Add(n)
	}

	// Loop over stream
	for p.isNot(lex.EOF) {
		if p.is(lex.Fn) {
			root.Add(p.fnDeclaration())
		} else {
			p.syntaxError(lex.Fn, lex.EOF)
			p.next()
		}
	}
	p.need(lex.EOF)
	return p.errs, root
}

func (p *Parser) fnDeclaration() *Node {

	// Match declaration
	p.need(lex.Fn)
	name := p.need(lex.Identifier)
	p.need(lex.LParen)
	p.need(lex.RParen)
	p.need(lex.LBrace)

	// Match calls
	var fnCalls []*Node
	for p.isNot(lex.RBrace) {
		if p.is(lex.Identifier) {
			fnCalls = append(fnCalls, p.fnCall())
		} else {
			p.syntaxError(lex.Identifier, lex.RBrace)
			p.next()
		}
	}
	p.need(lex.RBrace)
	return p.fnDclNode(name, fnCalls)
}

func (p *Parser) fnDclNode(token *lex.Token, fnCalls []*Node) *Node {

	// Check symtab for redeclare
	sym, found := p.symtab.Resolve(symFnDecl, token.Val)
	if found {
		p.symbolError(errRedeclaredMsg, token)
	} else {
		sym = &Function{token.Val, 0, 0}
		p.symtab.Define(sym) // Functions don't take params yet
	}
	return &Node{token : token, stats : fnCalls, op : opFuncDcl, sym : sym}
}

func (p *Parser) fnCall() *Node {
	return p.fnCallNode(p.need(lex.Identifier), p.parseArgs())
}

func (p *Parser) fnCallNode(token *lex.Token, args []*Node) *Node {
	// TODO: TEMPORARY WORKAROUND!
	sym, _ := p.symtab.Resolve(symFnDecl, token.Val)
	return &Node{token : token, stats : args, op : opFuncCall, sym : sym}
}

// next()            - pull in next token
// is(t) bool        - t == current
// match(t) bool     - t == current, next() if true, false otherwise
// matches(...t) t   - t == current, next() if true, false otherwise
// need(t) void      - !match() scan to ??

func (p *Parser) need(k lex.Kind) *lex.Token {
	for !p.is(k) {
		fmt.Printf(lex.KindValues[p.tokens[p.pos].Kind])
		p.syntaxError(k)
		p.next()
	}
	p.discard = false
	return p.next()
}

func (p *Parser) isNot(kinds...lex.Kind) bool {
	for _, k := range kinds {
		if p.is(k) {
			return false
		}
	}
	return true
}

func (p *Parser) is(kinds...lex.Kind) bool {
	for _, kind := range kinds {
		if p.tokens[p.pos].Kind == kind {
			return true
		}
	}
	return false
}

func (p *Parser) match(k lex.Kind) bool {
	if p.is(k) {
		p.next()
		p.discard = false
		return true
	}
	return false
}

func (p *Parser) parseArgs() (n []*Node) {
	p.need(lex.LParen)
	if p.isNot(lex.EOF, lex.RParen) {
		n = append(n, p.parseArg())
		for p.match(lex.Comma) {
			n = append(n, p.parseArg())
		}
	}
	p.need(lex.RParen)
	return n
}

func (p *Parser) parseArg() (*Node) {
	switch p.tokens[p.pos].Kind {
	case lex.Integer:
		// Match first arg
		arg := p.next()
		// Define symbol
		sym, found := p.symtab.Resolve(symIntegerLit, arg.Val)
		if !found {
			i, err := strconv.ParseInt(arg.Val, 10, 64)
			if err != nil {
				panic(err) // Should never happen
			}
			sym = &IntegerLiteralSymbol{val : i }
			p.symtab.Define(sym)
		}
		return &Node{token : arg, op : opIntegerLit, sym : sym}
	case lex.String:

		// Match first arg
		arg := p.next()

		// Define symbol
		sym, found := p.symtab.Resolve(symStrLit, arg.Val)
		if !found {
			sym = &StringLiteralSymbol{val : arg.Val }
			p.symtab.Define(sym)
		}
		return &Node{token : arg, op : opStrLit, sym : sym}
	default:
		p.syntaxError(lex.Integer, lex.String)
		p.next()
		return nil
	}
}

func (p *Parser) next() *lex.Token {
	// Panic if unexpectedly no more input
	if (p.pos+1 >= len(p.tokens)) {
		fmt.Println("EoF!")
		panic(errUnexpectedEof)
	}
	token := p.tokens[p.pos]
	p.pos++
	return token
}

func (p *Parser) symbolError(err string, token *lex.Token) {
	p.errs = append(p.errs,
		errors.New(fmt.Sprintf(err,
			token.File,
			token.Line,
			token.Pos,
			token.Val)))
}

func (p *Parser) syntaxError(expected...lex.Kind) {
	if !p.discard {
		// Enable discard mode
		p.discard = true

		// Gather values
		expectedValues := make([]string, 0, 2)
		for _, v := range expected {
			expectedValues = append(expectedValues, lex.KindValues[v])
		}

		// Store error
		token := p.tokens[p.pos]
		p.errs = append(p.errs,
			errors.New(fmt.Sprintf(syntaxErrMsg,
				token.File,
				token.Line,
				token.Pos,
				p.tokens[p.pos].Val,
				strings.Join(expectedValues, "' or '"))))
	}
}

func (p *Parser) onUnexpectedEof(errs *[]error) {
	if r := recover(); r != nil {
		*errs = p.errs
		if r != errUnexpectedEof {
			panic(r)
		}
	}
}
