package bench

import (
	"context"
	"errors"
	"math"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestCatalogIsValidAndSorted(t *testing.T) {
	items := Catalog()
	if len(items) < 10 {
		t.Fatalf("expected broad benchmark catalog, got %d items", len(items))
	}

	ids := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		if item.ID == "" || item.Name == "" || item.Dimension == "" || item.Focus == "" || item.Scenario == "" || item.Unit == "" {
			t.Fatalf("catalog item has missing metadata: %+v", item)
		}
		if item.Run == nil {
			t.Fatalf("catalog item %s has nil Run", item.ID)
		}
		if seen[item.ID] {
			t.Fatalf("duplicate ID %s", item.ID)
		}
		seen[item.ID] = true
		ids = append(ids, item.ID)
	}
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(ids, sorted) {
		t.Fatalf("catalog IDs are not sorted: %v", ids)
	}
}

func TestFind(t *testing.T) {
	item, ok := Find("hash")
	if !ok {
		t.Fatal("expected hash benchmark to exist")
	}
	if item.ID != "hash" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if _, ok := Find("missing"); ok {
		t.Fatal("missing benchmark should not exist")
	}
}

func TestParseIDs(t *testing.T) {
	all, err := ParseIDs("all")
	if err != nil {
		t.Fatalf("ParseIDs(all): %v", err)
	}
	if len(all) != len(Catalog()) {
		t.Fatalf("all returned %d IDs, want %d", len(all), len(Catalog()))
	}
	emptyMeansAll, err := ParseIDs("")
	if err != nil {
		t.Fatalf("ParseIDs(empty): %v", err)
	}
	if !reflect.DeepEqual(all, emptyMeansAll) {
		t.Fatalf("empty content should select all IDs")
	}

	ids, err := ParseIDs(" hash, int-crypto,hash ")
	if err != nil {
		t.Fatalf("ParseIDs selected: %v", err)
	}
	want := []string{"hash", "int-crypto"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("IDs = %v, want %v", ids, want)
	}

	if _, err := ParseIDs("nope"); err == nil {
		t.Fatal("expected unknown ID error")
	}
	if _, err := ParseIDs(",,,"); err == nil {
		t.Fatal("expected empty selection error")
	}
}

func TestNormalizeOptions(t *testing.T) {
	got := NormalizeOptions(RunOptions{})
	if got.Duration <= 0 {
		t.Fatalf("duration not defaulted: %v", got.Duration)
	}
	if got.Workers <= 0 {
		t.Fatalf("workers not defaulted: %d", got.Workers)
	}

	want := RunOptions{Duration: 10 * time.Millisecond, Workers: 3}
	got = NormalizeOptions(want)
	if got.Duration != want.Duration || got.Workers != want.Workers {
		t.Fatalf("NormalizeOptions changed explicit values: %+v", got)
	}
}

func TestMergeResult(t *testing.T) {
	got := mergeResult(
		WorkerResult{Operations: 1, Bytes: 2, Extra: map[string]float64{"a": 1}},
		WorkerResult{Operations: 3, Bytes: 4, Extra: map[string]float64{"a": 2, "b": 5}},
	)
	if got.Operations != 4 || got.Bytes != 6 {
		t.Fatalf("unexpected totals: %+v", got)
	}
	if got.Extra["a"] != 3 || got.Extra["b"] != 5 {
		t.Fatalf("unexpected extra: %+v", got.Extra)
	}
}

func TestBuildResultScores(t *testing.T) {
	cases := []struct {
		unit string
		raw  WorkerResult
		want float64
	}{
		{unit: "MiB/s", raw: WorkerResult{Bytes: 2 * 1024 * 1024}, want: 1},
		{unit: "GiB/s", raw: WorkerResult{Bytes: 2 * 1024 * 1024 * 1024}, want: 1},
		{unit: "Mops/s", raw: WorkerResult{Operations: 2_000_000}, want: 1},
		{unit: "Msteps/s", raw: WorkerResult{Operations: 2_000_000}, want: 1},
		{unit: "Mbranches/s", raw: WorkerResult{Operations: 2_000_000}, want: 1},
		{unit: "GFLOP/s", raw: WorkerResult{Operations: 2_000_000_000}, want: 1},
		{unit: "Mpixels/s", raw: WorkerResult{Operations: 2_000_000}, want: 1},
		{unit: "Mrows/s", raw: WorkerResult{Operations: 2_000_000}, want: 1},
		{unit: "transforms/s", raw: WorkerResult{Operations: 20}, want: 10},
		{unit: "files/s", raw: WorkerResult{Operations: 20}, want: 10},
		{unit: "score/cpu-second", raw: WorkerResult{Operations: 20, Extra: map[string]float64{"cpu_seconds": 4}}, want: 5},
		{unit: "score/cpu-second", raw: WorkerResult{Operations: 20, Extra: map[string]float64{}}, want: 10},
		{unit: "unknown", raw: WorkerResult{Operations: 20}, want: 10},
	}
	for _, tc := range cases {
		t.Run(tc.unit, func(t *testing.T) {
			got := scoreFor(tc.unit, tc.raw, 2)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("scoreFor(%q) = %v, want %v", tc.unit, got, tc.want)
			}
		})
	}
}

func TestBuildResultZeroElapsed(t *testing.T) {
	result := buildResult(Benchmark{ID: "x", Name: "X", Unit: "files/s"}, WorkerResult{Operations: 1}, 0, 1)
	if result.Duration <= 0 || result.Score <= 0 || result.OpsPerSecond <= 0 {
		t.Fatalf("expected zero elapsed to be guarded: %+v", result)
	}
}

func TestRunSelectedBenchmarks(t *testing.T) {
	results, err := Run(context.Background(), RunOptions{
		IDs:      []string{"hash", "render"},
		Duration: time.Millisecond,
		Workers:  2,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results", len(results))
	}
	if results[0].ID != "hash" || results[0].Workers != 1 {
		t.Fatalf("unexpected single-core result: %+v", results[0])
	}
	if results[1].ID != "render" || results[1].Workers != 2 {
		t.Fatalf("unexpected parallel result: %+v", results[1])
	}
	for _, result := range results {
		if result.Score <= 0 {
			t.Fatalf("%s score should be positive: %+v", result.ID, result)
		}
	}
}

func TestRunUnknownAndCanceled(t *testing.T) {
	if _, err := Run(context.Background(), RunOptions{IDs: []string{"missing"}, Duration: time.Millisecond}); err == nil {
		t.Fatal("expected unknown ID error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results, err := Run(ctx, RunOptions{IDs: []string{"hash"}, Duration: time.Millisecond})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(results) != 1 {
		t.Fatalf("canceled run should include partial result, got %d", len(results))
	}
}

func TestRunDefaultsToAll(t *testing.T) {
	results, err := Run(context.Background(), RunOptions{Duration: time.Nanosecond, Workers: 1})
	if err != nil {
		t.Fatalf("Run defaults: %v", err)
	}
	if len(results) != len(Catalog()) {
		t.Fatalf("got %d results, want %d", len(results), len(Catalog()))
	}
}
