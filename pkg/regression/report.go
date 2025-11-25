package regression

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// ReportFormat specifies the output format for regression reports
type ReportFormat int

const (
	FormatText ReportFormat = iota
	FormatMarkdown
	FormatJSON
)

// GenerateReport creates a regression report
func GenerateReport(w io.Writer, regressions []*Regression, format ReportFormat) error {
	switch format {
	case FormatText:
		return generateTextReport(w, regressions)
	case FormatMarkdown:
		return generateMarkdownReport(w, regressions)
	case FormatJSON:
		return generateJSONReport(w, regressions)
	default:
		return fmt.Errorf("unknown report format: %d", format)
	}
}

// generateTextReport creates a plain text report
func generateTextReport(w io.Writer, regressions []*Regression) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	if len(regressions) == 0 {
		fmt.Fprintln(w, "✓ No performance regressions detected")
		return nil
	}

	// Group by severity
	groups := GroupBySeverity(regressions)

	// Header
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════════════════")
	fmt.Fprintln(w, "                    PERFORMANCE REGRESSION REPORT")
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════════════════")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Total Regressions: %d\n", len(regressions))
	fmt.Fprintf(w, "  Critical: %d\n", len(groups[SeverityCritical]))
	fmt.Fprintf(w, "  Warning:  %d\n", len(groups[SeverityWarning]))
	fmt.Fprintln(w)

	// Critical regressions
	if len(groups[SeverityCritical]) > 0 {
		fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────────")
		fmt.Fprintln(w, "CRITICAL REGRESSIONS (> 25% slower)")
		fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────────")
		fmt.Fprintln(w)
		printRegressionTable(tw, groups[SeverityCritical])
		fmt.Fprintln(w)
	}

	// Warning regressions
	if len(groups[SeverityWarning]) > 0 {
		fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────────")
		fmt.Fprintln(w, "WARNINGS (10-25% slower)")
		fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────────")
		fmt.Fprintln(w)
		printRegressionTable(tw, groups[SeverityWarning])
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════════════════")

	return nil
}

// printRegressionTable prints a table of regressions
func printRegressionTable(w *tabwriter.Writer, regressions []*Regression) {
	// Sort by percent change (worst first)
	sort.Slice(regressions, func(i, j int) bool {
		return regressions[i].PercentChange > regressions[j].PercentChange
	})

	fmt.Fprintf(w, "Benchmark\tMetric\tBaseline\tCurrent\tChange\n")
	fmt.Fprintf(w, "─────────\t──────\t────────\t───────\t──────\n")

	for _, r := range regressions {
		baseValue := formatMetricValue(r.Metric, r.Baseline)
		currValue := formatMetricValue(r.Metric, r.Current)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t+%.1f%%\n",
			truncate(r.BenchmarkName, 40),
			r.Metric,
			baseValue,
			currValue,
			r.PercentChange,
		)
	}

	w.Flush()
}

// generateMarkdownReport creates a GitHub-flavored markdown report
func generateMarkdownReport(w io.Writer, regressions []*Regression) error {
	if len(regressions) == 0 {
		fmt.Fprintln(w, "## ✅ Performance Regression Check")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No performance regressions detected.")
		return nil
	}

	groups := GroupBySeverity(regressions)

	// Header
	if len(groups[SeverityCritical]) > 0 {
		fmt.Fprintln(w, "## ❌ Performance Regressions Detected")
	} else {
		fmt.Fprintln(w, "## ⚠️ Performance Warnings")
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "**Total Regressions:** %d\n", len(regressions))
	fmt.Fprintf(w, "- 🔴 Critical: %d\n", len(groups[SeverityCritical]))
	fmt.Fprintf(w, "- 🟡 Warning: %d\n\n", len(groups[SeverityWarning]))

	// Critical regressions
	if len(groups[SeverityCritical]) > 0 {
		fmt.Fprintln(w, "### 🔴 Critical Regressions (>25% slower)")
		fmt.Fprintln(w)
		printMarkdownTable(w, groups[SeverityCritical])
		fmt.Fprintln(w)
	}

	// Warning regressions
	if len(groups[SeverityWarning]) > 0 {
		fmt.Fprintln(w, "### 🟡 Warnings (10-25% slower)")
		fmt.Fprintln(w)
		printMarkdownTable(w, groups[SeverityWarning])
		fmt.Fprintln(w)
	}

	// Recommendations
	fmt.Fprintln(w, "### 📋 Recommendations")
	fmt.Fprintln(w)
	if len(groups[SeverityCritical]) > 0 {
		fmt.Fprintln(w, "- **Action Required:** Critical regressions detected. Please investigate and fix before merging.")
	} else {
		fmt.Fprintln(w, "- Review the warnings and consider optimizations if appropriate.")
	}
	fmt.Fprintln(w, "- Run `make bench-compare` locally to reproduce results.")
	fmt.Fprintln(w, "- Use `go test -bench=<name> -cpuprofile=cpu.prof` to profile specific benchmarks.")
	fmt.Fprintln(w)

	return nil
}

