package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	mode := "tui" // tui, build, preview, run, schema, validate, explain, data
	jsonMode := false

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
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				spec = args[i]
			}
		case strings.HasPrefix(arg, "--build="):
			mode = "build"
			spec = strings.TrimPrefix(arg, "--build=")
		case arg == "--preview":
			mode = "preview"
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				spec = args[i]
			}
		case strings.HasPrefix(arg, "--preview="):
			mode = "preview"
			spec = strings.TrimPrefix(arg, "--preview=")
		case arg == "--run":
			mode = "run"
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				spec = args[i]
			}
		case strings.HasPrefix(arg, "--run="):
			mode = "run"
			spec = strings.TrimPrefix(arg, "--run=")
		case arg == "--schema":
			mode = "schema"
		case arg == "--data":
			mode = "data"
		case arg == "--validate":
			mode = "validate"
		case arg == "--explain":
			mode = "explain"
		case arg == "--json":
			jsonMode = true
		case arg == "--help" || arg == "-h":
			printHelp()
			return
		default:
			if fileName == "" && !strings.HasPrefix(arg, "-") {
				fileName = arg
			}
		}
	}

	// Schema needs no file data
	if mode == "schema" {
		writeJSON(buildSchema())
		return
	}

	// Data mode: dump input data and exit
	if mode == "data" {
		fileData, displayName := loadFileData(fileName, jsonMode)
		lines := strings.Split(fileData, "\n")
		if jsonMode {
			var columns []string
			if len(lines) > 0 {
				columns = strings.Split(lines[0], ",")
			}
			writeJSON(dataResult{
				File:    displayName,
				Columns: columns,
				Rows:    len(lines) - 1, // exclude header
				Data:    lines,
			})
		} else {
			fmt.Printf("# %s (%d data rows)\n\n", displayName, len(lines)-1)
			for _, line := range lines {
				fmt.Println(line)
			}
		}
		return
	}

	// Validate and explain always use JSON I/O
	if mode == "validate" || mode == "explain" {
		jsonMode = true
	}

	fileData, displayName := loadFileData(fileName, jsonMode)

	switch mode {
	case "tui":
		if jsonMode {
			exitJSONError("--json cannot be used with TUI mode")
		}
		p := tea.NewProgram(model.New(displayName, fileData), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "build":
		blocks, err := resolveBlocks(spec, jsonMode)
		if err != nil {
			if jsonMode {
				exitJSONError(err.Error())
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cmd := pipeline.GenerateCommand(blocks, displayName)
		if jsonMode {
			writeJSON(buildResult{Command: cmd})
		} else {
			fmt.Println(cmd)
		}
	case "preview":
		blocks, err := resolveBlocks(spec, jsonMode)
		if err != nil {
			if jsonMode {
				exitJSONError(err.Error())
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cmd := pipeline.GenerateCommand(blocks, displayName)
		lines := pipeline.SimulateUpTo(blocks, len(blocks)-1, fileData)
		if jsonMode {
			writeJSON(previewResult{
				Command:  cmd,
				Lines:    lines,
				RowCount: len(lines),
			})
		} else {
			fmt.Printf("$ %s\n\n", cmd)
			for _, line := range lines {
				fmt.Println(line)
			}
			fmt.Printf("\n(%d rows)\n", len(lines))
		}
	case "run":
		blocks, err := resolveBlocks(spec, jsonMode)
		if err != nil {
			if jsonMode {
				exitJSONError(err.Error())
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cmd := pipeline.GenerateCommand(blocks, displayName)
		if jsonMode {
			sh := exec.Command("sh", "-c", cmd)
			var stdoutBuf, stderrBuf bytes.Buffer
			sh.Stdout = &stdoutBuf
			sh.Stderr = &stderrBuf
			exitCode := 0
			if err := sh.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					exitCode = 1
				}
			}
			writeJSON(runResult{
				Command:  cmd,
				ExitCode: exitCode,
				Stdout:   stdoutBuf.String(),
				Stderr:   stderrBuf.String(),
			})
			if exitCode != 0 {
				os.Exit(exitCode)
			}
		} else {
			fmt.Fprintf(os.Stderr, "$ %s\n", cmd)
			sh := exec.Command("sh", "-c", cmd)
			sh.Stdout = os.Stdout
			sh.Stderr = os.Stderr
			if err := sh.Run(); err != nil {
				os.Exit(1)
			}
		}
	case "validate":
		blocks, err := parseJSONPipeline()
		if err != nil {
			writeJSON(validateResult{Valid: false, Blocks: 0, Error: err.Error()})
			os.Exit(1)
		}
		writeJSON(validateResult{Valid: true, Blocks: len(blocks)})
	case "explain":
		blocks, err := parseJSONPipeline()
		if err != nil {
			exitJSONError(err.Error())
		}
		cmd := pipeline.GenerateCommand(blocks, displayName)
		explanations := pipeline.ExplainPipeline(blocks, fileData)
		steps := make([]explainStep, len(explanations))
		for i, e := range explanations {
			steps[i] = explainStep{
				Step:       e.StepNum,
				Command:    e.Command,
				Summary:    e.Summary,
				InputRows:  e.InputRows,
				OutputRows: e.OutputRows,
			}
		}
		writeJSON(explainResult{Command: cmd, Steps: steps})
	}
}

func resolveBlocks(spec string, jsonMode bool) ([]pipeline.Block, error) {
	if jsonMode {
		return parseJSONPipeline()
	}
	if spec == "" {
		return nil, fmt.Errorf("no pipeline specified")
	}
	return parseSpec(spec)
}

func loadFileData(fileName string, jsonMode bool) (fileData string, displayName string) {
	if fileName == "" {
		return data.SampleCSV, "sample_sales.csv"
	}

	raw, err := os.ReadFile(fileName)
	if err != nil {
		if jsonMode {
			exitJSONError(fmt.Sprintf("reading %s: %v", fileName, err))
		}
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", fileName, err)
		os.Exit(1)
	}

	content := strings.TrimRight(string(raw), "\n\r")

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

Agent modes:
  pipeforge --schema            Dump command registry as JSON
  pipeforge --data              Show input data (columns, rows)
  pipeforge --validate          Validate pipeline JSON from stdin
  pipeforge --explain           Explain pipeline steps with row counts

Options:
  --file, -f FILE       Input file (default: built-in sample CSV)
  --json                JSON mode: read pipeline from stdin, output JSON
  --help, -h            Show this help

SPEC format:
  "command:key=value,flag | command:key=value"

  Commands: grep, awk, cut, sed, tr, sort, uniq, wc, head, tail, tee, xargs, datamash

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
  pipeforge --build "grep:pattern=get,ignoreCase | sort | uniq:count"

JSON mode (for agents & scripts):
  pipeforge --schema
  echo '[...]' | pipeforge --validate
  echo '[...]' | pipeforge --explain -f data.csv
  echo '[...]' | pipeforge --build --json
  echo '[...]' | pipeforge --preview --json -f data.csv
  cat pipeline.json | pipeforge --run --json -f data.csv`)
}
