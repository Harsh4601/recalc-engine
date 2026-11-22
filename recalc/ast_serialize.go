package recalc

import "strings"

// serialize renders an AST back to formula text (without the leading '='),
// inserting the minimum parentheses needed to preserve structure. It is used by
// Engine.Formula and by structural edits to refresh a cell's stored text.
func serialize(n Node) string {
	var sb strings.Builder
	writeNode(&sb, n, 0, false)
	return sb.String()
}

// writeNode writes n. parentPrec is the precedence of the enclosing binary
// operator (0 at top level); rightChild reports whether n is the right operand,
// which together decide whether n must be parenthesized.
func writeNode(sb *strings.Builder, n Node, parentPrec int, rightChild bool) {
	switch nd := n.(type) {
	case NumberLit:
		sb.WriteString(formatNumber(nd.Val))
	case StringLit:
		sb.WriteByte('"')
		sb.WriteString(strings.ReplaceAll(nd.Val, `"`, `""`))
		sb.WriteByte('"')
	case BoolLit:
		if nd.Val {
			sb.WriteString("TRUE")
		} else {
			sb.WriteString("FALSE")
		}
	case RefNode:
		sb.WriteString(nd.Ref.A1())
	case RangeNode:
		sb.WriteString(nd.Start.A1())
		sb.WriteByte(':')
		sb.WriteString(nd.End.A1())
	case ErrorNode:
		sb.WriteString(errString(nd.Code))
	case UnaryNode:
		sb.WriteByte(nd.Op)
		writeNode(sb, nd.X, 6, false) // unary binds tighter than any binary
	case CallNode:
		sb.WriteString(nd.Name)
		sb.WriteByte('(')
		for i, a := range nd.Args {
			if i > 0 {
				sb.WriteByte(',')
			}
			writeNode(sb, a, 0, false)
		}
		sb.WriteByte(')')
	case BinaryNode:
		prec := binPrec(nd.Op)
		paren := prec < parentPrec
		if prec == parentPrec {
			// '^' is right-associative; the others are left-associative.
			if nd.Op == "^" {
				paren = !rightChild
			} else {
				paren = rightChild
			}
		}
		if paren {
			sb.WriteByte('(')
		}
		writeNode(sb, nd.L, prec, false)
		sb.WriteString(nd.Op)
		writeNode(sb, nd.R, prec, true)
		if paren {
			sb.WriteByte(')')
		}
	}
}
