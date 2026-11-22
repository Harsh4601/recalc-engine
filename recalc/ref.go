package recalc

import "strconv"

// CellRef is a parsed A1-style reference with absolute/relative flags.
type CellRef struct {
	Col, Row       int // zero-based
	ColAbs, RowAbs bool
}

func (c CellRef) key() [2]int { return [2]int{c.Col, c.Row} }

// colName converts a zero-based column index to letters (0 -> A, 26 -> AA).
func colName(col int) string {
	name := ""
	col++
	for col > 0 {
		col--
		name = string(rune('A'+col%26)) + name
		col /= 26
	}
	return name
}

// A1 renders the reference in A1 notation, preserving absolute markers.
func (c CellRef) A1() string {
	s := ""
	if c.ColAbs {
		s += "$"
	}
	s += colName(c.Col)
	if c.RowAbs {
		s += "$"
	}
	s += strconv.Itoa(c.Row + 1)
	return s
}

// parseA1 parses an uppercase A1-style reference such as A1, $B$2 or C$10.
func parseA1(s string) (CellRef, bool) {
	var ref CellRef
	i := 0
	if i < len(s) && s[i] == '$' {
		ref.ColAbs = true
		i++
	}
	colStart := i
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	if i == colStart {
		return ref, false
	}
	col := 0
	for _, ch := range s[colStart:i] {
		col = col*26 + int(ch-'A'+1)
	}
	ref.Col = col - 1
	if i < len(s) && s[i] == '$' {
		ref.RowAbs = true
		i++
	}
	rowStart := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == rowStart || i != len(s) {
		return ref, false
	}
	row, err := strconv.Atoi(s[rowStart:i])
	if err != nil || row < 1 {
		return ref, false
	}
	ref.Row = row - 1
	return ref, true
}
