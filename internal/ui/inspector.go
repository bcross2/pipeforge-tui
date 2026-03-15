package ui

import (
	"fmt"
	"strings"

	"github.com/bcross2/pipeforge-tui/internal/commands"
	"github.com/bcross2/pipeforge-tui/internal/pipeline"
	"github.com/charmbracelet/bubbles/textinput"
)

func RenderInspector(block *pipeline.Block, fieldCursor int, focused bool, editing bool, inputs []textinput.Model, height int, innerWidth int) string {
	title := PanelDimTitleStyle.Render("INSPECTOR")
	if focused {
		title = PanelTitleStyle.Render("INSPECTOR")
	}

	if block == nil {
		empty := DimStyle.Render("Select a block\nto configure")
		return title + "\n\n" + empty
	}

	def := commands.Registry[block.Type]
	subtitle := LabelStyle.Render("cfg ") + ValueStyle.Render(def.Label)

	var lines []string
	lines = append(lines, title)
	lines = append(lines, subtitle)
	lines = append(lines, "")

	inputIdx := 0
	for i, field := range def.Config {
		active := i == fieldCursor && focused

		switch field.Type {
		case commands.FieldCheck:
			checked := block.GetBool(field.Key)
			box := "[ ]"
			if checked {
				box = "[x]"
			}
			lbl := field.Label
			if len(lbl) > innerWidth-6 {
				lbl = lbl[:innerWidth-6]
			}
			label := fmt.Sprintf("%s %s", box, lbl)
			if active {
				lines = append(lines, FieldActiveStyle.Render("> "+label))
			} else {
				lines = append(lines, FieldLabelStyle.Render("  "+label))
			}

		case commands.FieldText, commands.FieldNumber:
			lbl := field.Label
			if len(lbl) > innerWidth-4 {
				lbl = lbl[:innerWidth-4]
			}
			if active {
				lines = append(lines, FieldActiveStyle.Render("> "+lbl))
			} else {
				lines = append(lines, FieldLabelStyle.Render("  "+lbl))
			}
			if inputIdx < len(inputs) {
				inp := inputs[inputIdx]
				inp.Width = innerWidth - 4
				lines = append(lines, "  "+inp.View())
				inputIdx++
			}

		case commands.FieldSelect:
			lbl := field.Label
			val := block.GetString(field.Key)
			display := "(custom)"
			for _, opt := range field.Options {
				if opt.Value == val {
					display = opt.Label
					break
				}
			}
			text := fmt.Sprintf("%s: %s", lbl, display)
			if len(text) > innerWidth-4 {
				text = text[:innerWidth-4]
			}
			if active {
				lines = append(lines, FieldActiveStyle.Render("> "+text))
			} else {
				lines = append(lines, FieldLabelStyle.Render("  "+text))
			}
		}
	}

	return strings.Join(lines, "\n")
}
