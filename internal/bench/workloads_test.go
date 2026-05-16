package bench

import (
	"context"
	"testing"
	"time"
)

func TestWorkloadHelpers(t *testing.T) {
	if !deadlineFrom(WorkerConfig{}).After(time.Now()) {
		t.Fatal("default deadline should be in the future")
	}
	if splitmix64(1) == splitmix64(2) {
		t.Fatal("splitmix64 should vary with input")
	}
	if rotl32(1, 1) != 2 {
		t.Fatal("rotl32 returned unexpected value")
	}
	data := fillBytes(16, 1)
	if len(data) != 16 {
		t.Fatalf("fillBytes length = %d", len(data))
	}
	if workloadScale(WorkerConfig{BlockSize: 12}, 34) != 12 {
		t.Fatal("explicit block size not used")
	}
	if workloadScale(WorkerConfig{}, 34) != 34 {
		t.Fatal("fallback block size not used")
	}
	if cacheLabel(32*kib) != "l1_like" || cacheLabel(256*kib) != "l2_like" || cacheLabel(mib) != "l3_like" {
		t.Fatal("unexpected cache labels")
	}
}

func TestDefaultAndMinimumWorkloadSizes(t *testing.T) {
	ctx := context.Background()
	config := WorkerConfig{Duration: 200 * time.Microsecond}
	if got := runHash(ctx, config); got.Bytes == 0 {
		t.Fatal("hash default block size produced no bytes")
	}
	if got := runCompression(ctx, config); got.Bytes == 0 {
		t.Fatal("compression default block size produced no bytes")
	}
	if got := runMemoryStream(ctx, WorkerConfig{Duration: 200 * time.Microsecond, BlockSize: 1}); got.Bytes == 0 {
		t.Fatal("memory stream minimum block size produced no bytes")
	}
	if got := runPointerChase(ctx, WorkerConfig{Duration: 200 * time.Microsecond, BlockSize: 1}); got.Operations == 0 {
		t.Fatal("pointer chase minimum block size produced no operations")
	}
}

func TestFFTAndCompilerHelpers(t *testing.T) {
	realPart := []float64{1, 0, 0, 0}
	imagPart := []float64{0, 0, 0, 0}
	fftInPlace(realPart, imagPart)
	for i := range realPart {
		if realPart[i] != 1 || imagPart[i] != 0 {
			t.Fatalf("unexpected FFT output at %d: %v + %vi", i, realPart[i], imagPart[i])
		}
	}

	source := generateSource(128, 1)
	if len(source) != 128 {
		t.Fatalf("source length = %d", len(source))
	}
	if tokens := scanAndParse(source); tokens == 0 {
		t.Fatal("expected generated source to produce tokens")
	}
}

func TestEveryCatalogWorkloadProducesWork(t *testing.T) {
	ctx := context.Background()
	for _, item := range Catalog() {
		t.Run(item.ID, func(t *testing.T) {
			result := item.Run(ctx, WorkerConfig{
				Duration:  200 * time.Microsecond,
				Workers:   1,
				WorkerID:  0,
				BlockSize: 8 * kib,
			})
			if result.Operations == 0 && result.Bytes == 0 {
				t.Fatalf("%s produced no measurable work: %+v", item.ID, result)
			}
		})
	}
}

func TestRunWorkersCombinesResults(t *testing.T) {
	result := runWorkers(context.Background(), func(_ context.Context, config WorkerConfig) WorkerResult {
		return WorkerResult{
			Operations: uint64(config.WorkerID + 1),
			Bytes:      10,
			Extra:      map[string]float64{"workers": 1},
		}
	}, WorkerConfig{Duration: time.Millisecond}, 3)
	if result.Operations != 6 || result.Bytes != 30 || result.Extra["workers"] != 3 {
		t.Fatalf("unexpected combined result: %+v", result)
	}
}
