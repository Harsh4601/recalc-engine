package recalc

import (
	"fmt"
	"strings"
)

// cellData is the stored state of one cell.
type cellData struct {
	raw      string
	formula  bool
	ast      Node
	parseErr bool
	val      Value
}

// Engine is a spreadsheet grid with on-demand recalculation and structural
// editing.
type Engine struct {
	cells map[[2]int]*cellData
	dirty bool
}

// New returns an empty engine.
func New() *Engine {
	return &Engine{cells: map[[2]int]*cellData{}}
}

// Set assigns a raw input to a cell. A value beginning with '=' is a formula;
// anything else is a literal. An empty input clears the cell.
func (e *Engine) Set(cellA1, input string) error {
	ref, ok := parseA1(strings.ToUpper(strings.TrimSpace(cellA1)))
	if !ok {
		return fmt.Errorf("bad cell reference %q", cellA1)
	}
	e.dirty = true
	if input == "" {
		delete(e.cells, ref.key())
		return nil
	}
	cd := &cellData{raw: input}
	if strings.HasPrefix(input, "=") {
		cd.formula = true
		ast, err := parseFormula(input[1:])
		if err != nil {
			cd.parseErr = true
		} else {
			cd.ast = ast
		}
	}
	e.cells[ref.key()] = cd
	return nil
}

// Get returns a cell's computed value, recalculating first if needed.
func (e *Engine) Get(cellA1 string) Value {
	if e.dirty {
		e.recalc()
	}
	ref, ok := parseA1(strings.ToUpper(strings.TrimSpace(cellA1)))
	if !ok {
		return Error(ErrRef)
	}
	cd, ok := e.cells[ref.key()]
	if !ok {
		return Blank()
	}
	return cd.val
}

// Formula returns a cell's current formula text (with the leading '='), or ""
// if the cell is empty or holds a literal. The text reflects any reference
// rewriting done by structural edits.
func (e *Engine) Formula(cellA1 string) string {
	ref, ok := parseA1(strings.ToUpper(strings.TrimSpace(cellA1)))
	if !ok {
		return ""
	}
	cd, ok := e.cells[ref.key()]
	if !ok || !cd.formula {
		return ""
	}
	if cd.parseErr {
		return cd.raw
	}
	return "=" + serialize(cd.ast)
}
