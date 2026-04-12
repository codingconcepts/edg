package main

import (
	"fmt"
	"os"
	"slices"
	"text/tabwriter"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
)

type queryStats struct {
	count         int64
	errors        int64
	totalLatency  time.Duration
	latencies     []time.Duration
	isTransaction bool
}

func printResults(results <-chan config.QueryResult, interval time.Duration, start time.Time, numWorkers int, totalDuration time.Duration) map[string]*queryStats {
	stats := map[string]*queryStats{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case r, ok := <-results:
			if !ok {
				printSummary(stats, start, numWorkers)
				return stats
			}
			s := stats[r.Name]
			if s == nil {
				s = &queryStats{}
				stats[r.Name] = s
			}
			if r.IsTransaction {
				s.isTransaction = true
			}
			if r.Err != nil {
				s.errors++
			} else {
				s.count += int64(r.Count)
				s.totalLatency += r.Latency
				s.latencies = append(s.latencies, r.Latency)
			}
		case <-ticker.C:
			printProgress(stats, start, totalDuration)
		}
	}
}

func printProgress(stats map[string]*queryStats, start time.Time, totalDuration time.Duration) {
	elapsed := time.Since(start)

	var queryNames, txNames []string
	for name, s := range stats {
		if s.isTransaction {
			txNames = append(txNames, name)
		} else {
			queryNames = append(queryNames, name)
		}
	}
	slices.Sort(queryNames)
	slices.Sort(txNames)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "\n%s / %s\n", elapsed.Round(time.Second), totalDuration.Round(time.Second))
	fmt.Fprintf(w, "QUERY\tCOUNT\tERRORS\tAVG\tp50\tp95\tp99\tQPS\n")
	for _, name := range queryNames {
		s := stats[name]
		var avg time.Duration
		if s.count > 0 {
			avg = s.totalLatency / time.Duration(s.count)
		}
		p50, p95, p99 := percentiles(s.latencies)
		qps := float64(s.count) / elapsed.Seconds()
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\t%s\t%s\t%.1f\n", name, s.count, s.errors,
			avg.Round(time.Microsecond),
			p50.Round(time.Microsecond),
			p95.Round(time.Microsecond),
			p99.Round(time.Microsecond),
			qps)
	}
	if len(txNames) > 0 {
		fmt.Fprintf(w, "\nTRANSACTION\tCOUNT\tERRORS\tAVG\tp50\tp95\tp99\tTPS\n")
		for _, name := range txNames {
			s := stats[name]
			var avg time.Duration
			if s.count > 0 {
				avg = s.totalLatency / time.Duration(s.count)
			}
			p50, p95, p99 := percentiles(s.latencies)
			tps := float64(s.count) / elapsed.Seconds()
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\t%s\t%s\t%.1f\n", name, s.count, s.errors,
				avg.Round(time.Microsecond),
				p50.Round(time.Microsecond),
				p95.Round(time.Microsecond),
				p99.Round(time.Microsecond),
				tps)
		}
	}
	w.Flush()
}

func printSummary(stats map[string]*queryStats, start time.Time, numWorkers int) {
	elapsed := time.Since(start)

	var queryNames, txNames []string
	for name, s := range stats {
		if s.isTransaction {
			txNames = append(txNames, name)
		} else {
			queryNames = append(queryNames, name)
		}
	}
	slices.Sort(queryNames)
	slices.Sort(txNames)

	var totalCount, totalErrors int64
	for _, s := range stats {
		if !s.isTransaction {
			totalCount += s.count
			totalErrors += s.errors
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "\nsummary\n")
	fmt.Fprintf(w, "Duration:\t%s\n", elapsed.Round(time.Millisecond))
	fmt.Fprintf(w, "Workers:\t%d\n", numWorkers)
	fmt.Fprintf(w, "\nQUERY\tCOUNT\tERRORS\tAVG\tp50\tp95\tp99\tQPS\n")
	for _, name := range queryNames {
		s := stats[name]
		var avg time.Duration
		if s.count > 0 {
			avg = s.totalLatency / time.Duration(s.count)
		}
		p50, p95, p99 := percentiles(s.latencies)
		qps := float64(s.count) / elapsed.Seconds()
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\t%s\t%s\t%.1f\n", name, s.count, s.errors,
			avg.Round(time.Microsecond),
			p50.Round(time.Microsecond),
			p95.Round(time.Microsecond),
			p99.Round(time.Microsecond),
			qps)
	}
	if len(txNames) > 0 {
		fmt.Fprintf(w, "\nTRANSACTION\tCOUNT\tERRORS\tAVG\tp50\tp95\tp99\tTPS\n")
		for _, name := range txNames {
			s := stats[name]
			var avg time.Duration
			if s.count > 0 {
				avg = s.totalLatency / time.Duration(s.count)
			}
			p50, p95, p99 := percentiles(s.latencies)
			tps := float64(s.count) / elapsed.Seconds()
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\t%s\t%s\t%.1f\n", name, s.count, s.errors,
				avg.Round(time.Microsecond),
				p50.Round(time.Microsecond),
				p95.Round(time.Microsecond),
				p99.Round(time.Microsecond),
				tps)
		}
	}
	tpm := float64(totalCount) / elapsed.Minutes()
	fmt.Fprintf(w, "\nTransactions:\t%d\n", totalCount)
	fmt.Fprintf(w, "Errors:\t%d\n", totalErrors)
	fmt.Fprintf(w, "tpm:\t%.1f\n", tpm)
	w.Flush()
}

// percentiles returns p50, p95, and p99 from a slice of latencies.
// It sorts a copy to avoid mutating the original (which is still
// being appended to during progress reporting).
func percentiles(latencies []time.Duration) (p50, p95, p99 time.Duration) {
	n := len(latencies)
	if n == 0 {
		return 0, 0, 0
	}

	sorted := make([]time.Duration, n)
	copy(sorted, latencies)
	slices.Sort(sorted)

	p50 = sorted[n*50/100]
	p95 = sorted[n*95/100]

	i99 := n * 99 / 100
	if i99 >= n {
		i99 = n - 1
	}
	p99 = sorted[i99]

	return p50, p95, p99
}
