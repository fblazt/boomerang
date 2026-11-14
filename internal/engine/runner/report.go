package runner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fblazt/boomerang/internal/models"
)

// FormatReportJSON encodes report into indented JSON suitable for artifact archiving or CI ingestion.
func FormatReportJSON(report models.SuiteRunReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// FormatReportMarkdown renders report as a concise human-readable Markdown suite execution summary.
func FormatReportMarkdown(report models.SuiteRunReport) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Suite Run Report: %s\n\n", report.CollectionName))
	b.WriteString(fmt.Sprintf("- **Run ID**: `%s`\n", report.RunID))
	b.WriteString(fmt.Sprintf("- **Concurrency**: %d worker threads\n", report.Concurrency))
	b.WriteString(fmt.Sprintf("- **Duration**: %d ms\n", report.TotalDurationMs))
	b.WriteString(fmt.Sprintf("- **Requests**: %d passed / %d failed (%d total)\n",
		report.PassedRequests, report.FailedRequests, report.TotalRequests))
	b.WriteString(fmt.Sprintf("- **Assertions**: %d passed / %d failed (%d total)\n\n",
		report.PassedAssertions, report.FailedAssertions, report.TotalAssertions))

	b.WriteString("## Step Breakdown\n\n")
	b.WriteString("| # | Result | Method | Request | Status | Duration | Assertions |\n")
	b.WriteString("|---|--------|--------|---------|--------|----------|------------|\n")

	for _, step := range report.Steps {
		result := "PASS"
		if !step.Passed {
			result = "FAIL"
		}
		passedA := 0
		for _, a := range step.Assertions {
			if a.Passed {
				passedA++
			}
		}
		assertCol := fmt.Sprintf("%d/%d", passedA, len(step.Assertions))
		if len(step.Assertions) == 0 {
			assertCol = "-"
		}
		statusCol := fmt.Sprintf("%d", step.StatusCode)
		if step.Error != "" && step.StatusCode == 0 {
			statusCol = "ERR"
		}
		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %d ms | %s |\n",
			step.Index+1, result, step.Method, step.Name, statusCol, step.DurationMs, assertCol))
	}

	return b.String()
}
