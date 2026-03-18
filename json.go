package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/bcross2/pipeforge-tui/internal/commands"
	"github.com/bcross2/pipeforge-tui/internal/pipeline"
)

// Result types for --build, --preview, --run

type buildResult struct {
	Command string `json:"command"`
}

type previewResult struct {
	Command  string   `json:"command"`
	Lines    []string `json:"lines"`
	RowCount int      `json:"row_count"`
}

type runResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
}

type dataResult struct {
	File    string   `json:"file"`
	Columns []string `json:"columns"`
	Rows    int      `json:"rows"`
	Data    []string `json:"data"`
}

type errorResult struct {
	Error string `json:"error"`
}

// Schema types for --schema

type schemaResult struct {
	Groups   []schemaGroup              `json:"groups"`
	Commands map[string]schemaCommand   `json:"commands"`
}

type schemaGroup struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type schemaCommand struct {
	Group       string         `json:"group"`
	Description string         `json:"description"`
	Config      []schemaField  `json:"config"`
	Defaults    map[string]any `json:"defaults"`
}

type schemaField struct {
	Key         string         `json:"key"`
	Type        string         `json:"type"`
	Label       string         `json:"label"`
	Placeholder string         `json:"placeholder,omitempty"`
	Hint        string         `json:"hint,omitempty"`
	Options     []schemaOption `json:"options,omitempty"`
}

type schemaOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Validate types for --validate

type validateResult struct {
	Valid  bool   `json:"valid"`
	Blocks int    `json:"blocks"`
	Error  string `json:"error,omitempty"`
}

// Explain types for --explain

type explainResult struct {
	Command string        `json:"command"`
	Steps   []explainStep `json:"steps"`
}

type explainStep struct {
	Step       int    `json:"step"`
	Command    string `json:"command"`
	Summary    string `json:"summary"`
	InputRows  int    `json:"input_rows"`
	OutputRows int    `json:"output_rows"`
}

// Schema builder

func buildSchema() schemaResult {
	groups := make([]schemaGroup, len(commands.Groups))
	for i, g := range commands.Groups {
		groups[i] = schemaGroup{ID: g.ID, Label: g.Label}
	}

	cmds := make(map[string]schemaCommand)
	for _, key := range commands.OrderedCommands() {
		def := commands.Registry[key]
		fields := make([]schemaField, len(def.Config))
		for i, f := range def.Config {
			sf := schemaField{
				Key:         f.Key,
				Type:        fieldTypeName(f.Type),
				Label:       f.Label,
				Placeholder: f.Placeholder,
				Hint:        f.Hint,
			}
			for _, o := range f.Options {
				sf.Options = append(sf.Options, schemaOption{Value: o.Value, Label: o.Label})
			}
			fields[i] = sf
		}
		cmds[key] = schemaCommand{
			Group:       def.Group,
			Description: def.Excel,
			Config:      fields,
			Defaults:    def.Defaults,
		}
	}

	return schemaResult{Groups: groups, Commands: cmds}
}

func fieldTypeName(ft commands.FieldType) string {
	switch ft {
	case commands.FieldText:
		return "text"
	case commands.FieldCheck:
		return "bool"
	case commands.FieldNumber:
		return "number"
	case commands.FieldSelect:
		return "select"
	default:
		return "text"
	}
}

// JSON pipeline input

func parseJSONPipeline() ([]pipeline.Block, error) {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	var blocks []pipeline.Block
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("empty pipeline")
	}
	for i, b := range blocks {
		if _, ok := commands.Registry[b.Type]; !ok {
			return nil, fmt.Errorf("block %d: unknown command %q", i, b.Type)
		}
		def := commands.Registry[b.Type]
		if b.Config == nil {
			blocks[i].Config = make(map[string]any)
		}
		for k, v := range def.Defaults {
			if _, exists := blocks[i].Config[k]; !exists {
				blocks[i].Config[k] = v
			}
		}
	}
	return blocks, nil
}

// JSON output helpers

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func exitJSONError(msg string) {
	writeJSON(errorResult{Error: msg})
	os.Exit(1)
}
