package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/example/cpu-benchmark/internal/bench"
)

var runBench = bench.Run

const usage = `cpubench - portable CPU benchmark

Usage:
  cpubench [options]

Options:
  -content string
        Comma-separated benchmark IDs to run, or "all". Use -list to see IDs. (default "all")
  -time duration
        Duration per benchmark item, for example 500ms, 3s, 1m. (default 1s)
  -cores int
        Worker count for parallel benchmark items. Single-core items always use one worker. (default NumCPU)
  -json
        Print machine-readable JSON results.
  -list
        List benchmark IDs.
  -help
        Show detailed help with benchmark focus and application scenarios.
`

func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cpubench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	content := fs.String("content", "all", "benchmark content")
	duration := fs.Duration("time", time.Second, "duration per benchmark item")
	cores := fs.Int("cores", runtime.NumCPU(), "parallel worker count")
	jsonOutput := fs.Bool("json", false, "print JSON")
	list := fs.Bool("list", false, "list benchmark IDs")
	help := fs.Bool("help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *help {
		printHelp(stdout)
		return 0
	}
	if *list {
		printList(stdout)
		return 0
	}
	if *duration <= 0 {
		fmt.Fprintln(stderr, "-time must be greater than zero")
		return 2
	}
	if *cores <= 0 {
		fmt.Fprintln(stderr, "-cores must be greater than zero")
		return 2
	}

	ids, err := bench.ParseIDs(*content)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	results, err := runBench(context.Background(), bench.RunOptions{
		IDs:      ids,
		Duration: *duration,
		Workers:  *cores,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(results); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	printResults(stdout, results)
	return 0
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, usage)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Benchmark contents:")
	writeCatalogTable(w, true)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Notes:")
	fmt.Fprintln(w, "  SIMD uses a compiler-vectorizable portable loop instead of forcing architecture-specific intrinsics.")
	fmt.Fprintln(w, "  Efficiency is a score per CPU-time proxy; true performance-per-watt needs external power telemetry.")
}

func printList(w io.Writer) {
	for _, item := range bench.Catalog() {
		fmt.Fprintf(w, "%s\t%s\n", item.ID, item.Name)
	}
}

func writeCatalogTable(w io.Writer, detailed bool) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if detailed {
		fmt.Fprintln(tw, "ID\tDimension\tParallel\tFocus\tScenario")
		for _, item := range bench.Catalog() {
			fmt.Fprintf(tw, "%s\t%s\t%v\t%s\t%s\n", item.ID, item.Dimension, item.Parallel, item.Focus, item.Scenario)
		}
	} else {
		fmt.Fprintln(tw, "ID\tDimension\tParallel\tUnit")
		for _, item := range bench.Catalog() {
			fmt.Fprintf(tw, "%s\t%s\t%v\t%s\n", item.ID, item.Dimension, item.Parallel, item.Unit)
		}
	}
	_ = tw.Flush()
}

func printResults(w io.Writer, results []bench.Result) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tDimension\tWorkers\tScore\tUnit\tDuration")
	for _, result := range results {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%.2f\t%s\t%.2fs\n",
			result.ID,
			result.Dimension,
			result.Workers,
			result.Score,
			result.Unit,
			result.Duration,
		)
	}
	_ = tw.Flush()
}

func KnownIDs() []string {
	items := bench.Catalog()
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	sort.Strings(ids)
	return ids
}

func JoinKnownIDs() string {
	return strings.Join(KnownIDs(), ",")
}
