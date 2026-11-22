package recalc

import (
	"math"
	"strconv"
	"strings"
)

// ValueKind is the dynamic type of a computed cell value.
type ValueKind int

const (
	KindBlank ValueKind = iota
	KindNumber
	KindText
	KindBool
	KindError
)

// ErrCode identifies a spreadsheet error value.
type ErrCode int

const (
	ErrNone  ErrCode = iota
	ErrName          // #NAME?
	ErrValue         // #VALUE!
	ErrDiv0          // #DIV/0!
	ErrRef           // #REF!
	ErrCycle         // #CYCLE!
)

// errPriority defines the aggregation priority of error values. When a range
// aggregate encounters several errors, the one with the highest priority wins.
// This ordering is part of the engine's specified semantics.
func errPriority(c ErrCode) int {
	switch c {
	case ErrCycle:
		return 5
	case ErrRef:
		return 4
	case ErrDiv0:
		return 3
	case ErrValue:
		return 2
	case ErrName:
		return 1
	}
	return 0
}

func errString(c ErrCode) string {
	switch c {
	case ErrName:
		return "#NAME?"
	case ErrValue:
		return "#VALUE!"
	case ErrDiv0:
		return "#DIV/0!"
	case ErrRef:
		return "#REF!"
	case ErrCycle:
		return "#CYCLE!"
	}
	return ""
}

func errFromString(s string) (ErrCode, bool) {
	switch s {
	case "#NAME?":
		return ErrName, true
	case "#VALUE!":
		return ErrValue, true
	case "#DIV/0!":
		return ErrDiv0, true
	case "#REF!":
		return ErrRef, true
	case "#CYCLE!":
		return ErrCycle, true
	}
	return ErrNone, false
}

// Value is a computed cell value.
type Value struct {
	Kind ValueKind
	Num  float64
	Text string
	Bool bool
	Err  ErrCode
}

func Blank() Value           { return Value{Kind: KindBlank} }
func Num(f float64) Value    { return Value{Kind: KindNumber, Num: f} }
func TextVal(s string) Value { return Value{Kind: KindText, Text: s} }
func Boolean(b bool) Value   { return Value{Kind: KindBool, Bool: b} }
func Error(c ErrCode) Value  { return Value{Kind: KindError, Err: c} }

// String renders the value in the engine's canonical textual form, which is
// what scenario expectations compare against.
func (v Value) String() string { return toText(v) }

func toText(v Value) string {
	switch v.Kind {
	case KindBlank:
		return ""
	case KindNumber:
		return formatNumber(v.Num)
	case KindText:
		return v.Text
	case KindBool:
		if v.Bool {
			return "TRUE"
		}
		return "FALSE"
	case KindError:
		return errString(v.Err)
	}
	return ""
}

// formatNumber renders a float the way the engine displays it: integers print
// without a decimal point, everything else uses Go's shortest round-trip form.
func formatNumber(f float64) string {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return "#NUM!"
	}
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// parseLiteral turns a raw (non-formula) cell input into a Value.
func parseLiteral(s string) Value {
	if s == "" {
		return Blank()
	}
	switch strings.ToUpper(s) {
	case "TRUE":
		return Boolean(true)
	case "FALSE":
		return Boolean(false)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return Num(f)
	}
	if code, ok := errFromString(s); ok {
		return Error(code)
	}
	return TextVal(s)
}

// higherError combines two values for error accumulation. A non-error operand
// never wins; between two errors the higher-priority code wins.
func higherError(a, b Value) Value {
	if a.Kind != KindError {
		return b
	}
	if b.Kind != KindError {
		return a
	}
	if errPriority(b.Err) > errPriority(a.Err) {
		return b
	}
	return a
}

// toNumber coerces a value to a number following the engine's coercion rules.
func toNumber(v Value) (float64, ErrCode) {
	switch v.Kind {
	case KindNumber:
		return v.Num, ErrNone
	case KindBool:
		if v.Bool {
			return 1, ErrNone
		}
		return 0, ErrNone
	case KindBlank:
		return 0, ErrNone
	case KindText:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v.Text), 64); err == nil {
			return f, ErrNone
		}
		return 0, ErrValue
	case KindError:
		return 0, v.Err
	}
	return 0, ErrValue
}

// toBool coerces a value to a boolean following the engine's coercion rules.
func toBool(v Value) (bool, ErrCode) {
	switch v.Kind {
	case KindBool:
		return v.Bool, ErrNone
	case KindNumber:
		return v.Num != 0, ErrNone
	case KindBlank:
		return false, ErrNone
	case KindText:
		switch strings.ToUpper(strings.TrimSpace(v.Text)) {
		case "TRUE":
			return true, ErrNone
		case "FALSE":
			return false, ErrNone
		}
		return false, ErrValue
	case KindError:
		return false, v.Err
	}
	return false, ErrValue
}
