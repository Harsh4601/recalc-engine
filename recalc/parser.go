package recalc

import (
	"fmt"
	"strings"
)

// parseFormula parses a formula body (the text after '=') into an AST.
func parseFormula(s string) (Node, error) {
	toks, err := lex(s)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	n, err := p.parseExpr(1)
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tEOF {
		return nil, fmt.Errorf("unexpected token %q", p.peek().text)
	}
	return n, nil
}

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() token { return p.toks[p.pos] }
func (p *parser) next() token { t := p.toks[p.pos]; p.pos++; return t }

func (p *parser) isOp(s string) bool {
	t := p.peek()
	return t.kind == tOp && t.text == s
}

func binPrec(op string) int {
	switch op {
	case "=", "<>", "<", "<=", ">", ">=":
		return 1
	case "&":
		return 2
	case "+", "-":
		return 3
	case "*", "/":
		return 4
	case "^":
		return 5
	}
	return -1
}

func (p *parser) parseExpr(min int) (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.kind != tOp {
			break
		}
		prec := binPrec(t.text)
		if prec < min {
			break
		}
		op := p.next().text
		nextMin := prec + 1
		if op == "^" {
			nextMin = prec // right associative
		}
		right, err := p.parseExpr(nextMin)
		if err != nil {
			return nil, err
		}
		left = BinaryNode{Op: op, L: left, R: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (Node, error) {
	t := p.peek()
	if t.kind == tOp && (t.text == "-" || t.text == "+") {
		p.next()
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return UnaryNode{Op: t.text[0], X: x}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Node, error) {
	t := p.next()
	switch t.kind {
	case tNumber:
		return NumberLit{Val: t.num}, nil
	case tString:
		return StringLit{Val: t.text}, nil
	case tName:
		return p.parseName(t.text)
	case tOp:
		if t.text == "(" {
			n, err := p.parseExpr(1)
			if err != nil {
				return nil, err
			}
			if !p.isOp(")") {
				return nil, fmt.Errorf("expected )")
			}
			p.next()
			return n, nil
		}
	}
	return nil, fmt.Errorf("unexpected token %q", t.text)
}

func (p *parser) parseName(name string) (Node, error) {
	up := strings.ToUpper(name)
	// Function call?
	if p.isOp("(") {
		p.next()
		var args []Node
		if !p.isOp(")") {
			for {
				a, err := p.parseExpr(1)
				if err != nil {
					return nil, err
				}
				args = append(args, a)
				if p.isOp(",") {
					p.next()
					continue
				}
				break
			}
		}
		if !p.isOp(")") {
			return nil, fmt.Errorf("expected ) in call to %s", up)
		}
		p.next()
		return CallNode{Name: up, Args: args}, nil
	}
	if up == "TRUE" {
		return BoolLit{Val: true}, nil
	}
	if up == "FALSE" {
		return BoolLit{Val: false}, nil
	}
	ref, ok := parseA1(up)
	if !ok {
		return nil, fmt.Errorf("bad reference %q", name)
	}
	if p.isOp(":") {
		p.next()
		t2 := p.next()
		if t2.kind != tName {
			return nil, fmt.Errorf("expected cell reference after ':'")
		}
		ref2, ok := parseA1(strings.ToUpper(t2.text))
		if !ok {
			return nil, fmt.Errorf("bad reference %q", t2.text)
		}
		return RangeNode{Start: ref, End: ref2}, nil
	}
	return RefNode{Ref: ref}, nil
}
