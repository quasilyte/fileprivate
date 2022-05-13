package fileprivate

import (
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "fileprivate",
	Doc:  docString,
	Run:  run,
}

const docString = `fileprivate enforces stricter code structure rules on your code.
It makes it illegal to access unexported type unexported fields or methods,
unless they're defined in the same file.`

func run(pass *analysis.Pass) (interface{}, error) {
	c := &packageChecker{
		Types: pass.TypesInfo,
		Fset:  pass.Fset,
	}

	for _, w := range c.CheckPackage(pass.Files) {
		pass.Report(analysis.Diagnostic{
			Pos:     w.Begin,
			End:     w.End,
			Message: w.Message,
		})
	}

	return nil, nil
}
