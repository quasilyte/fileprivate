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

	currentFunc *ast.FuncDecl

	ignoredObjects map[string]struct{}

	out []warning
}

type warning struct {
	Begin      token.Pos
	End        token.Pos
	Message    string
	ObjectName string
}

func (c *packageChecker) CheckPackage(files []*ast.File) []warning {
	for _, f := range files {
		c.checkFile(f)
	}

	out := c.out[:0]
	for _, w := range c.out {
		if _, ok := c.ignoredObjects[w.ObjectName]; ok {
			continue
		}
		out = append(out, w)
	}
	return out
}

func (c *packageChecker) checkFile(f *ast.File) {
	astutil.Apply(f, func(cursor *astutil.Cursor) bool {
		if fn, ok := cursor.Node().(*ast.FuncDecl); ok {
			if c.currentFunc != nil {
				panic("internal error: overwriting current func")
			}
			c.currentFunc = fn
		}
		c.checkNode(cursor.Node())
		return true
	}, func(cursor *astutil.Cursor) bool {
		if fn, ok := cursor.Node().(*ast.FuncDecl); ok {
			if c.currentFunc != fn {
				panic("internal error: mismatching function to pop")
			}
			c.currentFunc = nil
		}
		return true
	})
}

func (c *packageChecker) checkNode(n ast.Node) {
	switch n := n.(type) {
	case *ast.FuncDecl:
		c.checkFuncDecl(n)
	case *ast.GenDecl:
		c.checkGenDecl(n)
	case *ast.SelectorExpr:
		c.checkSelectorExpr(n)
	case *ast.CompositeLit:
		c.checkCompositeLit(n)
	}
}

func (c *packageChecker) checkGenDecl(decl *ast.GenDecl) {
	if c.currentFunc != nil {
		return // Skip non-global declarations
	}

	// For exported var/const/type we need to ignore types that
	// can leak to the outside through them.
	for _, spec := range decl.Specs {
		switch spec := spec.(type) {
		case *ast.ValueSpec:
			// If exported var/const has unexported type,
			// ignore the analysis for that unexported type.
			for _, name := range spec.Names {
				if !ast.IsExported(name.Name) {
					continue
				}
				c.ignoreObjectIfUnexported(c.getObject(name))
			}

		case *ast.TypeSpec:
			// If exported struct has any exported field that has
			// an unexported type, ignore the analysis for that unexported type.
			if !ast.IsExported(spec.Name.Name) {
				continue
			}
			object := c.getObject(spec.Name)
			if object == nil {
				continue
			}
			structType, ok := object.Type().Underlying().(*types.Struct)
			if !ok {
				continue
			}
			for i := 0; i < structType.NumFields(); i++ {
				field := structType.Field(i)
				if !ast.IsExported(field.Name()) {
					continue
				}
				c.ignoreObjectIfUnexported(c.getObjectFromType(field.Type()))
			}

		}
	}
}

func (c *packageChecker) checkFuncDecl(decl *ast.FuncDecl) {
	// If exported function returns unexported object of type T,
	// we record that type T as something that we need to ignore.
	if !ast.IsExported(decl.Name.Name) {
		return
	}
	fnType, ok := c.typeOf(decl.Name).(*types.Signature)
	if !ok {
		return
	}
	for i := 0; i < fnType.Results().Len(); i++ {
		field := fnType.Results().At(i)
		c.ignoreObjectIfUnexported(c.getObjectFromType(field.Type()))
	}
}

func (c *packageChecker) ignoreObjectIfUnexported(obj types.Object) {
	if obj == nil {
		return
	}
	if !ast.IsExported(obj.Name()) {
		c.ignoreObject(obj.Name())
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
		c.warnf(object, e.Sel, "accessing %s.%s outside of the suggested context", object.Name(), e.Sel)
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
				c.warnf(object, elt.Key, "accessing %s.%s outside of the suggested context", object.Name(), fieldName)
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
				c.warnf(object, elt, "accessing %s.%s outside of the suggested context (composite lit member %d)", object.Name(), field.Name(), i)
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

func (c *packageChecker) ignoreObject(name string) {
	if c.ignoredObjects == nil {
		c.ignoredObjects = make(map[string]struct{})
	}
	c.ignoredObjects[name] = struct{}{}
}

func (c *packageChecker) warnf(obj types.Object, n ast.Node, format string, args ...interface{}) {
	w := warning{
		Begin:      n.Pos(),
		End:        n.End(),
		Message:    fmt.Sprintf(format, args...),
		ObjectName: obj.Name(),
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
	fn := c.currentFunc
	if fn.Recv != nil && len(fn.Recv.List) == 1 && len(fn.Recv.List[0].Names) == 1 {
		return fn.Recv.List[0].Names[0]
	}
	return nil
}
