package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bcross2/pipeforge-tui/internal/commands"
	"github.com/bcross2/pipeforge-tui/internal/data"
	"github.com/bcross2/pipeforge-tui/internal/model"
	"github.com/bcross2/pipeforge-tui/internal/pipeline"
	tea "github.com/charmbracelet/bubbletea"
)

const maxPreviewLines = 200

func main() {
	args := os.Args[1:]

	fileName := ""
	spec := ""
	mode := "tui" // tui, build, preview, run

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--file" || arg == "-f":
			if i+1 < len(args) {
				i++
				fileName = args[i]
			}
		case strings.HasPrefix(arg, "--file="):
			fileName = strings.TrimPrefix(arg, "--file=")
		case arg == "--build":
			mode = "build"
			if i+1 < len(args) {
				i++
				spec = args[i]
			}
		case strings.HasPrefix(arg, "--build="):
			mode = "build"
			spec = strings.TrimPrefix(arg, "--build=")
		case arg == "--preview":
			mode = "preview"
			if i+1 < len(args) {
				i++
				spec = args[i]
			}
		case strings.HasPrefix(arg, "--preview="):
			mode = "preview"
			spec = strings.TrimPrefix(arg, "--preview=")
		case arg == "--run":
			mode = "run"
			if i+1 < len(args) {
				i++
				spec = args[i]
			}
		case strings.HasPrefix(arg, "--run="):
			mode = "run"
			spec = strings.TrimPrefix(arg, "--run=")
		case arg == "--help" || arg == "-h":
			printHelp()
			return
		default:
			// Bare argument = filename
			if fileName == "" && !strings.HasPrefix(arg, "-") {
				fileName = arg
			}
		}
	}

	// Load file data
	fileData, displayName := loadFileData(fileName)

	switch mode {
	case "tui":
		p := tea.NewProgram(model.New(displayName, fileData), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "build":
		if spec == "" {
			fmt.Fprintln(os.Stderr, "Usage: pipeforge --build SPEC [--file FILE]")
			os.Exit(1)
		}
		blocks, err := parseSpec(spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(pipeline.GenerateCommand(blocks, displayName))
	case "preview":
		if spec == "" {
			fmt.Fprintln(os.Stderr, "Usage: pipeforge --preview SPEC [--file FILE]")
			os.Exit(1)
		}
		blocks, err := parseSpec(spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cmd := pipeline.GenerateCommand(blocks, displayName)
		fmt.Printf("$ %s\n\n", cmd)
		lines := pipeline.SimulateUpTo(blocks, len(blocks)-1, fileData)
		for _, line := range lines {
			fmt.Println(line)
		}
		fmt.Printf("\n(%d rows)\n", len(lines))
	case "run":
		if spec == "" {
			fmt.Fprintln(os.Stderr, "Usage: pipeforge --run SPEC [--file FILE]")
			os.Exit(1)
		}
		blocks, err := parseSpec(spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cmd := pipeline.GenerateCommand(blocks, displayName)
		fmt.Fprintf(os.Stderr, "$ %s\n", cmd)
		sh := exec.Command("sh", "-c", cmd)
		sh.Stdout = os.Stdout
		sh.Stderr = os.Stderr
		if err := sh.Run(); err != nil {
			os.Exit(1)
		}
	}
}

func loadFileData(fileName string) (fileData string, displayName string) {
	if fileName == "" {
		return data.SampleCSV, "sample_sales.csv"
	}

	raw, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", fileName, err)
		os.Exit(1)
	}

	content := strings.TrimRight(string(raw), "\n\r")

	// For preview, limit to first N lines
	lines := strings.Split(content, "\n")
	if len(lines) > maxPreviewLines {
		fmt.Fprintf(os.Stderr, "Note: previewing first %d of %d lines\n", maxPreviewLines, len(lines))
		content = strings.Join(lines[:maxPreviewLines], "\n")
	}

	return content, fileName
}

func printHelp() {
	fmt.Println(`PipeForge — Visual Linux Pipeline Builder

Usage:
  pipeforge [FILE]              Launch TUI (with optional file)
  pipeforge --build SPEC        Output the shell command
  pipeforge --preview SPEC      Show simulated data preview
  pipeforge --run SPEC          Build and execute the command

Options:
  --file, -f FILE       Input file (default: built-in sample CSV)
  --help, -h            Show this help

SPEC format:
  "command:key=value,flag | command:key=value"

  Commands: grep, awk, cut, sed, tr, sort, uniq, wc, head, tail, tee, xargs

  Boolean flags (no =value):
    grep:pattern=hello,ignoreCase,invert
    sort:key=4,numeric,reverse
    uniq:count

  Text/number options (key=value):
    grep:pattern=North
    cut:fields=1;3
    head:lines=10

  Pipe steps with |

Examples:
  pipeforge data.csv
  pipeforge --file data.csv --build "grep:pattern=North | sort:key=4,numeric"
  pipeforge --file data.csv --preview "cut:fields=1;3 | sort | uniq:count"
  pipeforge --file data.csv --run "grep:pattern=error | wc:lines"
  pipeforge --build "grep:pattern=get,ignoreCase | sort | uniq:count"`)
}

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
	}
	return flags[s]
}
