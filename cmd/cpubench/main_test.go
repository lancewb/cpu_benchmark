package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-list"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run exit = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "hash") {
		t.Fatalf("expected list output, got:\n%s", stdout.String())
	}
}

func TestMainFunction(t *testing.T) {
	originalArgs := os.Args
	originalExit := exit
	defer func() {
		os.Args = originalArgs
		exit = originalExit
	}()

	os.Args = []string{"cpubench", "-help"}
	exit = func(code int) {
		if code != 0 {
			t.Fatalf("main exit = %d", code)
		}
		panic("exit")
	}

	defer func() {
		if recovered := recover(); recovered != "exit" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()
	main()
}
