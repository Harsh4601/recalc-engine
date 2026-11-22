package recalc

import (
	"math"
	"strconv"
	"strings"
)

// evalCall dispatches a function call to its implementation.
func (e *Engine) evalCall(nd CallNode, eval func([2]int) Value) Value {
	switch nd.Name {
	case "IF":
		return e.fnIf(nd.Args, eval)
	case "IFERROR":
		return e.fnIfError(nd.Args, eval)
	case "AND":
		return e.fnAndOr(nd.Args, eval, true)
	case "OR":
		return e.fnAndOr(nd.Args, eval, false)
	case "NOT":
		return e.fnNot(nd.Args, eval)
	case "SUM", "AVERAGE", "MIN", "MAX", "COUNT":
		return e.fnAgg(nd.Args, eval, nd.Name)
	case "ABS":
		return e.fnNumeric1(nd.Args, eval, math.Abs)
	case "ROUND":
		return e.fnRound(nd.Args, eval)
	case "LEN":
		return e.fnLen(nd.Args, eval)
	case "LEFT":
		return e.fnLeftRight(nd.Args, eval, true)
	case "RIGHT":
		return e.fnLeftRight(nd.Args, eval, false)
	case "MID":
		return e.fnMid(nd.Args, eval)
	case "CONCAT":
		return e.fnConcat(nd.Args, eval)
	case "ISERROR":
		return e.fnIsError(nd.Args, eval)
	case "ISNUMBER":
		return e.fnIsNumber(nd.Args, eval)
	}
	return Error(ErrName)
}

// rangeCells lists the cell keys covered by a range, normalizing corner order.
func (e *Engine) rangeCells(rn RangeNode) [][2]int {
	c1, c2 := rn.Start.Col, rn.End.Col
	r1, r2 := rn.Start.Row, rn.End.Row
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	var out [][2]int
	for row := r1; row <= r2; row++ {
		for col := c1; col <= c2; col++ {
			out = append(out, [2]int{col, row})
		}
	}
	return out
}

func (e *Engine) fnIf(args []Node, eval func([2]int) Value) Value {
	if len(args) < 2 || len(args) > 3 {
		return Error(ErrValue)
	}
	cond := e.evalNode(args[0], eval)
	if cond.Kind == KindError {
		return cond
	}
	b, code := toBool(cond)
	if code != ErrNone {
		return Error(code)
	}
	if b {
		return e.evalNode(args[1], eval)
	}
	if len(args) == 3 {
		return e.evalNode(args[2], eval)
	}
	return Boolean(false)
}

func (e *Engine) fnIfError(args []Node, eval func([2]int) Value) Value {
	if len(args) != 2 {
		return Error(ErrValue)
	}
	v := e.evalNode(args[0], eval)
	if v.Kind == KindError {
		return e.evalNode(args[1], eval)
	}
	return v
}

func (e *Engine) fnAndOr(args []Node, eval func([2]int) Value, isAnd bool) Value {
	var errAcc Value
	result := isAnd
	seen := false
	process := func(v Value) {
		if v.Kind == KindError {
			errAcc = higherError(errAcc, v)
			return
		}
		b, code := toBool(v)
		if code != ErrNone {
			errAcc = higherError(errAcc, Error(code))
			return
		}
		seen = true
		if isAnd {
			result = result && b
		} else {
			result = result || b
		}
	}
	for _, a := range args {
		if rn, ok := a.(RangeNode); ok {
			for _, k := range e.rangeCells(rn) {
				v := eval(k)
				if v.Kind == KindBlank {
					continue
				}
				process(v)
			}
			continue
		}
		process(e.evalNode(a, eval))
	}
	if errAcc.Kind == KindError {
		return errAcc
	}
	if !seen {
		return Error(ErrValue)
	}
	return Boolean(result)
}

func (e *Engine) fnNot(args []Node, eval func([2]int) Value) Value {
	if len(args) != 1 {
		return Error(ErrValue)
	}
	v := e.evalNode(args[0], eval)
	if v.Kind == KindError {
		return v
	}
	b, code := toBool(v)
	if code != ErrNone {
		return Error(code)
	}
	return Boolean(!b)
}

func (e *Engine) fnAgg(args []Node, eval func([2]int) Value, kind string) Value {
	var nums []float64
	var errAcc Value
	add := func(v Value) {
		if v.Kind == KindError {
			errAcc = higherError(errAcc, v)
			return
		}
		if v.Kind == KindNumber {
			nums = append(nums, v.Num)
		}
	}
	for _, a := range args {
		if rn, ok := a.(RangeNode); ok {
			// Inside a range, only numbers count; text/blank/bool are ignored.
			for _, k := range e.rangeCells(rn) {
				add(eval(k))
			}
			continue
		}
		v := e.evalNode(a, eval)
		if v.Kind == KindError {
			errAcc = higherError(errAcc, v)
			continue
		}
		switch v.Kind {
		case KindNumber:
			nums = append(nums, v.Num)
		case KindBool:
			if v.Bool {
				nums = append(nums, 1)
			} else {
				nums = append(nums, 0)
			}
		case KindBlank:
			// ignored
		case KindText:
			if f, err := strconv.ParseFloat(strings.TrimSpace(v.Text), 64); err == nil {
				nums = append(nums, f)
			} else {
				errAcc = higherError(errAcc, Error(ErrValue))
			}
		}
	}
	if errAcc.Kind == KindError {
		return errAcc
	}
	switch kind {
	case "SUM":
		s := 0.0
		for _, x := range nums {
			s += x
		}
		return Num(s)
	case "COUNT":
		return Num(float64(len(nums)))
	case "AVERAGE":
		if len(nums) == 0 {
			return Error(ErrDiv0)
		}
		s := 0.0
		for _, x := range nums {
			s += x
		}
		return Num(s / float64(len(nums)))
	case "MIN":
		if len(nums) == 0 {
			return Num(0)
		}
		m := nums[0]
		for _, x := range nums {
			if x < m {
				m = x
			}
		}
		return Num(m)
	case "MAX":
		if len(nums) == 0 {
			return Num(0)
		}
		m := nums[0]
		for _, x := range nums {
			if x > m {
				m = x
			}
		}
		return Num(m)
	}
	return Error(ErrValue)
}

