package test1

// It is OK to define extra methods for types in a separate file.
// The method body can access object data directly.
func (b *file1type) fn2() string {
	return b.name
}

func (b *file1type) fn3() string {
	var t file2type
	return t.x.name // want `\Qaccessing file2type.x outside of the suggested context`
}
