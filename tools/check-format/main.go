package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	if err := checkFormat("."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkFormat(path string) error {
	output, err := exec.Command("gofmt", "-l", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gofmt: %w: %s", err, output)
	}
	if len(output) != 0 {
		return fmt.Errorf("unformatted Go files:\n%s", output)
	}
	return nil
}
