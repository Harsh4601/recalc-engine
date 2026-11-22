package recalc

// Node is a parsed formula expression. The parser in parser.go produces these;
// the evaluator in engine.go and functions.go consumes them.
type Node interface{ isNode() }

type NumberLit struct{ Val float64 }
type StringLit struct{ Val string }
type BoolLit struct{ Val bool }
type RefNode struct{ Ref CellRef }
type RangeNode struct{ Start, End CellRef }
type UnaryNode struct {
	Op byte // '+' or '-'
	X  Node
}
type BinaryNode struct {
	Op   string
	L, R Node
}
type CallNode struct {
	Name string
	Args []Node
}

// ErrorNode is produced only by structural edits, when a reference is deleted.
// It evaluates to its error code and serializes to the code's text.
type ErrorNode struct{ Code ErrCode }

func (NumberLit) isNode()  {}
func (StringLit) isNode()  {}
func (BoolLit) isNode()    {}
func (RefNode) isNode()    {}
func (RangeNode) isNode()  {}
func (UnaryNode) isNode()  {}
func (BinaryNode) isNode() {}
func (CallNode) isNode()   {}
func (ErrorNode) isNode()  {}
