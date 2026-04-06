package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func writeTestCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAggregateBasic(t *testing.T) {
	csv := `campaign_id,date,impressions,clicks,spend,conversions
CMP001,2025-01-01,10000,500,100.00,10
CMP001,2025-01-02,20000,1000,200.00,20
CMP002,2025-01-01,5000,100,50.00,5
`
	path := writeTestCSV(t, csv)

	stats, err := Aggregate(path, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("expected 2 campaigns, got %d", len(stats))
	}

	c1 := stats["CMP001"]
	if c1 == nil {
		t.Fatal("CMP001 not found")
	}
	if c1.Impressions != 30000 {
		t.Errorf("CMP001 impressions: want 30000, got %d", c1.Impressions)
	}
	if c1.Clicks != 1500 {
		t.Errorf("CMP001 clicks: want 1500, got %d", c1.Clicks)
	}
	if c1.Spend != 300.00 {
		t.Errorf("CMP001 spend: want 300.00, got %f", c1.Spend)
	}
	if c1.Conversions != 30 {
		t.Errorf("CMP001 conversions: want 30, got %d", c1.Conversions)
	}

	c2 := stats["CMP002"]
	if c2 == nil {
		t.Fatal("CMP002 not found")
	}
	if c2.Impressions != 5000 {
		t.Errorf("CMP002 impressions: want 5000, got %d", c2.Impressions)
	}
}

