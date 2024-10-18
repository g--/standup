package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/cockroachdb/ttycolor"
)

func outputTitle(s string) {
	ttycolor.Stdout(ttycolor.Black)
	// bold
	os.Stdout.Write([]byte("\033[0;1;m"))
	fmt.Printf("%s\n", s)
	ttycolor.Stdout(ttycolor.Reset)
}

var indentTextRegexp *regexp.Regexp = regexp.MustCompile(`(\n)`)

func outputBody(s string) {
	indented := indentTextRegexp.ReplaceAllString(s, "\n    ")
	fmt.Printf("    %s\n\n", indented)
}