func (e *Engine) fnNumeric1(args []Node, eval func([2]int) Value, f func(float64) float64) Value {
	if len(args) != 1 {
		return Error(ErrValue)
	}
	v := e.evalNode(args[0], eval)
	if v.Kind == KindError {
		return v
	}
	n, code := toNumber(v)
	if code != ErrNone {
		return Error(code)
	}
	return Num(f(n))
}

func (e *Engine) fnRound(args []Node, eval func([2]int) Value) Value {
	if len(args) != 2 {
		return Error(ErrValue)
	}
	v := e.evalNode(args[0], eval)
	if v.Kind == KindError {
		return v
	}
	d := e.evalNode(args[1], eval)
	if d.Kind == KindError {
		return d
	}
	n, code := toNumber(v)
	if code != ErrNone {
		return Error(code)
	}
	digits, code2 := toNumber(d)
	if code2 != ErrNone {
		return Error(code2)
	}
	f := math.Pow(10, digits)
	return Num(math.Round(n*f) / f)
}

func (e *Engine) fnLen(args []Node, eval func([2]int) Value) Value {
	if len(args) != 1 {
		return Error(ErrValue)
	}
	v := e.evalNode(args[0], eval)
	if v.Kind == KindError {
		return v
	}
	return Num(float64(len([]rune(toText(v)))))
}

func (e *Engine) fnLeftRight(args []Node, eval func([2]int) Value, left bool) Value {
	if len(args) < 1 || len(args) > 2 {
		return Error(ErrValue)
	}
	v := e.evalNode(args[0], eval)
	if v.Kind == KindError {
		return v
	}
	k := 1
	if len(args) == 2 {
		d := e.evalNode(args[1], eval)
		if d.Kind == KindError {
			return d
		}
		n, code := toNumber(d)
		if code != ErrNone {
			return Error(code)
		}
		k = int(n)
	}
	if k < 0 {
		return Error(ErrValue)
	}
	r := []rune(toText(v))
	if k > len(r) {
		k = len(r)
	}
	if left {
		return TextVal(string(r[:k]))
	}
	return TextVal(string(r[len(r)-k:]))
}

func (e *Engine) fnMid(args []Node, eval func([2]int) Value) Value {
	if len(args) != 3 {
		return Error(ErrValue)
	}
	v := e.evalNode(args[0], eval)
	if v.Kind == KindError {
		return v
	}
	sv := e.evalNode(args[1], eval)
	if sv.Kind == KindError {
		return sv
	}
	lv := e.evalNode(args[2], eval)
	if lv.Kind == KindError {
		return lv
	}
	start, code := toNumber(sv)
	if code != ErrNone {
		return Error(code)
	}
	length, code2 := toNumber(lv)
	if code2 != ErrNone {
		return Error(code2)
	}
	if start < 1 || length < 0 {
		return Error(ErrValue)
	}
	r := []rune(toText(v))
	from := int(start) - 1
	if from >= len(r) {
		return TextVal("")
	}
	to := from + int(length)
	if to > len(r) {
		to = len(r)
	}
	return TextVal(string(r[from:to]))
}

func (e *Engine) fnConcat(args []Node, eval func([2]int) Value) Value {
	var sb strings.Builder
	for _, a := range args {
		if rn, ok := a.(RangeNode); ok {
			for _, k := range e.rangeCells(rn) {
				v := eval(k)
				if v.Kind == KindError {
					return v
				}
				sb.WriteString(toText(v))
			}
			continue
		}
		v := e.evalNode(a, eval)
		if v.Kind == KindError {
			return v
		}
		sb.WriteString(toText(v))
	}
	return TextVal(sb.String())
}

func (e *Engine) fnIsError(args []Node, eval func([2]int) Value) Value {
	if len(args) != 1 {
		return Error(ErrValue)
	}
	v := e.evalNode(args[0], eval)
	return Boolean(v.Kind == KindError)
}

func (e *Engine) fnIsNumber(args []Node, eval func([2]int) Value) Value {
	if len(args) != 1 {
		return Error(ErrValue)
	}
	v := e.evalNode(args[0], eval)
	return Boolean(v.Kind == KindNumber)
}
