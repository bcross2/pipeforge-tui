package model

import (
	"fmt"
	"strconv"

	"github.com/bcross2/pipeforge-tui/internal/commands"
	"github.com/bcross2/pipeforge-tui/internal/pipeline"
	"github.com/bcross2/pipeforge-tui/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Pipeline        []pipeline.Block
	SelectedIdx     int
	NextID          int
	ActivePanel     ui.FocusPanel
	LibraryCursor   int
	PipelineCursor  int
	InspectorCursor int
	Editing         bool
	TextInputs      []textinput.Model
	Width           int
	Height          int
	orderedCmds     []string
	FileName        string
	FileData        string
	ShowExplain     bool
}

func New(fileName, fileData string) Model {
	return Model{
		SelectedIdx: -1,
		NextID:      1,
		ActivePanel: ui.PanelLibrary,
		orderedCmds: commands.OrderedCommands(),
		FileName:    fileName,
		FileData:    fileData,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "q", "ctrl+c":
			if !m.Editing {
				return m, tea.Quit
			}
		case "ctrl+x":
			m.Pipeline = nil
			m.SelectedIdx = -1
			m.PipelineCursor = 0
			m.TextInputs = nil
			return m, nil
		case "e":
			if !m.Editing {
				m.ShowExplain = !m.ShowExplain
				return m, nil
			}
		case "tab":
			if !m.Editing {
				m.ActivePanel = (m.ActivePanel + 1) % 3
				return m, nil
			}
		case "shift+tab", "btab":
			if !m.Editing {
				m.ActivePanel = (m.ActivePanel + 2) % 3
				return m, nil
			}
		}

		// Panel-specific keys
		switch m.ActivePanel {
		case ui.PanelLibrary:
			return m.updateLibrary(msg)
		case ui.PanelPipeline:
			return m.updatePipeline(msg)
		case ui.PanelInspector:
			return m.updateInspector(msg)
		}
	}
	return m, nil
}

func (m Model) updateLibrary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.LibraryCursor > 0 {
			m.LibraryCursor--
		}
	case "down", "j":
		if m.LibraryCursor < len(m.orderedCmds)-1 {
			m.LibraryCursor++
		}
	case "enter":
		cmdType := m.orderedCmds[m.LibraryCursor]
		block := pipeline.NewBlock(m.NextID, cmdType)
		m.NextID++
		m.Pipeline = append(m.Pipeline, block)
		m.SelectedIdx = len(m.Pipeline) - 1
		m.PipelineCursor = m.SelectedIdx
		m.rebuildInputs()
		m.ActivePanel = ui.PanelInspector
	}
	return m, nil
}

func (m Model) updatePipeline(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.PipelineCursor > 0 {
			m.PipelineCursor--
		}
	case "down", "j":
		if m.PipelineCursor < len(m.Pipeline)-1 {
			m.PipelineCursor++
		}
	case "enter":
		if len(m.Pipeline) > 0 {
			if m.SelectedIdx == m.PipelineCursor {
				m.SelectedIdx = -1
				m.TextInputs = nil
			} else {
				m.SelectedIdx = m.PipelineCursor
				m.rebuildInputs()
			}
		}
	case "d", "delete", "backspace":
		if len(m.Pipeline) > 0 && m.PipelineCursor < len(m.Pipeline) {
			m.Pipeline = append(m.Pipeline[:m.PipelineCursor], m.Pipeline[m.PipelineCursor+1:]...)
			if m.SelectedIdx == m.PipelineCursor {
				m.SelectedIdx = -1
				m.TextInputs = nil
			} else if m.SelectedIdx > m.PipelineCursor {
				m.SelectedIdx--
			}
			if m.PipelineCursor >= len(m.Pipeline) && m.PipelineCursor > 0 {
				m.PipelineCursor--
			}
		}
	}
	return m, nil
}

