# fileprivate

A Go linter that enforces more strict members access rules inside packages.

# What exactly does it do?

It checks that **unexported** types **unexported** members are not accessed wildly inside a signle package.

There are two exceptions to this rule to keep things pragmatic:

* If the usage occurs in the same file then it's OK
* If type `T` member is accessed from a `T` method declaration from another file

This code will trigger a warning:

```go
// file1.go

type data struct {
    name string
}
```
```go
// file2.go

func f(d *date) string {
    return d.name // accessing data.name member outside of the suggested context
}
```

Let's fix it:

```diff
 type data struct {
-    name string
+    Name string 
 }
```

Fixed code example:

```go
// file1.go

type data struct {
    Name string
}
```
```go
// file2.go

func f(d *date) string {
    return d.Name
}
```

Note: since fileprivate reports unexported types, it's never a breaking change to rename a field or a method. All changes are package-local
and are invisible to the package users.

## Rationale

Go has no concepts of file-protected and/or type-private.

What benefits do we get:

* When refactoring comes, it's easier to move those types in another package: we already made necessary APIs exported and regulated the ways that object
  is used inside its original package.
* For big package that can't be split into parts that's the way to keep things sane and avoid unfortunate incapsulation violations.

Keep in mind that this tool is not suitable for every Go code base out there. I would also not recommend adding it to your CI pipeline unless
every member of your project agrees with it. 
