package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatGateRejectsUnformattedSource(t *testing.T) {
	for _, row := range []struct {
		source string
		fail   bool
	}{
		{"package main\n\nfunc main() {}\n", false},
		{"package main;func main(){}", true},
	} {
		path := filepath.Join(t.TempDir(), "main.go")
		if err := os.WriteFile(path, []byte(row.source), 0600); err != nil {
			t.Fatal(err)
		}
		if err := checkFormat(path); (err != nil) != row.fail {
			t.Fatalf("format error = %v, want failure %v", err, row.fail)
		}
	}
}
