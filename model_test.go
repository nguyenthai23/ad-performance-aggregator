package main

import (
	"reflect"
	"testing"
)

func campaignIDs(items []*CampaignStats) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = s.CampaignID
	}
	return out
}

func TestTopByCTRDeterministicWithTies(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Impressions: 1000, Clicks: 100}, // 0.10
		"B": {CampaignID: "B", Impressions: 10, Clicks: 1},     // 0.10 (tie with A)
		"C": {CampaignID: "C", Impressions: 1000, Clicks: 50},  // 0.05
		"D": {CampaignID: "D", Impressions: 100, Clicks: 30},   // 0.30
	}

	want := []string{"D", "A", "B"}
	for i := 0; i < 200; i++ {
		got := campaignIDs(TopByCTR(stats, 3))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: want %v, got %v", i, want, got)
		}
	}
}

func TestTopByCPADeterministicWithTiesAndZeroConversions(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Spend: 100, Conversions: 10}, // 10
		"B": {CampaignID: "B", Spend: 50, Conversions: 5},   // 10 (tie with A)
		"C": {CampaignID: "C", Spend: 12, Conversions: 3},   // 4
		"D": {CampaignID: "D", Spend: 70, Conversions: 0},   // excluded
		"E": {CampaignID: "E", Spend: 30, Conversions: 2},   // 15
	}

	want := []string{"C", "A", "B"}
	for i := 0; i < 200; i++ {
		got := campaignIDs(TopByCPA(stats, 3))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: want %v, got %v", i, want, got)
		}
	}
}

func TestTopKNonPositiveN(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Impressions: 100, Clicks: 10, Spend: 20, Conversions: 2},
	}

	if got := TopByCTR(stats, 0); got != nil {
		t.Fatalf("TopByCTR n=0: want nil, got %v", campaignIDs(got))
	}
	if got := TopByCTR(stats, -1); got != nil {
		t.Fatalf("TopByCTR n=-1: want nil, got %v", campaignIDs(got))
	}
	if got := TopByCPA(stats, 0); got != nil {
		t.Fatalf("TopByCPA n=0: want nil, got %v", campaignIDs(got))
	}
	if got := TopByCPA(stats, -1); got != nil {
		t.Fatalf("TopByCPA n=-1: want nil, got %v", campaignIDs(got))
	}
}

