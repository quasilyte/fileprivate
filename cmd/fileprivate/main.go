package main

import (
	"github.com/quasilyte/fileprivate"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(fileprivate.Analyzer)
}
