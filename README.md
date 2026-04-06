# Ad Performance Aggregator

A high-performance CLI tool that processes large CSV datasets of advertising performance records and produces aggregated analytics per campaign.

## Setup

**Prerequisites:** Go 1.22+ (or Docker)

```bash
git clone https://github.com/nguyenthai23/ad-performance-aggregator.git
cd ad-performance-aggregator
```

Place the `ad_data.csv` file in the project root.

### Build

```bash
go build -o aggregator .
```

Or use the Makefile:

```bash
make build
```

## How to Run

```bash
./aggregator --input ad_data.csv --output results
```

Or via Makefile (builds and runs):

```bash
make run
```

**Flags:**

| Flag        | Default          | Description                       |
|-------------|------------------|-----------------------------------|
| `--input`   | *(required)*     | Path to the input CSV file        |
| `--output`  | `results`        | Output directory for result files |
| `--workers` | Number of CPUs   | Parallel processing workers       |

**Output files:**
- `results/top10_ctr.csv` — Top 10 campaigns by highest CTR (Click-Through Rate)
- `results/top10_cpa.csv` — Top 10 campaigns by lowest CPA (Cost Per Acquisition)

### Docker

```bash
# Build image
docker build -t ad-aggregator .

# Run (mount the CSV and output directory)
docker run --rm \
  -v $(pwd)/ad_data.csv:/app/ad_data.csv \
  -v $(pwd)/results:/app/results \
  ad-aggregator --input ad_data.csv --output results/
```

Or via Makefile:

```bash
make docker-run
```

## Testing

```bash
# Run all tests with race detector
make test

# Run benchmarks
make bench
```

## Libraries Used

**None** — the solution uses only the Go standard library:

- `bufio` — buffered I/O for efficient file reading/writing
- `container/heap` — fixed-size heap for O(N log k) top-k selection
- `strconv` — fast numeric parsing without reflection
- `strings` — `IndexByte`-based field extraction (zero-slice allocation)
- `sync` — WaitGroup for goroutine coordination
- `flag` — CLI argument parsing
- `runtime` — CPU count detection and memory stats

No external dependencies were chosen intentionally to demonstrate proficiency with Go's stdlib and to minimize build complexity.

## Design Decisions

### Why streaming instead of loading the full file

The input CSV can exceed 1 GB. Loading it entirely into memory would require at least that much RAM plus allocation overhead for every parsed string. Instead, each worker streams its assigned byte range through a `bufio.Scanner` with a small fixed buffer (256 B default, 1 KB max). This keeps heap usage under 16 MB regardless of file size — critical for running in memory-constrained containers or alongside other services.

### Why a heap instead of sorting

Extracting the top-10 campaigns by CTR or CPA only requires tracking 10 elements at any time. A fixed-size min-heap (for highest CTR) or max-heap (for lowest CPA) processes all N campaigns in **O(N log k)** time and **O(k)** space, where k = 10. A full sort would cost O(N log N) and allocate a sorted slice of all campaigns — unnecessary when N could grow and k remains constant. The heap also avoids materializing an intermediate slice, reducing GC pressure.

### Why chunk-based concurrency

A single-goroutine reader would leave all but one core idle during the I/O-bound parse phase. Channel-based fan-out (one reader, many consumers) introduces a serialization point at the reader and per-row channel sends. Chunk-based parallelism sidesteps both problems: the file is divided into N byte-offset ranges aligned to newline boundaries, and each goroutine opens its own file descriptor, seeks to its chunk, and aggregates into a private map. This yields:

- **Zero lock contention** — no shared mutable state during processing.
- **Near-linear throughput scaling** — each worker drives independent I/O.
- **Trivial merge** — combining N small maps (≈50 keys each) is negligible.

### Trade-offs considered

| Decision | Benefit | Cost |
|---|---|---|
| Per-worker file descriptors | Eliminates shared reader bottleneck | N open FDs; minor OS overhead |
| `strings.IndexByte` parser | 1 alloc/row, 122 ns/op | No support for quoted fields or UTF-8 BOM |
| Skip malformed rows | Resilient to real-world data corruption | Silent data loss if format changes unexpectedly |
| Fixed-size heap for top-k | O(N log k) time, O(k) space | Slightly more complex code than `sort.Slice` |
| No external dependencies | Zero supply-chain risk, fast builds | Re-implements functionality available in libraries |

