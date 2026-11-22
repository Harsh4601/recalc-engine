# recalc

A small spreadsheet recalculation engine in Go. It parses A1-style formulas,
tracks dependencies between cells, recalculates on demand, and supports
structural edits (inserting and deleting rows and columns) that rewrite the
affected formulas.

- Values: numbers, text, booleans, blanks, and error values (`#DIV/0!`,
  `#VALUE!`, `#NAME?`, `#REF!`, `#CYCLE!`).
- Formulas: cell and range references (relative, absolute, mixed), the operators
  `+ - * / ^ & = <> < <= > >=`, and a library of functions (`IF`, `IFERROR`,
  `AND`, `OR`, `NOT`, `SUM`, `AVERAGE`, `MIN`, `MAX`, `COUNT`, `ABS`, `ROUND`,
  `LEN`, `LEFT`, `RIGHT`, `MID`, `CONCAT`, `ISERROR`, `ISNUMBER`).
- Recalculation: dependency-ordered evaluation with cycle detection.
- Structural edits: `InsertRows`, `DeleteRows`, `InsertCols`, `DeleteCols`, with
  reference rewriting and range contraction.

## Install

```
go get github.com/Harsh4601/recalc-engine/recalc
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/Harsh4601/recalc-engine/recalc"
)

func main() {
	e := recalc.New()
	e.Set("A1", "10")
	e.Set("A2", "20")
	e.Set("B1", "=SUM(A1:A2)")
	fmt.Println(e.Get("B1")) // 30

	// Insert a row above row 2; the formula updates automatically.
	e.InsertRows(1, 1)
	fmt.Println(e.Formula("B2")) // =SUM(A1:A3)
	fmt.Println(e.Get("B2"))     // 30
}
```

Row and column indices in the edit methods are 0-based: `InsertRows(1, 1)`
inserts one row before the second row.

## Semantics notes

- References adjust per component under structural edits: a relative part tracks
  the cell it points at (and becomes `#REF!` if that cell is deleted), while an
  absolute (`$`) part is pinned to its index.
- Deleting rows or columns that a range spans contracts the range; deleting all
  of it yields `#REF!`.
- When an aggregate sees several error values, the one with the highest priority
  wins (`#CYCLE!` > `#REF!` > `#DIV/0!` > `#VALUE!` > `#NAME?`).

## Tests

```
go test ./...
```

The behavior is pinned by a set of scenario files under
`recalc/testdata/scenarios`; each lists a sequence of cell edits and the expected
results.

## License

MIT. See [LICENSE](LICENSE).
