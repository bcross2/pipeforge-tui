package ui

import (
	"fmt"

	"github.com/bcross2/pipeforge-tui/internal/pipeline"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type FocusPanel int

const (
	PanelLibrary FocusPanel = iota
	PanelPipeline
	PanelInspector
)

type LayoutParams struct {
	Width           int
	Height          int
	ActivePanel     FocusPanel
	LibraryCursor   int
	PipelineCursor  int
	InspectorCursor int
	Editing         bool
	Blocks          []pipeline.Block
	SelectedIdx     int
	Inputs          []textinput.Model
	FileName        string
	FileData        string
	ShowExplain     bool
}

func RenderLayout(p LayoutParams) string {
	w := p.Width
	h := p.Height
	if w < 40 {
		w = 40
	}
	if h < 12 {
		h = 12
	}

	// Panel widths — scale with terminal width
	libraryWidth := w * 22 / 100
	inspectorWidth := w * 24 / 100
	if libraryWidth < 18 {
		libraryWidth = 18
	}
	if inspectorWidth < 20 {
		inspectorWidth = 20
	}
	canvasWidth := w - libraryWidth - inspectorWidth
	if canvasWidth < 16 {
		canvasWidth = 16
	}

	// Fixed rows: topbar=1, help=1 → 2 rows overhead
	availHeight := h - 2

	// 50/50 split between canvas and preview
	canvasHeight := availHeight / 2
	previewHeight := availHeight - canvasHeight
	if canvasHeight < 6 {
		canvasHeight = 6
	}
	if previewHeight < 4 {
		previewHeight = 4
	}

	// Side panels span full available height
	panelHeight := availHeight

	// Top bar
	brand := BrandStyle.Render("PIPEFORGE")
	blockCount := DimStyle.Render(pipelineStatus(len(p.Blocks)))
	fileInfo := DimStyle.Render(p.FileName)
	topBar := StatusBarStyle.Width(w).Render(brand + "  " + blockCount + "  " + fileInfo)

	// Inner widths (subtract border: 2 + padding: 2 = 4)
	libInner := libraryWidth - 4
	canInner := canvasWidth - 4
	inspInner := inspectorWidth - 4

	// Panels
	library := RenderLibrary(p.LibraryCursor, p.ActivePanel == PanelLibrary, panelHeight-2, libInner)

	var selectedBlock *pipeline.Block
	if p.SelectedIdx >= 0 && p.SelectedIdx < len(p.Blocks) {
		selectedBlock = &p.Blocks[p.SelectedIdx]
	}

	canvas := RenderCanvas(p.Blocks, p.SelectedIdx, p.PipelineCursor, p.ActivePanel == PanelPipeline, canInner, canvasHeight-2, p.FileName)

	previewRows := previewHeight - 5 // borders + header + separator
	if previewRows < 2 {
		previewRows = 2
	}
	preview := RenderPreview(p.Blocks, p.SelectedIdx, canvasWidth-4, p.FileData, previewRows, p.ShowExplain)

	// Stack canvas + preview vertically in the center column
	centerCol := lipgloss.JoinVertical(lipgloss.Left,
		panelBox(canvas, p.ActivePanel == PanelPipeline, canvasWidth, canvasHeight),
		panelBox(preview, false, canvasWidth, previewHeight),
	)

	inspector := RenderInspector(selectedBlock, p.InspectorCursor, p.ActivePanel == PanelInspector, p.Editing, p.Inputs, panelHeight, inspInner)

	libBox := panelBox(library, p.ActivePanel == PanelLibrary, libraryWidth, panelHeight)
	inspBox := panelBox(inspector, p.ActivePanel == PanelInspector, inspectorWidth, panelHeight)

	middle := lipgloss.JoinHorizontal(lipgloss.Top, libBox, centerCol, inspBox)

	// Help bar
	help := HelpStyle.Width(w).Render("  tab  enter  d  e  ctrl+x  q")

	return lipgloss.JoinVertical(lipgloss.Left, topBar, middle, help)
}

func panelBox(content string, focused bool, width, height int) string {
	borderColor := Border
	if focused {
		borderColor = Bone
	}
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}
	return lipgloss.NewStyle().
		Width(inner).
		MaxWidth(width).
		Height(innerH).
		MaxHeight(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(string(borderColor))).
		Render(content)
}

func pipelineStatus(n int) string {
	if n == 0 {
		return "0 blocks"
	}
	return fmt.Sprintf("%d block%s", n, plural(n))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
