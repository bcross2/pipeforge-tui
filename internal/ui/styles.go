package ui

import "github.com/charmbracelet/lipgloss"

var (
	PanelStyle = lipgloss.NewStyle().
			Background(Charcoal).
			Padding(1, 1)

	PanelActiveStyle = lipgloss.NewStyle().
				Background(Charcoal).
				Padding(1, 1).
				BorderForeground(Bone)

	PanelTitleStyle = lipgloss.NewStyle().
			Foreground(Bone).
			Bold(true).
			MarginBottom(1)

	PanelDimTitleStyle = lipgloss.NewStyle().
				Foreground(BoneDim).
				Bold(true).
				MarginBottom(1)

	GroupLabelStyle = lipgloss.NewStyle().
			Foreground(BoneDim).
			Bold(true)

	ItemStyle = lipgloss.NewStyle().
			Foreground(BoneMuted)

	ItemActiveStyle = lipgloss.NewStyle().
			Foreground(Bone).
			Background(Surface).
			Bold(true)

	IconStyle = lipgloss.NewStyle().
			Foreground(Bone).
			Background(Surface).
			Padding(0, 1)

	BlockStyle = lipgloss.NewStyle().
			Foreground(BoneMuted).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(0, 1)

	BlockSelectedStyle = lipgloss.NewStyle().
				Foreground(Bone).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Bone).
				Padding(0, 1).
				Bold(true)

	ConnectorStyle = lipgloss.NewStyle().
			Foreground(BorderLight)

	CommandBarStyle = lipgloss.NewStyle().
			Foreground(BoneMuted).
			Background(CharcoalLight).
			Padding(0, 1)

	CommandTextStyle = lipgloss.NewStyle().
			Foreground(Bone)

	LabelStyle = lipgloss.NewStyle().
			Foreground(BoneDim).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(Bone)

	DimStyle = lipgloss.NewStyle().
			Foreground(BoneDim)

	PreviewHeaderStyle = lipgloss.NewStyle().
				Foreground(BoneMuted).
				Bold(true)

	TableHeaderStyle = lipgloss.NewStyle().
			Foreground(Bone).
			Bold(true)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(BoneMuted)

	HelpStyle = lipgloss.NewStyle().
			Foreground(BoneDim)

	StatusBarStyle = lipgloss.NewStyle().
			Background(CharcoalLight).
			Foreground(BoneMuted).
			Padding(0, 1)

	BrandStyle = lipgloss.NewStyle().
			Foreground(Bone).
			Bold(true)

	CheckedStyle = lipgloss.NewStyle().
			Foreground(Bone)

	UncheckedStyle = lipgloss.NewStyle().
			Foreground(BoneDim)

	FieldLabelStyle = lipgloss.NewStyle().
			Foreground(BoneMuted)

	FieldActiveStyle = lipgloss.NewStyle().
			Foreground(Bone).
			Bold(true)

	InputStyle = lipgloss.NewStyle().
			Foreground(Bone).
			Background(Surface).
			Padding(0, 1)

	InputActiveStyle = lipgloss.NewStyle().
			Foreground(Bone).
			Background(SurfaceLight).
			Padding(0, 1).
			Bold(true)
)