func TestAggregateZeroConversions(t *testing.T) {
	csv := `campaign_id,date,impressions,clicks,spend,conversions
CMP001,2025-01-01,10000,500,100.00,0
CMP002,2025-01-01,5000,100,50.00,5
`
	path := writeTestCSV(t, csv)

	stats, err := Aggregate(path, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c1 := stats["CMP001"]
	if _, ok := c1.CPA(); ok {
		t.Error("CMP001 CPA: want ok=false for zero conversions")
	}

	c2 := stats["CMP002"]
	cpa, ok := c2.CPA()
	if !ok {
		t.Fatal("CMP002 CPA: want ok=true")
	}
	if cpa != 10.0 {
		t.Errorf("CMP002 CPA: want 10.0, got %f", cpa)
	}
}

func TestAggregateMalformedRows(t *testing.T) {
	csv := `campaign_id,date,impressions,clicks,spend,conversions
CMP001,2025-01-01,10000,500,100.00,10
BAD_ROW_MISSING_FIELDS
CMP001,2025-01-02,20000,1000,200.00,20
CMP002,2025-01-01,bad,100,50.00,5
`
	path := writeTestCSV(t, csv)

	stats, err := Aggregate(path, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c1 := stats["CMP001"]
	if c1 == nil {
		t.Fatal("CMP001 not found")
	}
	if c1.Impressions != 30000 {
		t.Errorf("CMP001 impressions: want 30000, got %d", c1.Impressions)
	}

	if _, ok := stats["CMP002"]; ok {
		t.Error("CMP002 should not exist (malformed row)")
	}
}

func TestAggregateEmptyFile(t *testing.T) {
	csv := `campaign_id,date,impressions,clicks,spend,conversions
`
	path := writeTestCSV(t, csv)

	stats, err := Aggregate(path, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected 0 campaigns, got %d", len(stats))
	}
}

func TestAggregateFileNotFound(t *testing.T) {
	_, err := Aggregate("/nonexistent/file.csv", 1)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestAggregateSingleWorker(t *testing.T) {
	csv := `campaign_id,date,impressions,clicks,spend,conversions
CMP001,2025-01-01,10000,500,100.00,10
CMP001,2025-01-02,20000,1000,200.00,20
`
	path := writeTestCSV(t, csv)

	stats, err := Aggregate(path, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c1 := stats["CMP001"]
	if c1.Impressions != 30000 || c1.Clicks != 1500 {
		t.Errorf("CMP001: want 30000/1500, got %d/%d", c1.Impressions, c1.Clicks)
	}
}

func TestAggregateMultipleWorkers(t *testing.T) {
	csv := `campaign_id,date,impressions,clicks,spend,conversions
CMP001,2025-01-01,1000,50,10.00,1
CMP002,2025-01-01,2000,100,20.00,2
CMP003,2025-01-01,3000,150,30.00,3
CMP001,2025-01-02,1000,50,10.00,1
CMP002,2025-01-02,2000,100,20.00,2
CMP003,2025-01-02,3000,150,30.00,3
`
	path := writeTestCSV(t, csv)

	for _, w := range []int{1, 2, 4, 8} {
		t.Run(fmt.Sprintf("workers=%d", w), func(t *testing.T) {
			stats, err := Aggregate(path, w)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(stats) != 3 {
				t.Fatalf("want 3 campaigns, got %d", len(stats))
			}
			c1 := stats["CMP001"]
			if c1.Impressions != 2000 || c1.Clicks != 100 {
				t.Errorf("CMP001: want 2000/100, got %d/%d", c1.Impressions, c1.Clicks)
			}
		})
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr bool
		wantID  string
	}{
		{
			name:   "valid",
			line:   "CMP001,2025-01-01,10000,500,100.50,10",
			wantID: "CMP001",
		},
		{
			name:    "too few fields",
			line:    "CMP001,2025-01-01",
			wantErr: true,
		},
		{
			name:    "bad impressions",
			line:    "CMP001,2025-01-01,abc,500,100.00,10",
			wantErr: true,
		},
		{
			name:    "bad clicks",
			line:    "CMP001,2025-01-01,10000,abc,100.00,10",
			wantErr: true,
		},
		{
			name:    "bad spend",
			line:    "CMP001,2025-01-01,10000,500,abc,10",
			wantErr: true,
		},
		{
			name:    "bad conversions",
			line:    "CMP001,2025-01-01,10000,500,100.00,abc",
			wantErr: true,
		},
		{
			name:    "empty string",
			line:    "",
			wantErr: true,
		},
		{
			name:    "single field",
			line:    "CMP001",
			wantErr: true,
		},
		{
			name:    "trailing comma (missing conversions value)",
			line:    "CMP001,2025-01-01,10000,500,100.00,",
			wantErr: true,
		},
		{
			name:    "extra commas in last field",
			line:    "CMP001,2025-01-01,10000,500,100.00,10,extra,fields",
			wantErr: true,
		},
		{
			name:    "negative impressions",
			line:    "CMP001,2025-01-01,-5,500,100.00,10",
			wantID:  "CMP001",
		},
		{
			name:    "negative spend",
			line:    "CMP001,2025-01-01,10000,500,-50.00,10",
			wantID:  "CMP001",
		},
		{
			name:    "zero impressions and zero conversions",
			line:    "CMP001,2025-01-01,0,0,0.00,0",
			wantID:  "CMP001",
		},
		{
			name:    "float in integer field",
			line:    "CMP001,2025-01-01,100.5,500,100.00,10",
			wantErr: true,
		},
		{
			name:    "whitespace in numeric field",
			line:    "CMP001,2025-01-01, 10000,500,100.00,10",
			wantErr: true,
		},
		{
			name:    "only commas",
			line:    ",,,,,",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := parseLine(tt.line)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.CampaignID != tt.wantID {
				t.Errorf("campaign_id: want %s, got %s", tt.wantID, s.CampaignID)
			}
		})
	}
}

func TestTopByCTR(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Impressions: 1000, Clicks: 100},
		"B": {CampaignID: "B", Impressions: 1000, Clicks: 200},
		"C": {CampaignID: "C", Impressions: 1000, Clicks: 50},
	}

	top := TopByCTR(stats, 2)
	if len(top) != 2 {
		t.Fatalf("want 2, got %d", len(top))
	}
	if top[0].CampaignID != "B" {
		t.Errorf("top[0]: want B, got %s", top[0].CampaignID)
	}
	if top[1].CampaignID != "A" {
		t.Errorf("top[1]: want A, got %s", top[1].CampaignID)
	}
}

func TestTopByCPA(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Spend: 100, Conversions: 10},
		"B": {CampaignID: "B", Spend: 100, Conversions: 5},
		"C": {CampaignID: "C", Spend: 100, Conversions: 0},
		"D": {CampaignID: "D", Spend: 100, Conversions: 20},
	}

	top := TopByCPA(stats, 3)
	if len(top) != 3 {
		t.Fatalf("want 3, got %d", len(top))
	}
	if top[0].CampaignID != "D" {
		t.Errorf("top[0]: want D (CPA=5), got %s", top[0].CampaignID)
	}
	if top[1].CampaignID != "A" {
		t.Errorf("top[1]: want A (CPA=10), got %s", top[1].CampaignID)
	}
	if top[2].CampaignID != "B" {
		t.Errorf("top[2]: want B (CPA=20), got %s", top[2].CampaignID)
	}
}

func BenchmarkParseLine(b *testing.B) {
	line := "CMP025,2025-04-18,3653,60,64.29,2"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseLine(line)
	}
}

func BenchmarkAggregate(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.csv")

	f, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	fmt.Fprintln(f, "campaign_id,date,impressions,clicks,spend,conversions")
	for i := 0; i < 100000; i++ {
		fmt.Fprintf(f, "CMP%03d,2025-01-%02d,%d,%d,%.2f,%d\n",
			i%50+1, i%28+1, 10000+i, 500+i%100, float64(100+i%50), 10+i%20)
	}
	f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Aggregate(path, 4)
		if err != nil {
			b.Fatal(err)
		}
	}
}
