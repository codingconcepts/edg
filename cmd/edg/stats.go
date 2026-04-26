package main

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/codingconcepts/edg/pkg/config"
	"github.com/expr-lang/expr"
)

type queryStats struct {
	count         int64
	errors        int64
	commits       int64
	rollbacks     int64
	totalLatency  time.Duration
	latencies     []time.Duration
	isTransaction bool
	printAggExprs []string
	printAggs     []*printAgg
}

type printAgg struct {
	freq     map[string]int64
	sum      float64
	min      float64
	max      float64
	count    int64
	numCount int64
}

func printResults(out io.Writer, results <-chan config.QueryResult, interval time.Duration, start time.Time, numWorkers int, totalDuration time.Duration, warmupDuration time.Duration) map[string]*queryStats {
	stats := map[string]*queryStats{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	warmingUp := warmupDuration > 0
	var warmupDeadline <-chan time.Time
	if warmingUp {
		warmupDeadline = time.After(warmupDuration)
	}

	for {
		select {
		case <-warmupDeadline:
			warmingUp = false
			start = time.Now()
			slog.Info("warmup complete, collecting metrics")

		case r, ok := <-results:
			if !ok {
				printSummary(out, stats, start, numWorkers)
				return stats
			}

			if warmingUp {
				continue
			}

			s := stats[r.Name]
			if s == nil {
				s = &queryStats{}
				stats[r.Name] = s
			}
			s.isTransaction = s.isTransaction || r.IsTransaction

			if r.Err != nil {
				s.errors++
				metricQueryErrors.WithLabelValues(r.Name).Inc()
				continue
			}

			if len(r.PrintValues) > 0 {
				if s.printAggs == nil {
					s.printAggExprs = r.PrintAggs
					s.printAggs = make([]*printAgg, len(r.PrintValues))
					for i := range s.printAggs {
						s.printAggs[i] = &printAgg{
							freq: map[string]int64{},
							min:  math.MaxFloat64,
							max:  -math.MaxFloat64,
						}
					}
				}
				for i, v := range r.PrintValues {
					if i >= len(s.printAggs) {
						continue
					}
					agg := s.printAggs[i]
					agg.freq[v]++
					agg.count++
					if f, err := strconv.ParseFloat(v, 64); err == nil {
						agg.sum += f
						agg.numCount++
						if f < agg.min {
							agg.min = f
						}
						if f > agg.max {
							agg.max = f
						}
					}
				}
			}
			s.count += int64(r.Count)
			s.totalLatency += r.Latency
			s.latencies = append(s.latencies, r.Latency)
			metricQueryDuration.WithLabelValues(r.Name).Observe(r.Latency.Seconds())

			if !r.IsTransaction {
				continue
			}

			switch r.Rollback {
			case true:
				s.rollbacks++
				metricTxRollbacks.WithLabelValues(r.Name).Inc()
			default:
				s.commits++
				metricTxCommits.WithLabelValues(r.Name).Inc()
			}
		case <-ticker.C:
			if !warmingUp {
				printProgress(out, stats, start, totalDuration)
			}
		}
	}
}

func printProgress(out io.Writer, stats map[string]*queryStats, start time.Time, totalDuration time.Duration) {
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

	customPrint := hasPrintConfig(stats)

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "\n%s / %s\n", elapsed.Round(time.Second), totalDuration.Round(time.Second))
	if !customPrint {
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
			fmt.Fprintf(w, "\nTRANSACTION\tCOMMITS\tROLLBACKS\tERRORS\tAVG\tp50\tp95\tp99\tTPS\n")
			for _, name := range txNames {
				s := stats[name]
				var avg time.Duration
				if s.count > 0 {
					avg = s.totalLatency / time.Duration(s.count)
				}
				p50, p95, p99 := percentiles(s.latencies)
				tps := float64(s.count) / elapsed.Seconds()
				fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%.1f\n", name, s.commits, s.rollbacks, s.errors,
					avg.Round(time.Microsecond),
					p50.Round(time.Microsecond),
					p95.Round(time.Microsecond),
					p99.Round(time.Microsecond),
					tps)
			}
		}
	}
	printPrintValues(w, stats, queryNames, txNames)
	w.Flush()
}