// printMarkdownTable prints a markdown table of regressions
func printMarkdownTable(w io.Writer, regressions []*Regression) {
	sort.Slice(regressions, func(i, j int) bool {
		return regressions[i].PercentChange > regressions[j].PercentChange
	})

	fmt.Fprintln(w, "| Benchmark | Metric | Baseline | Current | Change |")
	fmt.Fprintln(w, "|-----------|--------|----------|---------|--------|")

	for _, r := range regressions {
		baseValue := formatMetricValue(r.Metric, r.Baseline)
		currValue := formatMetricValue(r.Metric, r.Current)

		fmt.Fprintf(w, "| %s | %s | %s | %s | +%.1f%% |\n",
			escapeMarkdown(truncate(r.BenchmarkName, 50)),
			r.Metric,
			baseValue,
			currValue,
			r.PercentChange,
		)
	}
}

// generateJSONReport creates a JSON report
func generateJSONReport(w io.Writer, regressions []*Regression) error {
	// This would use json.Marshal in production
	// For now, simplified implementation
	fmt.Fprintln(w, "{")
	fmt.Fprintf(w, "  \"timestamp\": \"%s\",\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(w, "  \"total_regressions\": %d,\n", len(regressions))
	fmt.Fprintln(w, "  \"regressions\": [")

	for i, r := range regressions {
		fmt.Fprintln(w, "    {")
		fmt.Fprintf(w, "      \"benchmark\": \"%s\",\n", r.BenchmarkName)
		fmt.Fprintf(w, "      \"metric\": \"%s\",\n", r.Metric)
		fmt.Fprintf(w, "      \"severity\": \"%s\",\n", r.Severity)
		fmt.Fprintf(w, "      \"percent_change\": %.2f\n", r.PercentChange)
		if i < len(regressions)-1 {
			fmt.Fprintln(w, "    },")
		} else {
			fmt.Fprintln(w, "    }")
		}
	}

	fmt.Fprintln(w, "  ]")
	fmt.Fprintln(w, "}")

	return nil
}

// GenerateSummaryReport creates a compact summary suitable for CI output
func GenerateSummaryReport(w io.Writer, regressions []*Regression) error {
	groups := GroupBySeverity(regressions)

	if len(regressions) == 0 {
		fmt.Fprintln(w, "✓ No performance regressions detected")
		return nil
	}

	fmt.Fprintf(w, "Performance Check: %d regression(s) detected\n", len(regressions))
	fmt.Fprintf(w, "  Critical: %d\n", len(groups[SeverityCritical]))
	fmt.Fprintf(w, "  Warning:  %d\n", len(groups[SeverityWarning]))

	if len(groups[SeverityCritical]) > 0 {
		fmt.Fprintln(w, "\nCritical regressions:")
		for _, r := range groups[SeverityCritical] {
			fmt.Fprintf(w, "  - %s: %s +%.1f%%\n", truncate(r.BenchmarkName, 50), r.Metric, r.PercentChange)
		}
	}

	return nil
}

// GenerateTrendReport creates a historical trend report
func GenerateTrendReport(w io.Writer, trends map[string][]*BenchmarkResult) error {
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════════════════")
	fmt.Fprintln(w, "                    PERFORMANCE TREND REPORT")
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════════════════")
	fmt.Fprintln(w)

	// Get sorted benchmark names
	names := make([]string, 0, len(trends))
	for name := range trends {
		names = append(names, name)
	}
	sort.Strings(names)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	for _, name := range names {
		results := trends[name]
		if len(results) == 0 {
			continue
		}

		fmt.Fprintf(w, "\n%s\n", name)
		fmt.Fprintln(w, strings.Repeat("─", 70))

		fmt.Fprintf(tw, "Timestamp\tns/op\tB/op\tallocs/op\n")

		for _, r := range results {
			fmt.Fprintf(tw, "%s\t%.0f\t%d\t%d\n",
				r.Timestamp.Format("2006-01-02 15:04"),
				r.NsPerOp,
				r.BytesPerOp,
				r.AllocsPerOp,
			)
		}

		tw.Flush()
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════════════════")

	return nil
}

// Helper functions

func formatMetricValue(metric string, result *BenchmarkResult) string {
	switch metric {
	case "ns/op":
		return fmt.Sprintf("%.0f ns/op", result.NsPerOp)
	case "B/op":
		return fmt.Sprintf("%d B/op", result.BytesPerOp)
	case "allocs/op":
		return fmt.Sprintf("%d allocs/op", result.AllocsPerOp)
	default:
		return "N/A"
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"|", "\\|",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
	)
	return replacer.Replace(s)
}