func TestTopByCPAReturnsAllEligibleWhenNIsLarge(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Spend: 100, Conversions: 20}, // 5
		"B": {CampaignID: "B", Spend: 50, Conversions: 5},   // 10
		"C": {CampaignID: "C", Spend: 70, Conversions: 0},   // excluded
	}

	got := campaignIDs(TopByCPA(stats, 10))
	want := []string{"A", "B"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestTopByCTRNormalCase(t *testing.T) {
	stats := make(map[string]*CampaignStats, 15)
	for i, ctr := range []struct {
		id          string
		impressions int64
		clicks      int64
	}{
		{"C01", 1000, 200}, // 0.200
		{"C02", 1000, 180}, // 0.180
		{"C03", 1000, 160}, // 0.160
		{"C04", 1000, 140}, // 0.140
		{"C05", 1000, 120}, // 0.120
		{"C06", 1000, 100}, // 0.100
		{"C07", 1000, 80},  // 0.080
		{"C08", 1000, 60},  // 0.060
		{"C09", 1000, 40},  // 0.040
		{"C10", 1000, 20},  // 0.020
		{"C11", 1000, 10},  // 0.010 — should be excluded
		{"C12", 1000, 5},   // 0.005
		{"C13", 1000, 3},   // 0.003
		{"C14", 1000, 2},   // 0.002
		{"C15", 1000, 1},   // 0.001
	} {
		_ = i
		stats[ctr.id] = &CampaignStats{
			CampaignID:  ctr.id,
			Impressions: ctr.impressions,
			Clicks:      ctr.clicks,
		}
	}

	top := TopByCTR(stats, 10)
	if len(top) != 10 {
		t.Fatalf("want 10 results, got %d", len(top))
	}

	want := []string{"C01", "C02", "C03", "C04", "C05", "C06", "C07", "C08", "C09", "C10"}
	got := campaignIDs(top)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	for i := 1; i < len(top); i++ {
		if top[i].CTR() > top[i-1].CTR() {
			t.Errorf("not sorted: %s (%.3f) > %s (%.3f)",
				top[i].CampaignID, top[i].CTR(), top[i-1].CampaignID, top[i-1].CTR())
		}
	}
}

func TestTopByCPANormalCase(t *testing.T) {
	stats := make(map[string]*CampaignStats, 15)
	for _, c := range []struct {
		id          string
		spend       float64
		conversions int64
	}{
		{"C01", 10, 10},  // CPA = 1
		{"C02", 20, 10},  // CPA = 2
		{"C03", 30, 10},  // CPA = 3
		{"C04", 40, 10},  // CPA = 4
		{"C05", 50, 10},  // CPA = 5
		{"C06", 60, 10},  // CPA = 6
		{"C07", 70, 10},  // CPA = 7
		{"C08", 80, 10},  // CPA = 8
		{"C09", 90, 10},  // CPA = 9
		{"C10", 100, 10}, // CPA = 10
		{"C11", 110, 10}, // CPA = 11 — should be excluded
		{"C12", 120, 10}, // CPA = 12
		{"C13", 200, 0},  // zero conversions — excluded
	} {
		stats[c.id] = &CampaignStats{
			CampaignID:  c.id,
			Spend:       c.spend,
			Conversions: c.conversions,
		}
	}

	top := TopByCPA(stats, 10)
	if len(top) != 10 {
		t.Fatalf("want 10 results, got %d", len(top))
	}

	want := []string{"C01", "C02", "C03", "C04", "C05", "C06", "C07", "C08", "C09", "C10"}
	got := campaignIDs(top)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	for i := 1; i < len(top); i++ {
		curCPA, _ := top[i].CPA()
		prevCPA, _ := top[i-1].CPA()
		if curCPA < prevCPA {
			t.Errorf("not sorted: %s (%.2f) < %s (%.2f)",
				top[i].CampaignID, curCPA, top[i-1].CampaignID, prevCPA)
		}
	}
}

func TestTopByCTRTiesBreakByCampaignID(t *testing.T) {
	stats := map[string]*CampaignStats{
		"Alpha":   {CampaignID: "Alpha", Impressions: 1000, Clicks: 100},   // 0.10
		"Bravo":   {CampaignID: "Bravo", Impressions: 500, Clicks: 50},     // 0.10 (tie)
		"Charlie": {CampaignID: "Charlie", Impressions: 200, Clicks: 20},   // 0.10 (tie)
		"Delta":   {CampaignID: "Delta", Impressions: 2000, Clicks: 200},   // 0.10 (tie)
		"Echo":    {CampaignID: "Echo", Impressions: 1000, Clicks: 300},    // 0.30 — highest
		"Foxtrot": {CampaignID: "Foxtrot", Impressions: 1000, Clicks: 250}, // 0.25
	}

	want := []string{"Echo", "Foxtrot", "Alpha", "Bravo", "Charlie", "Delta"}
	for i := 0; i < 200; i++ {
		got := campaignIDs(TopByCTR(stats, 6))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: want %v, got %v", i, want, got)
		}
	}
}

func TestTopByCPATiesBreakByCampaignID(t *testing.T) {
	stats := map[string]*CampaignStats{
		"Alpha":   {CampaignID: "Alpha", Spend: 100, Conversions: 10},   // CPA = 10
		"Bravo":   {CampaignID: "Bravo", Spend: 50, Conversions: 5},     // CPA = 10 (tie)
		"Charlie": {CampaignID: "Charlie", Spend: 200, Conversions: 20}, // CPA = 10 (tie)
		"Delta":   {CampaignID: "Delta", Spend: 30, Conversions: 10},    // CPA = 3 — best
		"Echo":    {CampaignID: "Echo", Spend: 70, Conversions: 10},     // CPA = 7
		"Foxtrot": {CampaignID: "Foxtrot", Spend: 80, Conversions: 0},  // excluded
	}

	want := []string{"Delta", "Echo", "Alpha", "Bravo", "Charlie"}
	for i := 0; i < 200; i++ {
		got := campaignIDs(TopByCPA(stats, 5))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: want %v, got %v", i, want, got)
		}
	}
}

