package search

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const (
	opAnd  = "AND"
	opOr   = "OR"
	opAdj  = "ADJ"
	opNear = "NEAR"
)

type Node interface {
	positiveTerms(negated bool, out map[string]struct{})
}

type TermNode struct {
	Term string
}

type NotNode struct {
	Child Node
}

type BinaryNode struct {
	Op     string
	Left   Node
	Right  Node
	Window uint32
}

func (n TermNode) positiveTerms(negated bool, out map[string]struct{}) {
	if negated {
		return
	}
	term := NormalizeTerm(n.Term)
	if term != "" {
		out[term] = struct{}{}
	}
}

func (n NotNode) positiveTerms(negated bool, out map[string]struct{}) {
	n.Child.positiveTerms(!negated, out)
}

func (n BinaryNode) positiveTerms(negated bool, out map[string]struct{}) {
	n.Left.positiveTerms(negated, out)
	n.Right.positiveTerms(negated, out)
}

type queryToken struct {
	kind string
	text string
}

type queryParser struct {
	tokens []queryToken
	pos    int
}

func ParseQuery(query string) (Node, error) {
	tokens := lexQuery(query)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty query")
	}
	p := queryParser{tokens: tokens}
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if !p.done() {
		return nil, fmt.Errorf("unexpected token %q", p.peek().text)
	}
	return node, nil
}

func lexQuery(query string) []queryToken {
	var tokens []queryToken
	var b strings.Builder
	for _, r := range query {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '/' || r == '_' || r == '-':
			b.WriteRune(r)
		case r == '(':
			flushQueryToken(&tokens, &b)
			tokens = append(tokens, queryToken{kind: "(", text: "("})
		case r == ')':
			flushQueryToken(&tokens, &b)
			tokens = append(tokens, queryToken{kind: ")", text: ")"})
		default:
			flushQueryToken(&tokens, &b)
		}
	}
	flushQueryToken(&tokens, &b)
	return tokens
}

func flushQueryToken(tokens *[]queryToken, b *strings.Builder) {
	if b.Len() == 0 {
		return
	}
	text := b.String()
	*tokens = append(*tokens, queryToken{kind: tokenKind(text), text: text})
	b.Reset()
}

func tokenKind(text string) string {
	upper := strings.ToUpper(text)
	switch {
	case upper == "AND" || upper == "ANT":
		return opAnd
	case upper == "OR":
		return opOr
	case upper == "NOT":
		return "NOT"
	case upper == "ADJ" || upper == "EDGE":
		return opAdj
	case upper == "NEAR" || strings.HasPrefix(upper, "NEAR/"):
		return opNear
	default:
		return "WORD"
	}
}

func (p *queryParser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.match(opOr) {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = BinaryNode{Op: opOr, Left: left, Right: right}
	}
	return left, nil
}

func (p *queryParser) parseAnd() (Node, error) {
	left, err := p.parseNear()
	if err != nil {
		return nil, err
	}
	for {
		if p.match(opAnd) || p.startsPrimary() {
			right, err := p.parseNear()
			if err != nil {
				return nil, err
			}
			left = BinaryNode{Op: opAnd, Left: left, Right: right}
			continue
		}
		return left, nil
	}
}

func (p *queryParser) parseNear() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.peekKind(opAdj) || p.peekKind(opNear) {
		tok := p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		window := uint32(1)
		if tok.kind == opNear {
			window = nearWindow(tok.text)
		}
		left = BinaryNode{Op: tok.kind, Left: left, Right: right, Window: window}
	}
	return left, nil
}

func (p *queryParser) parseUnary() (Node, error) {
	if p.match("NOT") {
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return NotNode{Child: child}, nil
	}
	return p.parsePrimary()
}

func (p *queryParser) parsePrimary() (Node, error) {
	if p.match("(") {
		node, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if !p.match(")") {
			return nil, fmt.Errorf("missing closing parenthesis")
		}
		return node, nil
	}
	if p.done() {
		return nil, fmt.Errorf("unexpected end of query")
	}
	tok := p.advance()
	if tok.kind != "WORD" {
		return nil, fmt.Errorf("expected term, got %q", tok.text)
	}
	term := NormalizeTerm(tok.text)
	if term == "" {
		return nil, fmt.Errorf("empty term %q", tok.text)
	}
	return TermNode{Term: term}, nil
}

func nearWindow(text string) uint32 {
	upper := strings.ToUpper(text)
	if !strings.HasPrefix(upper, "NEAR/") {
		return 5
	}
	n, err := strconv.Atoi(upper[5:])
	if err != nil || n < 1 {
		return 5
	}
	return uint32(n)
}

func (p *queryParser) startsPrimary() bool {
	if p.done() {
		return false
	}
	kind := p.peek().kind
	return kind == "WORD" || kind == "NOT" || kind == "("
}

func (p *queryParser) match(kind string) bool {
	if !p.peekKind(kind) {
		return false
	}
	p.pos++
	return true
}

func (p *queryParser) peekKind(kind string) bool {
	return !p.done() && p.peek().kind == kind
}

func (p *queryParser) advance() queryToken {
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}

func (p *queryParser) peek() queryToken {
	return p.tokens[p.pos]
}

func (p *queryParser) done() bool {
	return p.pos >= len(p.tokens)
}
