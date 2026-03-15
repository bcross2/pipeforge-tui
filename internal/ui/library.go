package ui

import (
	"fmt"
	"strings"

	"github.com/bcross2/pipeforge-tui/internal/commands"
)

func RenderLibrary(cursor int, focused bool, height int, innerWidth int) string {
	title := PanelDimTitleStyle.Render("LIBRARY")
	if focused {
		title = PanelTitleStyle.Render("LIBRARY")
	}

	// Build all lines first
	type entry struct {
		text    string
		cmdIdx  int // -1 for group headers and blanks
	}
	var entries []entry

	idx := 0
	ordered := commands.OrderedCommands()
	currentGroup := ""

	for _, key := range ordered {
		def := commands.Registry[key]
		if def.Group != currentGroup {
			currentGroup = def.Group
			for _, g := range commands.Groups {
				if g.ID == currentGroup {
					entries = append(entries, entry{"", -1})
					entries = append(entries, entry{GroupLabelStyle.Render(strings.ToUpper(g.Label)), -1})
					break
				}
			}
		}

		label := fmt.Sprintf("[%s] %-5s", def.Icon, def.Label)
		if len(label) > innerWidth {
			label = label[:innerWidth]
		}

		var line string
		if idx == cursor && focused {
			line = ItemActiveStyle.Width(innerWidth).Render(label)
		} else {
			line = ItemStyle.Width(innerWidth).Render(label)
		}
		entries = append(entries, entry{line, idx})
		idx++
	}

	// Find which line the cursor is on
	cursorLine := 0
	for i, e := range entries {
		if e.cmdIdx == cursor {
			cursorLine = i
			break
		}
	}

	// Visible lines: height is now the INNER height (borders already subtracted in layout.go)
	// Overhead: title(1) + potential "...more" indicator(1) = 2
	visibleLines := height - 2
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
	for i := scrollOffset; i < end; i++ {
		lines = append(lines, entries[i].text)
	}

	// Scroll indicator
	if end < len(entries) {
		lines = append(lines, DimStyle.Render(" ...more"))
	}

	return strings.Join(lines, "\n")
}