## Performance Results

**Environment:** Apple M1, 8 cores, macOS

**Input:** `ad_data.csv` — 995 MB, 26,843,544 data rows, 50 campaigns

| Metric              | Value         |
|---------------------|---------------|
| Processing time     | **4.54s**     |
| Peak memory (Sys)   | **15.77 MB**  |
| Heap in use         | 4.17 MB       |
| Workers             | 8             |

### Benchmarks

```
BenchmarkParseLine-8    9953758    121.9 ns/op    48 B/op    1 allocs/op
BenchmarkAggregate-8         85    15.5 ms/op     9.6 MB/op  (100K row fixture)
```

## Performance Considerations

### Time complexity

| Phase | Complexity | Notes |
|---|---|---|
| Chunk splitting | O(W) | W seeks + newline alignment scans |
| Parsing & aggregation | O(R / W) per worker | R = total rows, each parsed in O(1) with 6 fixed fields |
| Merge | O(W × C) | C = unique campaigns; combine W local maps |
| Top-k selection | O(C log k) | Fixed-size heap with k = 10 |
| **End-to-end** | **O(R / W + C log k)** | Parsing dominates; merge and top-k are negligible |

With R = 26.8 M rows and W = 8, effective throughput is **~219 MB/s** on an M1 — close to SSD sequential read speed.

### Memory usage

Memory consumption is independent of file size:

- **Scanner buffers:** W × 1 KB = 8 KB (256 B default, 1 KB max per worker).
- **Aggregation maps:** W × C × ~80 B per `CampaignStats` struct. For 8 workers × 50 campaigns = ~32 KB.
- **Heap:** k × 80 B = 800 B for top-10 selection.
- **Measured peak:** 15.77 MB `Sys` / 4.17 MB heap in use — the gap is Go runtime overhead (GC metadata, stack space, page-aligned arena reservation).

No row data is retained after parsing. The only long-lived allocations are the per-worker campaign maps and the final merged map.

### Why the solution scales for a 1 GB file

1. **Memory is O(W × C), not O(R).** Doubling the file size doubles processing time but adds zero heap growth. The 995 MB benchmark used only 4.17 MB of heap — a 240:1 ratio between input and memory.
2. **I/O parallelism saturates disk bandwidth.** Each worker issues independent sequential reads on its own file descriptor. On NVMe/SSD hardware this approaches the drive's throughput ceiling; adding workers beyond the I/O saturation point yields diminishing returns but never regresses.
3. **No intermediate data structures.** Rows are parsed, aggregated, and discarded in a single pass. There is no in-memory row buffer, no channel queue, and no serialized reader — eliminating the allocation and GC costs that typically limit Go programs on large inputs.
4. **Constant-factor output.** Regardless of input size, the final output is two 10-row CSV files. The heap-based top-k extraction ensures this phase never becomes a bottleneck even if the campaign count grows to thousands.

## Concurrency Model

The aggregator uses a chunk-based parallel pipeline to process large CSV files:

1. **Split** — The input file is divided into N byte-range chunks (one per worker),
   with boundaries aligned to newline characters so no row is split across workers.
2. **Map** — Each worker opens its own file handle, seeks to its chunk, and
   aggregates rows into a local `map[string]*CampaignStats`. Workers share no
   mutable state and require no synchronization during processing.
3. **Reduce** — After all workers complete, the main goroutine merges the per-worker
   maps into a single result by calling `CampaignStats.Merge` for overlapping keys.

This design eliminates lock contention on the critical read path at the cost of
temporarily duplicating the campaign map across workers. For typical workloads
(millions of rows, thousands of campaigns), the memory overhead is negligible and
throughput scales near-linearly with available cores.

Worker count defaults to `runtime.NumCPU()` and can be overridden with `--workers`.