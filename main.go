package main

import (
	"github.com/alecthomas/kong"
)

var CLI struct {
	Branch struct {
	} `cmd:"" help:"Branch status"`
}

func main() {
	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	case "branch":
		branchStatus()
	default:
		panic(ctx.Command())
	}
}
