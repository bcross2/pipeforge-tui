package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bcross2/pipeforge-tui/internal/commands"
	"github.com/bcross2/pipeforge-tui/internal/pipeline"
)

func RenderPreview(blocks []pipeline.Block, selectedIdx int, width int, fileData string, maxRows int) string {
	var lines []string
	label := "Input Data"

	if selectedIdx >= 0 && selectedIdx < len(blocks) {
		lines = pipeline.SimulateUpTo(blocks, selectedIdx, fileData)
		def := commands.Registry[blocks[selectedIdx].Type]
		label = "After " + def.Label
	} else if len(blocks) > 0 {
		lines = pipeline.SimulateUpTo(blocks, len(blocks)-1, fileData)
		label = "Output"
	} else {
		lines = strings.Split(fileData, "\n")
	}

	total := len(lines)
	header := PreviewHeaderStyle.Render(fmt.Sprintf(" PREVIEW  %s  %d rows", label, total))

	// Try CSV table
	if len(lines) > 0 && strings.Contains(lines[0], ",") {
		// Limit data rows shown (keep header row)
		displayLines := lines
		truncated := false
		if len(lines) > maxRows+1 { // +1 for header
			displayLines = lines[:maxRows+1]
			truncated = true
		}
		table := renderTable(displayLines, width-2)
		if truncated {
			table += "\n" + DimStyle.Render(fmt.Sprintf(" ...%d more rows", total-maxRows-1))
		}
		return header + "\n" + table
	}

	// Raw lines
	var raw []string
	showLines := lines
	truncated := false
	if len(lines) > maxRows {
		showLines = lines[:maxRows]
		truncated = true
	}
	for i, l := range showLines {
		num := DimStyle.Render(fmt.Sprintf(" %2d ", i+1))
		raw = append(raw, num+ValueStyle.Render(l))
	}
	if truncated {
		raw = append(raw, DimStyle.Render(fmt.Sprintf(" ...%d more rows", total-maxRows)))
	}
	return header + "\n" + strings.Join(raw, "\n")
}

func renderTable(lines []string, maxWidth int) string {
	if len(lines) == 0 {
		return ""
	}

	allFields := make([][]string, len(lines))
	maxCols := 0
	for i, l := range lines {
		allFields[i] = strings.Split(l, ",")
		if len(allFields[i]) > maxCols {
			maxCols = len(allFields[i])
		}
	}

	// Format numbers with commas for display
	displayFields := make([][]string, len(allFields))
	for i, row := range allFields {
		displayFields[i] = make([]string, len(row))
		for j, cell := range row {
			if i == 0 {
				// Header row — no formatting
				displayFields[i][j] = cell
			} else {
				displayFields[i][j] = formatNumber(cell)
			}
		}
	}

	// Column widths based on DATA values only (skip header row)
	colWidths := make([]int, maxCols)
	for i := 1; i < len(displayFields); i++ {
		for j, cell := range displayFields[i] {
			if len(cell) > colWidths[j] {
				colWidths[j] = len(cell)
			}
		}
	}

	// Ensure minimum width for headers (but cap at maxHeaderWidth)
	maxHeaderWidth := 12
	for j := 0; j < maxCols; j++ {
		headerLen := 0
		if j < len(displayFields[0]) {
			headerLen = len(displayFields[0][j])
		}
		if headerLen < maxHeaderWidth && headerLen > colWidths[j] {
			colWidths[j] = headerLen
		} else if colWidths[j] < maxHeaderWidth && colWidths[j] < headerLen {
			// Header is longer than data but we cap it
			if maxHeaderWidth > colWidths[j] {
				colWidths[j] = maxHeaderWidth
			}
		}
	}

	var rows []string

	// Header — truncate long names to fit column width
	var hdr []string
	for j := 0; j < maxCols; j++ {
		cell := ""
		if j < len(displayFields[0]) {
			cell = displayFields[0][j]
		}
		if len(cell) > colWidths[j] {
			cell = cell[:colWidths[j]-1] + "~"
		}
		hdr = append(hdr, TableHeaderStyle.Render(padRight(cell, colWidths[j])))
	}
	rows = append(rows, " "+strings.Join(hdr, "  "))

	// Separator
	var sep []string
	for j := 0; j < maxCols; j++ {
		sep = append(sep, DimStyle.Render(strings.Repeat("-", colWidths[j])))
	}
	rows = append(rows, " "+strings.Join(sep, "--"))

	// Data
	for i := 1; i < len(displayFields); i++ {
		var cells []string
		for j := 0; j < maxCols; j++ {
			cell := ""
			if j < len(displayFields[i]) {
				cell = displayFields[i][j]
			}
			cells = append(cells, TableCellStyle.Render(padRight(cell, colWidths[j])))
		}
		rows = append(rows, " "+strings.Join(cells, "  "))
	}

	return strings.Join(rows, "\n")
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func formatNumber(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// Handle negative numbers
	negative := false
	num := s
	if len(num) > 0 && num[0] == '-' {
		negative = true
		num = num[1:]
	}

	// Split on decimal point
	whole := num
	decimal := ""
	if dot := strings.Index(num, "."); dot >= 0 {
		whole = num[:dot]
		decimal = num[dot:]
	}

	// Check it's actually a number
	if _, err := strconv.Atoi(whole); err != nil {
		return s
	}

	// Insert commas from right
	if len(whole) <= 3 {
		if negative {
			return "-" + whole + decimal
		}
		return whole + decimal
	}

	var buf strings.Builder
	remainder := len(whole) % 3
	if remainder > 0 {
		buf.WriteString(whole[:remainder])
	}
	for i := remainder; i < len(whole); i += 3 {
		if buf.Len() > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(whole[i : i+3])
	}

	result := buf.String() + decimal
	if negative {
		return "-" + result
	}
	return result
}