func TestTopByCTRFewerThan10(t *testing.T) {
	stats := map[string]*CampaignStats{
		"X": {CampaignID: "X", Impressions: 1000, Clicks: 300}, // 0.30
		"Y": {CampaignID: "Y", Impressions: 1000, Clicks: 200}, // 0.20
		"Z": {CampaignID: "Z", Impressions: 1000, Clicks: 100}, // 0.10
	}

	top := TopByCTR(stats, 10)
	if len(top) != 3 {
		t.Fatalf("want 3 results (all available), got %d", len(top))
	}

	want := []string{"X", "Y", "Z"}
	got := campaignIDs(top)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestCTRZeroImpressions(t *testing.T) {
	c := &CampaignStats{CampaignID: "Z", Impressions: 0, Clicks: 0}
	if got := c.CTR(); got != 0 {
		t.Errorf("CTR with 0 impressions: want 0, got %f", got)
	}

	cWithClicks := &CampaignStats{CampaignID: "Z", Impressions: 0, Clicks: 50}
	if got := cWithClicks.CTR(); got != 0 {
		t.Errorf("CTR with 0 impressions and non-zero clicks: want 0, got %f", got)
	}
}

func TestCPAZeroConversions(t *testing.T) {
	c := &CampaignStats{CampaignID: "Z", Spend: 500, Conversions: 0}
	cpa, ok := c.CPA()
	if ok {
		t.Error("CPA with 0 conversions: want ok=false, got ok=true")
	}
	if cpa != 0 {
		t.Errorf("CPA with 0 conversions: want 0, got %f", cpa)
	}
}

func TestTopByCTRWithZeroImpressions(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Impressions: 0, Clicks: 0},      // CTR = 0
		"B": {CampaignID: "B", Impressions: 1000, Clicks: 100},  // CTR = 0.10
		"C": {CampaignID: "C", Impressions: 0, Clicks: 10},      // CTR = 0 (despite clicks)
		"D": {CampaignID: "D", Impressions: 500, Clicks: 50},    // CTR = 0.10
		"E": {CampaignID: "E", Impressions: 0, Clicks: 0},       // CTR = 0
	}

	top := TopByCTR(stats, 5)
	if len(top) != 5 {
		t.Fatalf("want 5 results, got %d", len(top))
	}

	if top[0].CampaignID != "B" && top[0].CampaignID != "D" {
		t.Errorf("top[0]: want B or D (CTR=0.10), got %s", top[0].CampaignID)
	}

	for i := 1; i < len(top); i++ {
		if top[i].CTR() > top[i-1].CTR() {
			t.Errorf("not descending: %s (%.4f) > %s (%.4f)",
				top[i].CampaignID, top[i].CTR(), top[i-1].CampaignID, top[i-1].CTR())
		}
	}
}

func TestTopByCTRAllZeroImpressions(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Impressions: 0, Clicks: 0},
		"B": {CampaignID: "B", Impressions: 0, Clicks: 0},
		"C": {CampaignID: "C", Impressions: 0, Clicks: 0},
	}

	top := TopByCTR(stats, 10)
	if len(top) != 3 {
		t.Fatalf("want 3, got %d", len(top))
	}
	for _, s := range top {
		if s.CTR() != 0 {
			t.Errorf("%s: want CTR=0, got %f", s.CampaignID, s.CTR())
		}
	}

	want := []string{"A", "B", "C"}
	got := campaignIDs(top)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tie-break by ID: want %v, got %v", want, got)
	}
}

func TestTopByCPAAllZeroConversions(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Spend: 100, Conversions: 0},
		"B": {CampaignID: "B", Spend: 200, Conversions: 0},
		"C": {CampaignID: "C", Spend: 300, Conversions: 0},
	}

	top := TopByCPA(stats, 10)
	if len(top) != 0 {
		t.Fatalf("want 0 results (all excluded), got %d: %v", len(top), campaignIDs(top))
	}
}

func TestTopByCPAMixedZeroConversions(t *testing.T) {
	stats := map[string]*CampaignStats{
		"A": {CampaignID: "A", Spend: 100, Conversions: 0},
		"B": {CampaignID: "B", Spend: 0, Conversions: 0},
		"C": {CampaignID: "C", Spend: 50, Conversions: 5},    // CPA = 10
		"D": {CampaignID: "D", Spend: 0, Conversions: 10},    // CPA = 0
		"E": {CampaignID: "E", Spend: 200, Conversions: 0},
		"F": {CampaignID: "F", Spend: 120, Conversions: 4},   // CPA = 30
	}

	top := TopByCPA(stats, 10)
	if len(top) != 3 {
		t.Fatalf("want 3 eligible results, got %d: %v", len(top), campaignIDs(top))
	}

	want := []string{"D", "C", "F"}
	got := campaignIDs(top)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestTopByCPAFewerThan10(t *testing.T) {
	stats := map[string]*CampaignStats{
		"P": {CampaignID: "P", Spend: 50, Conversions: 10},  // CPA = 5
		"Q": {CampaignID: "Q", Spend: 100, Conversions: 10}, // CPA = 10
		"R": {CampaignID: "R", Spend: 200, Conversions: 0},  // excluded
	}

	top := TopByCPA(stats, 10)
	if len(top) != 2 {
		t.Fatalf("want 2 results (eligible only), got %d", len(top))
	}

	want := []string{"P", "Q"}
	got := campaignIDs(top)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}