func printPrintValues(w *tabwriter.Writer, stats map[string]*queryStats, queryNames, txNames []string) {
	var printNames []string
	for _, name := range queryNames {
		if len(stats[name].printAggs) > 0 {
			printNames = append(printNames, name)
		}
	}
	for _, name := range txNames {
		if len(stats[name].printAggs) > 0 {
			printNames = append(printNames, name)
		}
	}
	if len(printNames) == 0 {
		return
	}
	fmt.Fprintf(w, "\nPRINT\tVALUES\n")
	for _, name := range printNames {
		s := stats[name]
		for i, agg := range s.printAggs {
			aggExpr := ""
			if i < len(s.printAggExprs) {
				aggExpr = s.printAggExprs[i]
			}

			if aggExpr != "" {
				fmt.Fprintf(w, "%s\t%s\n", name, evalAggExpr(aggExpr, agg))
			} else {
				switch {
				case agg.count == 0:
					fmt.Fprintf(w, "%s\t(no data)\n", name)
				case agg.numCount == agg.count:
					avg := agg.sum / float64(agg.count)
					fmt.Fprintf(w, "%s\tmin=%.2f avg=%.2f max=%.2f n=%d\n", name, avg, agg.min, agg.max, agg.count)
				default:
					fmt.Fprintf(w, "%s\t%s\n", name, formatFreq(agg.freq))
				}
			}
		}
	}
}

func evalAggExpr(expression string, agg *printAgg) string {
	var avg, minVal, maxVal float64
	if agg.numCount > 0 {
		avg = agg.sum / float64(agg.numCount)
		minVal = agg.min
		maxVal = agg.max
	}

	aggEnv := map[string]any{
		"min":   minVal,
		"max":   maxVal,
		"avg":   avg,
		"sum":   agg.sum,
		"count": agg.count,
		"freq":  agg.freq,
	}

	program, err := expr.Compile(expression, expr.Env(aggEnv))
	if err != nil {
		return fmt.Sprintf("<compile error: %v>", err)
	}
	result, err := expr.Run(program, aggEnv)
	if err != nil {
		return fmt.Sprintf("<eval error: %v>", err)
	}
	return fmt.Sprintf("%v", result)
}

func formatFreq(freq map[string]int64) string {
	type entry struct {
		val   string
		count int64
	}
	entries := make([]entry, 0, len(freq))
	for v, c := range freq {
		entries = append(entries, entry{v, c})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		return cmp.Compare(b.count, a.count)
	})
	if len(entries) > 10 {
		entries = entries[:10]
	}
	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = fmt.Sprintf("%s=%d", e.val, e.count)
	}
	return strings.Join(parts, " ")
}

func printSummary(out io.Writer, stats map[string]*queryStats, start time.Time, numWorkers int) {
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

	customPrint := hasPrintConfig(stats)

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "\nsummary\n")
	fmt.Fprintf(w, "Duration:\t%s\n", elapsed.Round(time.Millisecond))
	fmt.Fprintf(w, "Workers:\t%d\n", numWorkers)
	if !customPrint {
		var totalCount, totalErrors int64
		for _, s := range stats {
			if !s.isTransaction {
				totalCount += s.count
				totalErrors += s.errors
			}
		}

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
			fmt.Fprintf(w, "\nTRANSACTION\tCOMMITS\tROLLBACKS\tERRORS\tAVG\tp50\tp95\tp99\tTPS\n")
			for _, name := range txNames {
				s := stats[name]
				var avg time.Duration
				if s.count > 0 {
					avg = s.totalLatency / time.Duration(s.count)
				}
				p50, p95, p99 := percentiles(s.latencies)
				tps := float64(s.count) / elapsed.Seconds()
				fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%.1f\n", name, s.commits, s.rollbacks, s.errors,
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
	}
	printPrintValues(w, stats, queryNames, txNames)
	w.Flush()
}

func hasPrintConfig(stats map[string]*queryStats) bool {
	for _, s := range stats {
		if len(s.printAggs) > 0 {
			return true
		}
	}
	return false
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
