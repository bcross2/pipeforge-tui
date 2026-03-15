package ui

import (
	"fmt"
	"strings"

	"github.com/bcross2/pipeforge-tui/internal/commands"
	"github.com/bcross2/pipeforge-tui/internal/pipeline"
	"github.com/charmbracelet/lipgloss"
)

func RenderCanvas(blocks []pipeline.Block, selectedIdx int, cursor int, focused bool, innerWidth, height int, fileName string) string {
	title := PanelDimTitleStyle.Render("PIPELINE")
	if focused {
		title = PanelTitleStyle.Render("PIPELINE")
	}

	// Pinned command bar at top
	cmd := pipeline.GenerateCommand(blocks, fileName)
	var cmdLine string
	if cmd != "" {
		maxCmd := innerWidth - 4
		cmdText := cmd
		if len(cmdText) > maxCmd {
			cmdText = cmdText[:maxCmd-1] + "…"
		}
		cmdLine = DimStyle.Render("$ ") + CommandTextStyle.Render(cmdText)
	} else {
		cmdLine = DimStyle.Render("$ (empty)")
	}

	if len(blocks) == 0 {
		empty := DimStyle.Render("Enter in Library to add")
		return title + "\n" + cmdLine + "\n\n" + empty
	}

	// Build all content lines, tracking which line each block is on
	type entry struct {
		text     string
		blockIdx int // -1 for non-block lines
	}
	var entries []entry

	// Input file
	entries = append(entries, entry{DimStyle.Render("  " + fileName), -1})
	entries = append(entries, entry{ConnectorStyle.Render("       |"), -1})

	for i, block := range blocks {
		def := commands.Registry[block.Type]
		preview := pipeline.GetConfigPreview(block)

		name := def.Label
		if preview != "" {
			name += " " + DimStyle.Render(truncate(preview, innerWidth-12))
		}

		isSel := i == selectedIdx
		isCur := i == cursor && focused

		var line string
		if isCur || isSel {
			marker := "  "
			if isCur {
				marker = "> "
			}
			line = BlockSelectedStyle.Width(innerWidth - 2).Render(marker + name)
		} else {
			line = BlockStyle.Width(innerWidth - 2).Render("  " + name)
		}
		entries = append(entries, entry{line, i})

		if i < len(blocks)-1 {
			entries = append(entries, entry{ConnectorStyle.Render("       |"), -1})
		}
	}

	// Find cursor line
	cursorLine := 0
	for i, e := range entries {
		if e.blockIdx == cursor {
			cursorLine = i
			break
		}
	}

	// Visible lines: height is now the INNER height (borders already subtracted in layout.go)
	// Overhead: title(1) + cmdLine(1) + top indicator/blank(1) + bottom indicator reserve(1) = 4
	visibleLines := height - 4
	if visibleLines < 3 {
		visibleLines = 3
	}

	// Scroll to keep cursor roughly centered in visible area
	scrollOffset := cursorLine - visibleLines/2
	if scrollOffset > len(entries)-visibleLines {
		scrollOffset = len(entries) - visibleLines
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	end := scrollOffset + visibleLines
	if end > len(entries) {
		end = len(entries)
	}

	var lines []string
	lines = append(lines, title)
	lines = append(lines, cmdLine)

	if scrollOffset > 0 {
		lines = append(lines, DimStyle.Render(fmt.Sprintf(" ...%d above", scrollOffset)))
	} else {
		lines = append(lines, "")
	}

	for i := scrollOffset; i < end; i++ {
		lines = append(lines, entries[i].text)
	}

	if end < len(entries) {
		lines = append(lines, DimStyle.Render(fmt.Sprintf(" ...%d below", len(entries)-end)))
	}

	return strings.Join(lines, "\n")
}

func RenderCommandBar(blocks []pipeline.Block, width int, fileName string) string {
	cmd := pipeline.GenerateCommand(blocks, fileName)
	if cmd == "" {
		return StatusBarStyle.Width(width).Render("$ " + DimStyle.Render("(empty pipeline)"))
	}

	label := lipgloss.NewStyle().Foreground(BoneDim).Render("$ ")
	return StatusBarStyle.Width(width).Render(label + strings.TrimSpace(cmd))
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
