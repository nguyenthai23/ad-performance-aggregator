package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type chunk struct {
	start int64
	end   int64
}

func Aggregate(path string, workers int) (map[string]*CampaignStats, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	fileSize := fi.Size()

	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	chunks, err := splitChunks(path, fileSize, workers)
	if err != nil {
		return nil, fmt.Errorf("split chunks: %w", err)
	}

	type workerResult struct {
		stats map[string]*CampaignStats
		err   error
	}

	results := make([]workerResult, len(chunks))
	var wg sync.WaitGroup

	for i, c := range chunks {
		wg.Add(1)
		go func(idx int, c chunk) {
			defer wg.Done()
			stats, err := processChunk(path, c)
			results[idx] = workerResult{stats: stats, err: err}
		}(i, c)
	}

	wg.Wait()

	merged := make(map[string]*CampaignStats)
	for _, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("worker error: %w", r.err)
		}
		for id, s := range r.stats {
			if existing, ok := merged[id]; ok {
				existing.Merge(s)
			} else {
				merged[id] = s
			}
		}
	}

	return merged, nil
}

func splitChunks(path string, fileSize int64, n int) ([]chunk, error) {
	if fileSize == 0 {
		return nil, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Skip header line to find where data starts.
	br := bufio.NewReader(f)
	header, err := br.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read header: %w", err)
	}
	dataStart := int64(len(header))

	dataSize := fileSize - dataStart
	if dataSize <= 0 {
		return nil, nil
	}

	chunkSize := dataSize / int64(n)
	if chunkSize == 0 {
		return []chunk{{start: dataStart, end: fileSize}}, nil
	}

	chunks := make([]chunk, 0, n)
	offset := dataStart

	for i := 0; i < n && offset < fileSize; i++ {
		end := offset + chunkSize
		if i == n-1 || end >= fileSize {
			end = fileSize
		} else {
			end, err = alignToNewline(f, end)
			if err != nil {
				return nil, fmt.Errorf("align chunk %d: %w", i, err)
			}
		}
		if end > offset {
			chunks = append(chunks, chunk{start: offset, end: end})
		}
		offset = end
	}

	return chunks, nil
}

// alignToNewline seeks to pos and scans forward to the next newline,
// returning the byte offset immediately after that newline.
func alignToNewline(f *os.File, pos int64) (int64, error) {
	if _, err := f.Seek(pos, io.SeekStart); err != nil {
		return 0, err
	}
	buf := make([]byte, 4096)
	cur := pos
	for {
		n, err := f.Read(buf)
		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				return cur + int64(i) + 1, nil
			}
		}
		cur += int64(n)
		if err == io.EOF {
			return cur, nil
		}
		if err != nil {
			return 0, err
		}
	}
}

func processChunk(path string, c chunk) (map[string]*CampaignStats, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Seek(c.start, io.SeekStart); err != nil {
		return nil, err
	}

	reader := io.LimitReader(f, c.end-c.start)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 256), 1024)

	stats := make(map[string]*CampaignStats, 64)
	var lineNum int

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		s, err := parseLine(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: skipping malformed row at chunk offset %d, line %d: %v\n", c.start, lineNum, err)
			continue
		}

		if existing, ok := stats[s.CampaignID]; ok {
			existing.Merge(s)
		} else {
			stats[s.CampaignID] = s
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner: %w", err)
	}

	return stats, nil
}

func parseLine(line string) (*CampaignStats, error) {
	// Fast field extraction using IndexByte instead of strings.Split
	// Expected: campaign_id,date,impressions,clicks,spend,conversions
	var fields [6]string
	remaining := line

	for i := 0; i < 5; i++ {
		idx := strings.IndexByte(remaining, ',')
		if idx < 0 {
			return nil, fmt.Errorf("expected 6 fields, got %d", i+1)
		}
		fields[i] = remaining[:idx]
		remaining = remaining[idx+1:]
	}
	fields[5] = remaining

	impressions, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse impressions %q: %w", fields[2], err)
	}

	clicks, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse clicks %q: %w", fields[3], err)
	}

	spend, err := strconv.ParseFloat(fields[4], 64)
	if err != nil {
		return nil, fmt.Errorf("parse spend %q: %w", fields[4], err)
	}

	conversions, err := strconv.ParseInt(fields[5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse conversions %q: %w", fields[5], err)
	}

	return &CampaignStats{
		CampaignID:  fields[0],
		Impressions: impressions,
		Clicks:      clicks,
		Spend:       spend,
		Conversions: conversions,
	}, nil
}
