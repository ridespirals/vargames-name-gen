package main

import (
	"fmt"
	"html"
	"os"
	"time"

	"vargames-name-gen/src/igdb"
)

func writeFetchReport(metrics *igdb.Metrics, totalWallClock time.Duration, itemCount int, outPath string) error {
	entity, posts := metrics.Snapshot()
	requests, totalRetries, sumDuration := metrics.Totals()

	var minDur, maxDur time.Duration
	if len(posts) > 0 {
		minDur = posts[0].Duration
		maxDur = posts[0].Duration
	}
	for _, p := range posts {
		if p.Duration < minDur {
			minDur = p.Duration
		}
		if p.Duration > maxDur {
			maxDur = p.Duration
		}
	}
	avgDur := time.Duration(0)
	if requests > 0 {
		avgDur = sumDuration / time.Duration(requests)
	}

	// Retries distribution: count how many requests had 0, 1, 2, ... retries
	retriesBuckets := make(map[int]int)
	for _, p := range posts {
		retriesBuckets[p.Retries]++
	}
	maxRetriesInData := 0
	for r := range retriesBuckets {
		if r > maxRetriesInData {
			maxRetriesInData = r
		}
	}

	// Duration buckets for a simple bar chart (0-500ms, 500ms-1s, 1s-2s, 2s-5s, 5s+)
	durBuckets := []struct {
		label string
		max   time.Duration
		count int
	}{
		{"0–500ms", 500 * time.Millisecond, 0},
		{"500ms–1s", time.Second, 0},
		{"1–2s", 2 * time.Second, 0},
		{"2–5s", 5 * time.Second, 0},
		{"5s+", 1<<63 - 1, 0},
	}
	for _, p := range posts {
		for i := range durBuckets {
			if p.Duration <= durBuckets[i].max {
				durBuckets[i].count++
				break
			}
		}
	}
	maxCount := 0
	for _, b := range durBuckets {
		if b.count > maxCount {
			maxCount = b.count
		}
	}

	b := new(stringBuilder)
	b.w("<html><head><meta charset=\"utf-8\"><title>Fetch report: %s</title>",
		html.EscapeString(entity))
	b.w("<style>body{font-family:system-ui,sans-serif;padding:2rem;background:#1a1a1a;color:#e0e0e0;}")
	b.w("table{border-collapse:collapse;} th,td{border:1px solid #444;padding:0.5rem 1rem;text-align:left;} th{background:#333;}")
	b.w("h1{font-size:1.5rem;} h2{font-size:1.2rem;margin-top:2rem;} .meta{color:#888;margin-bottom:1.5rem;}")
	b.w(".bar-wrap{display:flex;align-items:center;gap:0.5rem;margin:0.25rem 0;} .bar{background:#4a9eff;height:1rem;border-radius:2px;} .bar-lbl{min-width:80px;} .bar-cnt{min-width:60px;}</style></head><body>")
	b.w("<h1>Fetch report: %s</h1>", html.EscapeString(entity))
	b.w("<p class=\"meta\">Generated at %s</p>", time.Now().Format(time.RFC3339))

	b.w("<h2>Summary</h2><table><tbody>")
	b.w("<tr><td>Fetchers (entity runs)</td><td>1</td></tr>")
	b.w("<tr><td>Entity</td><td>%s</td></tr>", html.EscapeString(entity))
	b.w("<tr><td>Total wall clock</td><td>%s</td></tr>", formatDur(totalWallClock))
	b.w("<tr><td>Total items fetched</td><td>%d</td></tr>", itemCount)
	b.w("<tr><td>Total requests (pages)</td><td>%d</td></tr>", requests)
	b.w("<tr><td>Total retries (all requests)</td><td>%d</td></tr>", totalRetries)
	b.w("<tr><td>Total attempts (HTTP calls)</td><td>%d</td></tr>", requests+totalRetries)
	b.w("<tr><td>Request duration (min / avg / max)</td><td>%s / %s / %s</td></tr>",
		formatDur(minDur), formatDur(avgDur), formatDur(maxDur))
	b.w("</tbody></table>")

	b.w("<h2>Per-fetcher</h2><table><thead><tr><th>Entity</th><th>Retries</th><th>Requests</th><th>Wall clock</th><th>Items</th></tr></thead><tbody>")
	b.w("<tr><td>%s</td><td>%d</td><td>%d</td><td>%s</td><td>%d</td></tr>",
		html.EscapeString(entity), totalRetries, requests, formatDur(totalWallClock), itemCount)
	b.w("</tbody></table>")

	b.w("<h2>Retries distribution</h2><p>Number of requests that required 0, 1, 2, … retries.</p>")
	for r := 0; r <= maxRetriesInData; r++ {
		c := retriesBuckets[r]
		pct := 0.0
		if requests > 0 {
			pct = 100 * float64(c) / float64(requests)
		}
		barW := 0
		if requests > 0 {
			barW = (c * 300) / requests
		}
		b.w("<div class=\"bar-wrap\"><span class=\"bar-lbl\">%d retries</span><span class=\"bar\" style=\"width:%dpx\"></span><span class=\"bar-cnt\">%d (%.1f%%)</span></div>",
			r, barW, c, pct)
	}

	b.w("<h2>Request duration distribution</h2>")
	for _, d := range durBuckets {
		lbl := d.label
		if d.max == 1<<63-1 {
			lbl = "5s+"
		}
		pct := 0.0
		if requests > 0 {
			pct = 100 * float64(d.count) / float64(requests)
		}
		barW := 0
		if maxCount > 0 {
			barW = (d.count * 300) / maxCount
		}
		b.w("<div class=\"bar-wrap\"><span class=\"bar-lbl\">%s</span><span class=\"bar\" style=\"width:%dpx\"></span><span class=\"bar-cnt\">%d (%.1f%%)</span></div>",
			lbl, barW, d.count, pct)
	}

	b.w("<h2>Sample of requests (first 20)</h2><table><thead><tr><th>#</th><th>Endpoint</th><th>Retries</th><th>Duration</th></tr></thead><tbody>")
	for i, p := range posts {
		if i >= 20 {
			break
		}
		b.w("<tr><td>%d</td><td>%s</td><td>%d</td><td>%s</td></tr>",
			i+1, html.EscapeString(p.Endpoint), p.Retries, formatDur(p.Duration))
	}
	b.w("</tbody></table></body></html>")

	return os.WriteFile(outPath, []byte(b.s), 0644)
}

func formatDur(d time.Duration) string {
	if d >= time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(time.Microsecond).String()
}

type stringBuilder struct {
	s string
}

func (b *stringBuilder) w(f string, a ...interface{}) {
	b.s += fmt.Sprintf(f, a...)
}
