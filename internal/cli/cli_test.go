package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/example/cpu-benchmark/internal/bench"
)

func TestHelpAndList(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"-help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit = %d, stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Benchmark contents") || !strings.Contains(out, "Scenario") || !strings.Contains(out, "SIMD uses") {
		t.Fatalf("help output missing expected content:\n%s", out)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"-list"}, &stdout, &stderr); code != 0 {
		t.Fatalf("list exit = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "hash") {
		t.Fatalf("list output missing hash:\n%s", stdout.String())
	}
}

func TestRunJSONAndTable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"-content", "hash", "-time", "1ms", "-cores", "2", "-json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("json run exit = %d, stderr=%s", code, stderr.String())
	}
	var decoded []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if len(decoded) != 1 || decoded[0]["id"] != "hash" {
		t.Fatalf("unexpected JSON result: %#v", decoded)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"-content", "hash", "-time", "1ms"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("table run exit = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Score") || !strings.Contains(stdout.String(), "hash") {
		t.Fatalf("table output missing expected fields:\n%s", stdout.String())
	}
}

func TestInvalidArguments(t *testing.T) {
	cases := [][]string{
		{"-bad"},
		{"-time", "0s"},
		{"-cores", "0"},
		{"-content", "missing", "-time", "1ms"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(args, &stdout, &stderr); code == 0 {
				t.Fatalf("Run(%v) unexpectedly succeeded", args)
			}
		})
	}
}

func TestKnownIDs(t *testing.T) {
	ids := KnownIDs()
	if len(ids) == 0 {
		t.Fatal("KnownIDs returned no IDs")
	}
	joined := JoinKnownIDs()
	if !strings.Contains(joined, ids[0]) {
		t.Fatalf("JoinKnownIDs missing first ID: %s", joined)
	}
}

func TestCatalogSummaryTable(t *testing.T) {
	var stdout bytes.Buffer
	writeCatalogTable(&stdout, false)
	out := stdout.String()
	if !strings.Contains(out, "Unit") || !strings.Contains(out, "hash") {
		t.Fatalf("summary table missing expected content:\n%s", out)
	}
}

func TestRunBenchErrorAndJSONEncodeError(t *testing.T) {
	original := runBench
	defer func() { runBench = original }()

	runBench = func(context.Context, bench.RunOptions) ([]bench.Result, error) {
		return nil, errors.New("boom")
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"-content", "hash", "-time", "1ms"}, &stdout, &stderr); code != 1 {
		t.Fatalf("bench error exit = %d, stderr=%s", code, stderr.String())
	}

	runBench = func(context.Context, bench.RunOptions) ([]bench.Result, error) {
		return []bench.Result{{ID: "hash", Duration: float64(time.Millisecond)}}, nil
	}
	stderr.Reset()
	if code := Run([]string{"-content", "hash", "-time", "1ms", "-json"}, &errWriter{}, &stderr); code != 1 {
		t.Fatalf("json encode error exit = %d, stderr=%s", code, stderr.String())
	}
}

type errWriter struct{}

func (e *errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