func (m Model) updateInspector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.SelectedIdx < 0 || m.SelectedIdx >= len(m.Pipeline) {
		return m, nil
	}

	block := &m.Pipeline[m.SelectedIdx]
	def := commands.Registry[block.Type]

	if m.Editing {
		switch msg.Type {
		case tea.KeyEscape, tea.KeyEnter:
			m.Editing = false
			m.syncInputsToConfig()
			return m, nil
		case tea.KeyTab, tea.KeyShiftTab:
			// Let tab pass through to global handler
			m.Editing = false
			m.syncInputsToConfig()
			m.ActivePanel = (m.ActivePanel + 1) % 3
			return m, nil
		default:
			m.updateActiveInput(msg)
			m.syncInputsToConfig()
			return m, nil
		}
	}

	switch msg.String() {
	case "up", "k":
		if m.InspectorCursor > 0 {
			m.InspectorCursor--
		}
	case "down", "j":
		if m.InspectorCursor < len(def.Config)-1 {
			m.InspectorCursor++
		}
	case "enter", " ":
		if m.InspectorCursor < len(def.Config) {
			field := def.Config[m.InspectorCursor]
			switch field.Type {
			case commands.FieldCheck:
				current := block.GetBool(field.Key)
				block.Config[field.Key] = !current
				// Sync awk conditionPreset -> condition
				if block.Type == "awk" && field.Key == "conditionPreset" {
					// Not a checkbox, handled in select
				}
			case commands.FieldText, commands.FieldNumber:
				m.Editing = true
				m.focusInput(m.InspectorCursor)
			case commands.FieldSelect:
				// Cycle through options
				current := block.GetString(field.Key)
				opts := field.Options
				idx := 0
				for i, o := range opts {
					if o.Value == current {
						idx = i
						break
					}
				}
				idx = (idx + 1) % len(opts)
				block.Config[field.Key] = opts[idx].Value
				// If awk conditionPreset, sync to condition
				if block.Type == "awk" && field.Key == "conditionPreset" {
					block.Config["condition"] = opts[idx].Value
					m.rebuildInputs()
				}
			}
		}
	}
	return m, nil
}

func (m *Model) rebuildInputs() {
	if m.SelectedIdx < 0 || m.SelectedIdx >= len(m.Pipeline) {
		m.TextInputs = nil
		m.InspectorCursor = 0
		return
	}

	block := m.Pipeline[m.SelectedIdx]
	def := commands.Registry[block.Type]

	var inputs []textinput.Model
	for _, field := range def.Config {
		if field.Type == commands.FieldText || field.Type == commands.FieldNumber {
			ti := textinput.New()
			ti.Placeholder = field.Placeholder
			ti.CharLimit = 256
			ti.Width = 24

			val := block.GetString(field.Key)
			if val == "" && field.Type == commands.FieldNumber {
				n := block.GetInt(field.Key)
				if n > 0 {
					val = fmt.Sprintf("%d", n)
				}
			}
			ti.SetValue(val)
			inputs = append(inputs, ti)
		}
	}
	m.TextInputs = inputs
	m.InspectorCursor = 0
}

func (m *Model) focusInput(fieldIdx int) {
	if m.SelectedIdx < 0 || m.SelectedIdx >= len(m.Pipeline) {
		return
	}
	def := commands.Registry[m.Pipeline[m.SelectedIdx].Type]

	inputIdx := 0
	for i, field := range def.Config {
		if field.Type == commands.FieldText || field.Type == commands.FieldNumber {
			if i == fieldIdx {
				break
			}
			inputIdx++
		}
	}

	for i := range m.TextInputs {
		m.TextInputs[i].Blur()
	}
	if inputIdx < len(m.TextInputs) {
		m.TextInputs[inputIdx].Focus()
	}
}

func (m *Model) updateActiveInput(msg tea.KeyMsg) {
	for i := range m.TextInputs {
		if m.TextInputs[i].Focused() {
			var cmd tea.Cmd
			m.TextInputs[i], cmd = m.TextInputs[i].Update(msg)
			_ = cmd
			return
		}
	}
}

func (m *Model) syncInputsToConfig() {
	if m.SelectedIdx < 0 || m.SelectedIdx >= len(m.Pipeline) {
		return
	}

	block := &m.Pipeline[m.SelectedIdx]
	def := commands.Registry[block.Type]

	inputIdx := 0
	for _, field := range def.Config {
		if field.Type == commands.FieldText || field.Type == commands.FieldNumber {
			if inputIdx < len(m.TextInputs) {
				val := m.TextInputs[inputIdx].Value()
				if field.Type == commands.FieldNumber {
					if n, err := strconv.Atoi(val); err == nil {
						block.Config[field.Key] = n
					} else if val == "" {
						block.Config[field.Key] = 0
					}
				} else {
					block.Config[field.Key] = val
				}
				inputIdx++
			}
		}
	}
}

func (m Model) View() string {
	if m.Width == 0 {
		return "Loading..."
	}

	return ui.RenderLayout(ui.LayoutParams{
		Width:           m.Width,
		Height:          m.Height,
		ActivePanel:     m.ActivePanel,
		LibraryCursor:   m.LibraryCursor,
		PipelineCursor:  m.PipelineCursor,
		InspectorCursor: m.InspectorCursor,
		Editing:         m.Editing,
		Blocks:          m.Pipeline,
		SelectedIdx:     m.SelectedIdx,
		Inputs:          m.TextInputs,
		FileName:        m.FileName,
		FileData:        m.FileData,
		ShowExplain:     m.ShowExplain,
	})
}
