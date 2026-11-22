package recalc

// This file implements structural edits: inserting and deleting rows and
// columns. An edit moves the cells on one side of the edit and rewrites the
// references inside every formula so they still denote the right cells.
//
// Reference-adjustment rules (per component, i.e. the row part and the column
// part of a reference are handled independently on the axis being edited):
//
//   - A RELATIVE component tracks the cell it points at: it shifts when that
//     cell moves, and the whole reference becomes #REF! when that cell is
//     deleted.
//   - An ABSOLUTE ($) component is pinned to its index: it never shifts and is
//     never deleted; after the edit it denotes whatever now sits at that index.
//
// A range endpoint follows the same rules. On deletion, if one relative
// endpoint falls inside the deleted band the range CONTRACTS to the surviving
// part; if the whole span is deleted the range becomes #REF!.

const (
	axisCol = 0
	axisRow = 1
)

type axisEdit struct {
	axis   int
	insert bool
	at     int
	count  int
}

// InsertRows inserts count blank rows before row index `before` (0-based).
func (e *Engine) InsertRows(before, count int) {
	e.applyEdit(axisEdit{axis: axisRow, insert: true, at: before, count: count})
}

// DeleteRows deletes count rows starting at row index `at` (0-based).
func (e *Engine) DeleteRows(at, count int) {
	e.applyEdit(axisEdit{axis: axisRow, insert: false, at: at, count: count})
}

// InsertCols inserts count blank columns before column index `before`.
func (e *Engine) InsertCols(before, count int) {
	e.applyEdit(axisEdit{axis: axisCol, insert: true, at: before, count: count})
}

// DeleteCols deletes count columns starting at column index `at`.
func (e *Engine) DeleteCols(at, count int) {
	e.applyEdit(axisEdit{axis: axisCol, insert: false, at: at, count: count})
}

func (e *Engine) applyEdit(ed axisEdit) {
	if ed.count <= 0 {
		return
	}
	if ed.at < 0 {
		ed.at = 0
	}
	newCells := make(map[[2]int]*cellData, len(e.cells))
	for k, cd := range e.cells {
		ni, dropped := mapContent(ed, k[ed.axis])
		if dropped {
			continue // cell sat on a deleted line
		}
		if cd.formula && !cd.parseErr {
			cd.ast = adjustNode(cd.ast, ed)
			cd.raw = "=" + serialize(cd.ast)
		}
		nk := k
		nk[ed.axis] = ni
		newCells[nk] = cd
	}
	e.cells = newCells
	e.dirty = true
}

// mapContent maps a cell's own coordinate on the edited axis. A cell behaves
// like a relative reference: it shifts, or is dropped when its line is deleted.
func mapContent(ed axisEdit, idx int) (int, bool) {
	if ed.insert {
		if idx >= ed.at {
			return idx + ed.count, false
		}
		return idx, false
	}
	switch {
	case idx < ed.at:
		return idx, false
	case idx < ed.at+ed.count:
		return 0, true // deleted
	default:
		return idx - ed.count, false
	}
}

// mapComponent maps one reference component (value + absolute flag) on the
// edited axis, reporting whether a relative component was deleted.
func mapComponent(ed axisEdit, idx int, abs bool) (int, bool) {
	if abs {
		return idx, false // pinned: never shifts, never deleted
	}
	return mapContent(ed, idx)
}

func component(r CellRef, axis int) (int, bool) {
	if axis == axisRow {
		return r.Row, r.RowAbs
	}
	return r.Col, r.ColAbs
}

func withComponent(r CellRef, axis, v int) CellRef {
	if axis == axisRow {
		r.Row = v
	} else {
		r.Col = v
	}
	return r
}

func adjustNode(n Node, ed axisEdit) Node {
	switch nd := n.(type) {
	case RefNode:
		idx, abs := component(nd.Ref, ed.axis)
		ni, deleted := mapComponent(ed, idx, abs)
		if deleted {
			return ErrorNode{Code: ErrRef}
		}
		return RefNode{Ref: withComponent(nd.Ref, ed.axis, ni)}
	case RangeNode:
		return adjustRange(nd, ed)
	case UnaryNode:
		return UnaryNode{Op: nd.Op, X: adjustNode(nd.X, ed)}
	case BinaryNode:
		return BinaryNode{Op: nd.Op, L: adjustNode(nd.L, ed), R: adjustNode(nd.R, ed)}
	case CallNode:
		args := make([]Node, len(nd.Args))
		for i, a := range nd.Args {
			args[i] = adjustNode(a, ed)
		}
		return CallNode{Name: nd.Name, Args: args}
	default:
		return n // literals and ErrorNode are unchanged
	}
}

func adjustRange(rn RangeNode, ed axisEdit) Node {
	sIdx, sAbs := component(rn.Start, ed.axis)
	eIdx, eAbs := component(rn.End, ed.axis)

	if ed.insert {
		sNew, _ := mapComponent(ed, sIdx, sAbs)
		eNew, _ := mapComponent(ed, eIdx, eAbs)
		return RangeNode{
			Start: withComponent(rn.Start, ed.axis, sNew),
			End:   withComponent(rn.End, ed.axis, eNew),
		}
	}

	sNew, sDel := mapComponent(ed, sIdx, sAbs)
	eNew, eDel := mapComponent(ed, eIdx, eAbs)
	switch {
	case sDel && eDel:
		return ErrorNode{Code: ErrRef} // whole span deleted
	case sDel:
		// top endpoint deleted: contract down to the first surviving line
		sNew = ed.at
	case eDel:
		// bottom endpoint deleted: contract up to the last surviving line
		eNew = ed.at - 1
	}
	return RangeNode{
		Start: withComponent(rn.Start, ed.axis, sNew),
		End:   withComponent(rn.End, ed.axis, eNew),
	}
}
