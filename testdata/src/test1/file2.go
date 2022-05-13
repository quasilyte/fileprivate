package test1

func stringPtr(s *string) {}

type file2type struct {
	x    file1type
	xptr *file1type
}

func useIface(f file1iface) {
	f.foo()
}

func _() {
	var t file2type
	t.x = file1type{}
	t.xptr = &t.x
	t.x.name = "123"    // want `\Qaccessing file1type.name outside of the suggested context`
	t.xptr.name = "123" // want `\Qaccessing file1type.name outside of the suggested context`

	t.x = file1type{
		"a", // want `\Qaccessing file1type.name outside of the suggested context (composite lit member 0)`
		"b",
	}

	useIface(&t.x)
	useIface(t.xptr)
}

func _() {
	var b file1type
	b.name = "123" // want `\Qaccessing file1type.name outside of the suggested context`
	b.Exported = "024"
	b = file1type{name: "123"} // want `\Qaccessing file1type.name outside of the suggested context`
	b2 := &file1type{
		Exported: "4",
		name:     "123", // want `\Qaccessing file1type.name outside of the suggested context`
	}
	println(b2.name) // want `\Qaccessing file1type.name outside of the suggested context`
	switch b2.name { // want `\Qaccessing file1type.name outside of the suggested context`
	case "1":
	}
	stringPtr(&b2.name) // want `\Qaccessing file1type.name outside of the suggested context`

	b.fn() // want `\Qaccessing file1type.fn outside of the suggested context`
	useIface(&b)
}
