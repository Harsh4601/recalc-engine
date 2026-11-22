package recalc

import (
	"fmt"
	"strconv"
	"strings"
)

type tokKind int

const (
	tEOF tokKind = iota
	tNumber
	tString
	tName
	tOp
)

type token struct {
	kind tokKind
	text string
	num  float64
}

// lex turns a formula body (without the leading '=') into tokens.
func lex(s string) ([]token, error) {
	var toks []token
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case (c >= '0' && c <= '9') || c == '.':
			j := i
			for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.') {
				j++
			}
			if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
				j++
				if j < len(s) && (s[j] == '+' || s[j] == '-') {
					j++
				}
				for j < len(s) && s[j] >= '0' && s[j] <= '9' {
					j++
				}
			}
			f, err := strconv.ParseFloat(s[i:j], 64)
			if err != nil {
				return nil, fmt.Errorf("bad number %q", s[i:j])
			}
			toks = append(toks, token{kind: tNumber, text: s[i:j], num: f})
			i = j
		case c == '"':
			j := i + 1
			var sb strings.Builder
			closed := false
			for j < len(s) {
				if s[j] == '"' {
					if j+1 < len(s) && s[j+1] == '"' {
						sb.WriteByte('"')
						j += 2
						continue
					}
					closed = true
					break
				}
				sb.WriteByte(s[j])
				j++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated string")
			}
			toks = append(toks, token{kind: tString, text: sb.String()})
			i = j + 1
		case c == '$' || c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'):
			j := i
			for j < len(s) {
				d := s[j]
				if d == '$' || d == '_' || (d >= 'A' && d <= 'Z') || (d >= 'a' && d <= 'z') || (d >= '0' && d <= '9') {
					j++
					continue
				}
				break
			}
			toks = append(toks, token{kind: tName, text: s[i:j]})
			i = j
		default:
			if i+1 < len(s) {
				two := s[i : i+2]
				if two == "<>" || two == "<=" || two == ">=" {
					toks = append(toks, token{kind: tOp, text: two})
					i += 2
					continue
				}
			}
			if strings.IndexByte("+-*/^&=<>(),:", c) >= 0 {
				toks = append(toks, token{kind: tOp, text: string(c)})
				i++
				continue
			}
			return nil, fmt.Errorf("unexpected character %q", string(c))
		}
	}
	toks = append(toks, token{kind: tEOF})
	return toks, nil
}
