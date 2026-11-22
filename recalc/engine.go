package recalc

import (
	"math"
	"sort"
	"strings"
)

// recalc recomputes every cell. Literal cells are resolved first, then every
// formula cell is evaluated with a depth-first walk that memoizes results and
// marks cells participating in a reference cycle as #CYCLE!.
func (e *Engine) recalc() {
	for _, cd := range e.cells {
		if !cd.formula {
			cd.val = parseLiteral(cd.raw)
		}
	}

	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)
	state := make(map[[2]int]int, len(e.cells))

	var eval func(k [2]int) Value
	eval = func(k [2]int) Value {
		cd, ok := e.cells[k]
		if !ok {
			return Blank()
		}
		if !cd.formula {
			return cd.val
		}
		switch state[k] {
		case done:
			return cd.val
		case visiting:
			cd.val = Error(ErrCycle)
			return cd.val
		}
		state[k] = visiting
		var v Value
		if cd.parseErr {
			v = Error(ErrName)
		} else {
			v = e.evalNode(cd.ast, eval)
		}
		cd.val = v
		state[k] = done
		return v
	}

	keys := make([][2]int, 0, len(e.cells))
	for k := range e.cells {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][1] != keys[j][1] {
			return keys[i][1] < keys[j][1]
		}
		return keys[i][0] < keys[j][0]
	})
	for _, k := range keys {
		if e.cells[k].formula {
			eval(k)
		}
	}
	e.dirty = false
}

// evalNode evaluates a single AST node. Reference lookups go through eval so
// that memoization and cycle detection stay centralized.
func (e *Engine) evalNode(n Node, eval func([2]int) Value) Value {
	switch nd := n.(type) {
	case NumberLit:
		return Num(nd.Val)
	case StringLit:
		return TextVal(nd.Val)
	case BoolLit:
		return Boolean(nd.Val)
	case ErrorNode:
		return Error(nd.Code)
	case RefNode:
		return eval(nd.Ref.key())
	case RangeNode:
		// A bare range used in scalar context is an error.
		return Error(ErrValue)
	case UnaryNode:
		x := e.evalNode(nd.X, eval)
		if x.Kind == KindError {
			return x
		}
		num, code := toNumber(x)
		if code != ErrNone {
			return Error(code)
		}
		if nd.Op == '-' {
			return Num(-num)
		}
		return Num(num)
	case BinaryNode:
		return e.evalBinary(nd, eval)
	case CallNode:
		return e.evalCall(nd, eval)
	}
	return Error(ErrValue)
}

func (e *Engine) evalBinary(nd BinaryNode, eval func([2]int) Value) Value {
	l := e.evalNode(nd.L, eval)
	r := e.evalNode(nd.R, eval)
	if l.Kind == KindError || r.Kind == KindError {
		return higherError(l, r)
	}
	switch nd.Op {
	case "&":
		return TextVal(toText(l) + toText(r))
	case "=", "<>", "<", "<=", ">", ">=":
		return compare(l, r, nd.Op)
	}
	ln, e1 := toNumber(l)
	rn, e2 := toNumber(r)
	if e1 != ErrNone {
		return Error(e1)
	}
	if e2 != ErrNone {
		return Error(e2)
	}
	switch nd.Op {
	case "+":
		return Num(ln + rn)
	case "-":
		return Num(ln - rn)
	case "*":
		return Num(ln * rn)
	case "/":
		if rn == 0 {
			return Error(ErrDiv0)
		}
		return Num(ln / rn)
	case "^":
		return Num(math.Pow(ln, rn))
	}
	return Error(ErrValue)
}

// typeRank orders the value kinds for cross-type comparison:
// number/blank < text < bool.
func typeRank(v Value) int {
	switch v.Kind {
	case KindNumber, KindBlank:
		return 0
	case KindText:
		return 1
	case KindBool:
		return 2
	}
	return 3
}

func compare(l, r Value, op string) Value {
	var c int
	if typeRank(l) == typeRank(r) {
		switch {
		case typeRank(l) == 0:
			ln, _ := toNumber(l)
			rn, _ := toNumber(r)
			switch {
			case ln < rn:
				c = -1
			case ln > rn:
				c = 1
			}
		case l.Kind == KindText:
			c = strings.Compare(strings.ToUpper(l.Text), strings.ToUpper(r.Text))
		case l.Kind == KindBool:
			switch {
			case !l.Bool && r.Bool:
				c = -1
			case l.Bool && !r.Bool:
				c = 1
			}
		}
	} else if typeRank(l) < typeRank(r) {
		c = -1
	} else {
		c = 1
	}
	var res bool
	switch op {
	case "=":
		res = c == 0
	case "<>":
		res = c != 0
	case "<":
		res = c < 0
	case "<=":
		res = c <= 0
	case ">":
		res = c > 0
	case ">=":
		res = c >= 0
	}
	return Boolean(res)
}
