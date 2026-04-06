package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

const csvHeader = "campaign_id,total_impressions,total_clicks,total_spend,total_conversions,CTR,CPA"

func WriteCSV(campaigns []*CampaignStats, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	fmt.Fprintln(w, csvHeader)

	for _, c := range campaigns {
		cpaStr := ""
		if cpa, ok := c.CPA(); ok {
			cpaStr = fmt.Sprintf("%.2f", cpa)
		}
		fmt.Fprintf(w, "%s,%d,%d,%.2f,%d,%.4f,%s\n",
			c.CampaignID,
			c.Impressions,
			c.Clicks,
			c.Spend,
			c.Conversions,
			c.CTR(),
			cpaStr,
		)
	}

	return nil
}
