package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func run(args []string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	// out, err := cmd.Output()
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%v failed with %w; stderr: \n%s", args, err, errb.String())
	}

	return outb.String(), nil
}
