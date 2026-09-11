package cli

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/saurabhsharma2u/iambot/internal/logparse"
	"github.com/saurabhsharma2u/iambot/internal/matcher"
	"github.com/saurabhsharma2u/iambot/internal/registry"
	"github.com/saurabhsharma2u/iambot/internal/report"
)

// BenchmarkScanPipeline mirrors scan's hot loop (parse + lookup + report)
// and reports throughput as lines/s.
func BenchmarkScanPipeline(b *testing.B) {
	ctx := context.Background()
	reg := registry.NewDiskRegistry(b.TempDir())

	meta := matcher.Meta{Source: "bench", Category: "bench", Name: "bench"}
	var entries []registry.Entry
	for i := 0; i < 10000; i++ {
		entries = append(entries, registry.Entry{
			Prefix: fmt.Sprintf("10.%d.%d.0/24", (i/256)%256, i%256),
			Meta:   meta,
		})
	}
	entries = append(entries,
		registry.Entry{Prefix: "40.77.167.0/24", Meta: meta},
		registry.Entry{Prefix: "66.249.79.0/27", Meta: meta},
	)
	stats := registry.Stats{
		Sources:    map[string]int{"bench": len(entries)},
		Categories: map[string]int{"bench": len(entries)},
	}
	if err := reg.SaveRaw(ctx, entries, stats); err != nil {
		b.Fatal(err)
	}
	if err := reg.Load(ctx); err != nil {
		b.Fatal(err)
	}

	ips := []string{"40.77.167.61", "66.249.79.132", "8.8.8.8", "1.1.1.1", "205.234.147.102"}
	lines := make([]string, 2500)
	for i := range lines {
		lines[i] = ips[i%len(ips)] + ` - - [11/Sep/2026:00:08:19 +0000] "GET / HTTP/2.0" 200 100 "-" "bench"`
	}

	parser, err := logparse.GetParser("auto")
	if err != nil {
		b.Fatal(err)
	}
	reporter, err := report.GetReporter("text")
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry, err := parser.Parse(lines[i%len(lines)])
		if err != nil {
			b.Fatal(err)
		}
		hit, matched := reg.Contains(entry.IP)
		if !matched {
			continue
		}
		res := report.Result{Entry: entry, Hit: hit, Match: true}
		if err := reporter.Report(io.Discard, res); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "lines/s")
}
