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
	var ignored1 ignoredType1
	ignored1.data1 = 10
	var ignored2 ignoredType2
	ignored2.data2 = 34
	var e ExportedType
	e.unexportedField.name = "23" // want `\Qaccessing file1type.name outside of the suggested context`
	e.ExportedField.private = "32"
	var ignored3 ignoredType3
	ignored3.private = "sd"
	var ignored4 ignoredType4
	ignored4.x = 2.5
}

func _() {
	var t file2type
	t.x = file1type{}
	t.xptr = &t.x
	t.x.name = "123"    // want `\Qaccessing file1type.name outside of the suggested context`
	t.xptr.name = "123" // want `\Qaccessing file1type.name outside of the suggested context`

	var (
		t2 = file1type{}
	)
	t2.name = "d" // want `\Qaccessing file1type.name outside of the suggested context`

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
