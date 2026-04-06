package main

import "container/heap"

type CampaignStats struct {
	CampaignID  string
	Impressions int64
	Clicks      int64
	Spend       float64
	Conversions int64
}

func (c *CampaignStats) CTR() float64 {
	if c.Impressions == 0 {
		return 0
	}
	return float64(c.Clicks) / float64(c.Impressions)
}

func (c *CampaignStats) CPA() (float64, bool) {
	if c.Conversions == 0 {
		return 0, false
	}
	return c.Spend / float64(c.Conversions), true
}

func (c *CampaignStats) Merge(other *CampaignStats) {
	c.Impressions += other.Impressions
	c.Clicks += other.Clicks
	c.Spend += other.Spend
	c.Conversions += other.Conversions
}

// ctrHeap is a min-heap by CTR. The root is the smallest CTR in the heap,
// so it gets evicted first when a higher-CTR candidate arrives.
// Tie-break: higher CampaignID is "less" (evicted before lower ID).
type ctrHeap []*CampaignStats

func (h ctrHeap) Len() int      { return len(h) }
func (h ctrHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h ctrHeap) Less(i, j int) bool {
	ci, cj := h[i].CTR(), h[j].CTR()
	if ci != cj {
		return ci < cj
	}
	return h[i].CampaignID > h[j].CampaignID
}

func (h *ctrHeap) Push(x any) { *h = append(*h, x.(*CampaignStats)) }
func (h *ctrHeap) Pop() any {
	old := *h
	n := len(old)
	out := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return out
}

// cpaHeap is a max-heap by CPA. The root is the largest CPA in the heap,
// so it gets evicted first when a lower-CPA candidate arrives.
// Tie-break: higher CampaignID is "less" (evicted before lower ID).
type cpaHeap []*CampaignStats

func (h cpaHeap) Len() int      { return len(h) }
func (h cpaHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h cpaHeap) Less(i, j int) bool {
	ci, _ := h[i].CPA()
	cj, _ := h[j].CPA()
	if ci != cj {
		return ci > cj
	}
	return h[i].CampaignID > h[j].CampaignID
}

func (h *cpaHeap) Push(x any) { *h = append(*h, x.(*CampaignStats)) }
func (h *cpaHeap) Pop() any {
	old := *h
	n := len(old)
	out := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return out
}

// TopByCTR returns the top n campaigns by highest CTR using a fixed-size
// min-heap. Ties are broken by CampaignID ascending.
func TopByCTR(stats map[string]*CampaignStats, n int) []*CampaignStats {
	if n <= 0 || len(stats) == 0 {
		return nil
	}

	h := make(ctrHeap, 0, n)
	for _, s := range stats {
		if h.Len() < n {
			heap.Push(&h, s)
			continue
		}
		sc, rc := s.CTR(), h[0].CTR()
		if sc > rc || (sc == rc && s.CampaignID < h[0].CampaignID) {
			h[0] = s
			heap.Fix(&h, 0)
		}
	}
	result := make([]*CampaignStats, h.Len())
	for i := len(result) - 1; i >= 0; i-- {
		result[i] = heap.Pop(&h).(*CampaignStats)
	}
	return result
}

// TopByCPA returns the top n campaigns by lowest CPA using a fixed-size
// max-heap. Campaigns with zero conversions are excluded.
// Ties are broken by CampaignID ascending.
func TopByCPA(stats map[string]*CampaignStats, n int) []*CampaignStats {
	if n <= 0 || len(stats) == 0 {
		return nil
	}

	h := make(cpaHeap, 0, n)
	for _, s := range stats {
		if s.Conversions == 0 {
			continue
		}
		if h.Len() < n {
			heap.Push(&h, s)
			continue
		}
		sc, _ := s.CPA()
		rc, _ := h[0].CPA()
		if sc < rc || (sc == rc && s.CampaignID < h[0].CampaignID) {
			h[0] = s
			heap.Fix(&h, 0)
		}
	}
	result := make([]*CampaignStats, h.Len())
	for i := len(result) - 1; i >= 0; i-- {
		result[i] = heap.Pop(&h).(*CampaignStats)
	}
	return result
}
