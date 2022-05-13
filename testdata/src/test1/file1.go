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

type ignoredType1 struct {
	data1 int
}

type ignoredType2 struct {
	data2 int
}

type ignoredType3 struct {
	private string
}

type ignoredType4 struct {
	x float64
}

type ExportedType struct {
	ExportedField   ignoredType3
	unexportedField file1type
}

var ExportedVar = ignoredType4{}

func ExportedFunc1() *ignoredType1 {
	return &ignoredType1{}
}

func ExportedFunc2() ignoredType2 {
	return ignoredType2{}
}
