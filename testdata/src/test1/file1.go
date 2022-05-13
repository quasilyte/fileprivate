package test1

type file1type struct {
	name     string
	Exported string
}

type file1iface interface {
	foo()
}

func (b *file1type) ExportedFunc() {}
func (b *file1type) fn()           {}

func (b *file1type) foo() {}
