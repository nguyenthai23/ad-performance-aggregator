package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func main() {
	input := flag.String("input", "", "path to the input CSV file")
	output := flag.String("output", "results", "output directory for result CSV files")
	workers := flag.Int("workers", runtime.NumCPU(), "number of parallel workers")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "error: --input flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(*input); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: input file %q does not exist\n", *input)
		os.Exit(1)
	}

	fmt.Printf("Processing %s with %d workers...\n", *input, *workers)
	start := time.Now()

	stats, err := Aggregate(*input, *workers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: aggregation failed: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)
	fmt.Printf("Aggregated %d campaigns in %v\n", len(stats), elapsed)

	topCTR := TopByCTR(stats, 10)
	topCPA := TopByCPA(stats, 10)

	ctrPath := filepath.Join(*output, "top10_ctr.csv")
	cpaPath := filepath.Join(*output, "top10_cpa.csv")

	if err := WriteCSV(topCTR, ctrPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: write CTR results: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s\n", ctrPath)

	if err := WriteCSV(topCPA, cpaPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: write CPA results: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s\n", cpaPath)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Printf("\n--- Performance ---\n")
	fmt.Printf("Total time:       %v\n", elapsed)
	fmt.Printf("Peak memory:      %.2f MB\n", float64(mem.Sys)/1024/1024)
	fmt.Printf("Heap in use:      %.2f MB\n", float64(mem.HeapInuse)/1024/1024)
	fmt.Printf("Total allocated:  %.2f MB\n", float64(mem.TotalAlloc)/1024/1024)
}
