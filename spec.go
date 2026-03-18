package main

import (
	"fmt"
	"strings"

	"github.com/bcross2/pipeforge-tui/internal/commands"
	"github.com/bcross2/pipeforge-tui/internal/pipeline"
)

func parseSpec(spec string) ([]pipeline.Block, error) {
	steps := strings.Split(spec, "|")
	var blocks []pipeline.Block
	nextID := 1

	for _, step := range steps {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}

		cmdType := step
		configStr := ""
		if idx := strings.Index(step, ":"); idx >= 0 {
			cmdType = step[:idx]
			configStr = step[idx+1:]
		}
		cmdType = strings.TrimSpace(cmdType)

		if _, ok := commands.Registry[cmdType]; !ok {
			return nil, fmt.Errorf("unknown command: %q", cmdType)
		}

		block := pipeline.NewBlock(nextID, cmdType)
		nextID++

		if configStr != "" {
			opts := splitConfig(configStr)
			for _, opt := range opts {
				opt = strings.TrimSpace(opt)
				if opt == "" {
					continue
				}
				if idx := strings.Index(opt, "="); idx >= 0 {
					key := opt[:idx]
					val := opt[idx+1:]
					val = strings.ReplaceAll(val, ";", ",")
					block.Config[key] = val
				} else {
					block.Config[opt] = true
				}
			}
		}

		blocks = append(blocks, block)
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf("empty pipeline spec")
	}

	return blocks, nil
}

func splitConfig(s string) []string {
	var parts []string
	var current strings.Builder
	inValue := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '=' {
			inValue = true
			current.WriteByte(ch)
		} else if ch == ',' && !inValue {
			parts = append(parts, current.String())
			current.Reset()
		} else if ch == ',' && inValue {
			rest := s[i+1:]
			nextComma := strings.Index(rest, ",")
			nextEq := strings.Index(rest, "=")
			nextToken := rest
			if nextComma >= 0 {
				nextToken = rest[:nextComma]
			}

			isBoolFlag := isBooleanFlag(strings.TrimSpace(nextToken))
			if nextEq >= 0 && (nextComma < 0 || nextEq < nextComma) || isBoolFlag {
				parts = append(parts, current.String())
				current.Reset()
				inValue = false
			} else {
				current.WriteByte(ch)
			}
		} else {
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func isBooleanFlag(s string) bool {
	flags := map[string]bool{
		"ignoreCase": true, "invert": true, "global": true,
		"numeric": true, "reverse": true, "count": true,
		"duplicatesOnly": true, "lines": true, "words": true,
		"chars": true, "squeeze": true, "delete": true,
		"placeholder": true,
		"sortInput":   true,
		"headerIn":    true,
	}
	return flags[s]
}
