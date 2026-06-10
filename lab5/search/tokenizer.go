package search

import (
	"strings"
	"unicode"
)

type Token struct {
	Term string
	Pos  uint32
}

func Tokenize(text string) []Token {
	var out []Token
	var b strings.Builder
	pos := uint32(0)
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		if b.Len() == 0 {
			continue
		}
		out = append(out, Token{Term: b.String(), Pos: pos})
		pos++
		b.Reset()
	}
	if b.Len() > 0 {
		out = append(out, Token{Term: b.String(), Pos: pos})
	}
	return out
}

func NormalizeTerm(term string) string {
	tokens := Tokenize(term)
	if len(tokens) == 0 {
		return ""
	}
	return tokens[0].Term
}
