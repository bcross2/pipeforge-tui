package pipeline

import (
	"fmt"
	"strings"
)

type StepExplanation struct {
	StepNum    int
	Command    string
	Summary    string
	InputRows  int
	OutputRows int
}

func ExplainPipeline(blocks []Block, sampleData string) []StepExplanation {
	lines := strings.Split(sampleData, "\n")
	var steps []StepExplanation
	for i, block := range blocks {
		inputRows := len(lines)
		lines = SimulateStep(lines, block)
		steps = append(steps, StepExplanation{
			StepNum:    i + 1,
			Command:    block.Type,
			Summary:    explainBlock(block),
			InputRows:  inputRows,
			OutputRows: len(lines),
		})
	}
	return steps
}

func explainBlock(block Block) string {
	c := block.Config
	switch block.Type {
	case "grep":
		pat := getString(c, "pattern")
		if pat == "" {
			return "Keep all rows (no pattern set)"
		}
		verb := "Keep"
		if getBool(c, "invert") {
			verb = "Remove"
		}
		s := fmt.Sprintf("%s rows containing '%s'", verb, pat)
		if getBool(c, "ignoreCase") {
			s += " (case insensitive)"
		}
		return s

	case "awk":
		cond := getString(c, "condition")
		act := getString(c, "action")
		if cond == "" && (act == "" || act == "print $0") {
			return "Pass all rows through"
		}
		if cond != "" && (act == "" || act == "print $0") {
			return fmt.Sprintf("Filter rows where %s", cond)
		}
		if cond == "" {
			return fmt.Sprintf("Transform each row: %s", act)
		}
		return fmt.Sprintf("Where %s, do %s", cond, act)

	case "group":
		keyCol := getString(c, "keyCol")
		if keyCol == "" {
			keyCol = "3"
		}
		valCol := getString(c, "valCol")
		if valCol == "" {
			valCol = "4"
		}
		agg := getString(c, "agg")
		if agg == "" {
			agg = "sum"
		}
		return fmt.Sprintf("Group by column %s, %s of column %s", keyCol, agg, valCol)

	case "cut":
		f := getString(c, "fields")
		if f == "" {
			return "Keep all columns (no fields set)"
		}
		s := fmt.Sprintf("Keep columns %s", f)
		delim := getString(c, "delimiter")
		if delim != "" && delim != "," {
			s += fmt.Sprintf(" (delimiter: '%s')", delim)
		}
		return s

	case "sort":
		s := "Sort rows"
		if k := getString(c, "key"); k != "" {
			s += fmt.Sprintf(" by column %s", k)
		}
		if getBool(c, "numeric") {
			s += ", numerically"
		}
		if getBool(c, "reverse") {
			s += ", highest first"
		}
		return s

	case "uniq":
		if getBool(c, "count") {
			return "Count consecutive duplicate rows"
		}
		if getBool(c, "duplicatesOnly") {
			return "Show only duplicated rows"
		}
		return "Remove consecutive duplicate rows"

	case "head":
		n := getInt(c, "lines", 5)
		return fmt.Sprintf("Keep the first %d rows", n)

	case "tail":
		n := getInt(c, "lines", 5)
		return fmt.Sprintf("Keep the last %d rows", n)

	case "sed":
		pat := getString(c, "pattern")
		if pat == "" {
			return "Find and replace (no pattern set)"
		}
		rep := getString(c, "replacement")
		s := fmt.Sprintf("Replace '%s' with '%s'", pat, rep)
		if getBool(c, "global") {
			s += " (all occurrences)"
		} else {
			s += " (first per line)"
		}
		return s

	case "tr":
		from := getString(c, "from")
		if from == "" {
			return "Translate characters (not configured)"
		}
		if getBool(c, "delete") {
			s := fmt.Sprintf("Delete characters '%s'", from)
			if getBool(c, "squeeze") {
				s += ", squeezing repeats"
			}
			return s
		}
		to := getString(c, "to")
		s := fmt.Sprintf("Translate '%s' to '%s'", from, to)
		if getBool(c, "squeeze") {
			s += ", squeezing repeats"
		}
		return s

	case "tee":
		fn := getString(c, "filename")
		if fn == "" {
			fn = "output.csv"
		}
		return fmt.Sprintf("Save a copy to '%s' and pass data through", fn)

	case "xargs":
		target := getString(c, "command")
		if target == "" {
			target = "echo"
		}
		if getBool(c, "placeholder") {
			rs := getString(c, "replaceStr")
			if rs == "" {
				rs = "{}"
			}
			return fmt.Sprintf("Run '%s', replacing '%s' with each row", target, rs)
		}
		return fmt.Sprintf("Run '%s' with each row as argument", target)

	case "wc":
		var parts []string
		if getBool(c, "lines") || (!getBool(c, "words") && !getBool(c, "chars")) {
			parts = append(parts, "lines")
		}
		if getBool(c, "words") {
			parts = append(parts, "words")
		}
		if getBool(c, "chars") {
			parts = append(parts, "characters")
		}
		return fmt.Sprintf("Count the number of %s", strings.Join(parts, ", "))
	}
	return "Process rows"
}
