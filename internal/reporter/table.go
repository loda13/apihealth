package reporter

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/lodatang/apihealth/internal/checker"
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

// RenderTable displays check results in a colorized table
func RenderTable(results []checker.Result) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Target", "Model", "Status", "Response Time", "Status Code", "Error"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	for _, r := range results {
		var status, errorMsg string
		var statusCode string

		if r.Success {
			status = green("✓")
			errorMsg = "-"
		} else {
			if r.Error != nil && r.Error.Type == checker.ErrorTypeRateLimit {
				status = yellow("⚠")
			} else {
				status = red("✗")
			}

			if r.Error != nil {
				errorMsg = truncate(r.Error.Message, 50)
			} else {
				errorMsg = "Unknown error"
			}
		}

		if r.StatusCode > 0 {
			statusCode = fmt.Sprintf("%d", r.StatusCode)
		} else {
			statusCode = "-"
		}

		table.Append([]string{
			r.Name,
			r.Model,
			status,
			fmt.Sprintf("%dms", r.Duration.Milliseconds()),
			statusCode,
			errorMsg,
		})
	}

	table.Render()
}

// truncate shortens a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
