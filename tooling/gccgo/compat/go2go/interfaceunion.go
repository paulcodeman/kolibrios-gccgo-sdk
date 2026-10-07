package go2go

import (
	"go/ast"
	"go/token"
	"go/types"
)

// A union containing an unconstrained interface, such as any | []byte, can
// denote an ordinary method-set interface. go/types has already checked the
// union and proven that its type set is universal. GCC's older parser cannot
// read the union syntax. Such an embedded union imposes no restriction and
// contributes no methods, so omit it while preserving the named interface and
// all explicit methods and ordinary embedded interfaces.
func (t *translator) lowerUniversalInterfaceUnions(node *ast.InterfaceType) {
	typ := t.importer.info.TypeOf(node)
	if typ == nil {
		return
	}
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok || !iface.IsMethodSet() {
		return
	}
	fields := node.Methods.List[:0]
	for _, field := range node.Methods.List {
		expr := field.Type
		for {
			paren, ok := expr.(*ast.ParenExpr)
			if !ok {
				break
			}
			expr = paren.X
		}
		if union, ok := expr.(*ast.BinaryExpr); ok && union.Op == token.OR && len(field.Names) == 0 {
			continue
		}
		fields = append(fields, field)
	}
	node.Methods.List = fields
}
