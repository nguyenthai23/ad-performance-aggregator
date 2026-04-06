package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output", "test.csv")

	campaigns := []*CampaignStats{
		{CampaignID: "CMP001", Impressions: 100000, Clicks: 5000, Spend: 10000.50, Conversions: 500},
		{CampaignID: "CMP002", Impressions: 200000, Clicks: 8000, Spend: 16000.00, Conversions: 0},
	}

	if err := WriteCSV(campaigns, path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines (header + 2 rows), got %d", len(lines))
	}

	if lines[0] != csvHeader {
		t.Errorf("header: want %q, got %q", csvHeader, lines[0])
	}

	if !strings.Contains(lines[1], "CMP001") {
		t.Errorf("row 1 should contain CMP001: %s", lines[1])
	}
	if !strings.Contains(lines[1], "0.0500") {
		t.Errorf("row 1 CTR should be 0.0500: %s", lines[1])
	}
	if !strings.Contains(lines[1], "20.00") {
		t.Errorf("row 1 CPA should be 20.00: %s", lines[1])
	}

	if !strings.Contains(lines[2], "CMP002") {
		t.Errorf("row 2 should contain CMP002: %s", lines[2])
	}
	// CPA should be empty for zero conversions
	fields := strings.Split(lines[2], ",")
	if len(fields) != 7 {
		t.Fatalf("row 2: want 7 fields, got %d", len(fields))
	}
	if fields[6] != "" {
		t.Errorf("row 2 CPA should be empty for 0 conversions, got %q", fields[6])
	}
}

func TestWriteCSVCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "test.csv")

	campaigns := []*CampaignStats{
		{CampaignID: "CMP001", Impressions: 1000, Clicks: 50, Spend: 10.00, Conversions: 5},
	}

	if err := WriteCSV(campaigns, path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("output file was not created")
	}
}

func TestWriteCSVEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")

	if err := WriteCSV(nil, path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Errorf("want 1 line (header only), got %d", len(lines))
	}
}
