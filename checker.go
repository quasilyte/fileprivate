package fileprivate

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ast/astutil"
)

// Cases that are covered:
//
// * Calling unexported method
// * Assignment to unexported field
// * Read from unexported field
// * Setting unexported field via composite literal

type packageChecker struct {
	Types *types.Info
	Fset  *token.FileSet

	funcStack []*ast.FuncDecl

	out []warning
}

type warning struct {
	Begin   token.Pos
	End     token.Pos
	Message string
}

func (c *packageChecker) CheckPackage(files []*ast.File) []warning {
	for _, f := range files {
		c.checkFile(f)
	}

	return c.out
}

func (c *packageChecker) warnf(n ast.Node, format string, args ...interface{}) {
	w := warning{
		Begin:   n.Pos(),
		End:     n.End(),
		Message: fmt.Sprintf(format, args...),
	}
	c.out = append(c.out, w)
}

func (c *packageChecker) typeOf(e ast.Expr) types.Type {
	typ := c.Types.TypeOf(e)
	if typ == nil {
		return types.Typ[types.Invalid]
	}
	return typ
}

func (c *packageChecker) getObjectFromType(typ types.Type) types.Object {
	switch typ := typ.(type) {
	case *types.Named:
		return typ.Obj()
	case *types.Pointer:
		return c.getObjectFromType(typ.Elem())
	default:
		return nil
	}
}

func (c *packageChecker) getObject(e ast.Expr) types.Object {
	return c.getObjectFromType(c.typeOf(e))
}

func (c *packageChecker) getMethodReceiver() *ast.Ident {
	for i := len(c.funcStack) - 1; i >= 0; i-- {
		fn := c.funcStack[i]
		if fn.Recv != nil && len(fn.Recv.List) == 1 && len(fn.Recv.List[0].Names) == 1 {
			return fn.Recv.List[0].Names[0]
		}
	}
	return nil
}

func (c *packageChecker) checkFile(f *ast.File) {
	astutil.Apply(f, func(cursor *astutil.Cursor) bool {
		if fn, ok := cursor.Node().(*ast.FuncDecl); ok {
			c.funcStack = append(c.funcStack, fn)
			return true
		}
		c.checkNode(cursor.Node())
		return true
	}, func(cursor *astutil.Cursor) bool {
		if fn, ok := cursor.Node().(*ast.FuncDecl); ok {
			if c.funcStack[len(c.funcStack)-1] != fn {
				panic("internal error: mismatching function to pop")
			}
			c.funcStack = c.funcStack[:len(c.funcStack)-1]
		}
		return true
	})
}

func (c *packageChecker) checkNode(n ast.Node) {
	switch n := n.(type) {
	case *ast.SelectorExpr:
		c.checkSelectorExpr(n)
	case *ast.CompositeLit:
		c.checkCompositeLit(n)
	}
}

func (c *packageChecker) checkSelectorExpr(e *ast.SelectorExpr) {
	if ast.IsExported(e.Sel.Name) {
		return
	}

	object := c.getObject(e.X)
	if object == nil || ast.IsExported(object.Name()) {
		return
	}
	if _, isIface := object.Type().Underlying().(*types.Interface); isIface {
		return
	}
	if !c.canUse(object, e) {
		c.warnf(e.Sel, "accessing %s.%s outside of the suggested context", object.Name(), e.Sel)
	}
}

func (c *packageChecker) checkCompositeLit(e *ast.CompositeLit) {
	object := c.getObject(e.Type)
	if object == nil || ast.IsExported(object.Name()) {
		return
	}
	allowed := c.canUse(object, e)
	if allowed {
		return
	}
	structType, _ := object.Type().Underlying().(*types.Struct)
	for i, elt := range e.Elts {
		switch elt := elt.(type) {
		case *ast.KeyValueExpr:
			fieldName, ok := elt.Key.(*ast.Ident)
			if !ok || ast.IsExported(fieldName.Name) {
				continue
			}
			if !allowed {
				c.warnf(elt.Key, "accessing %s.%s outside of the suggested context", object.Name(), fieldName)
			}
		default:
			if structType == nil || i >= structType.NumFields() {
				continue
			}
			field := structType.Field(i)
			if ast.IsExported(field.Name()) {
				continue
			}
			if !allowed {
				c.warnf(elt, "accessing %s.%s outside of the suggested context (composite lit member %d)", object.Name(), field.Name(), i)
			}
		}
	}
}

func (c *packageChecker) canUse(o types.Object, usage ast.Expr) bool {
	declPos := c.Fset.Position(o.Pos())
	usagePos := c.Fset.Position(usage.Pos())
	if declPos.Filename == usagePos.Filename {
		return true
	}
	recv := c.getMethodReceiver()
	if recv != nil && c.getObject(recv) == o {
		return true
	}
	return false
}
