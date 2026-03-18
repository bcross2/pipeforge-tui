package pipeline

import (
	"fmt"
	"strings"
)

func GenerateCommand(blocks []Block, inputFile string) string {
	if len(blocks) == 0 {
		return ""
	}
	var parts []string
	for i, block := range blocks {
		isFirst := i == 0
		c := block.Config
		file := ""
		if isFirst {
			file = inputFile
		}
		var cmd string
		switch block.Type {
		case "grep":
			var flags []string
			if getBool(c, "ignoreCase") {
				flags = append(flags, "-i")
			}
			if getBool(c, "invert") {
				flags = append(flags, "-v")
			}
			pat := getString(c, "pattern")
			if pat == "" {
				pat = "pattern"
			}
			pieces := []string{"grep"}
			pieces = append(pieces, flags...)
			pieces = append(pieces, ShellQuote(pat))
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

		case "awk":
			cond := getString(c, "condition")
			act := getString(c, "action")
			if act == "" {
				act = "print $0"
			}
			body := fmt.Sprintf("{%s}", act)
			if cond != "" {
				body = fmt.Sprintf("%s {%s}", cond, act)
			}
			pieces := []string{"awk"}
			delim := getString(c, "delimiter")
			if delim != "" {
				pieces = append(pieces, "-F"+ShellQuote(delim))
			}
			pieces = append(pieces, ShellQuote(body))
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

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
			var body string
			switch agg {
			case "count":
				body = fmt.Sprintf("{count[$%s]++} END{for(k in count) print k,count[k]}", keyCol)
			case "avg":
				body = fmt.Sprintf("{sum[$%s]+=$%s; count[$%s]++} END{for(k in sum) print k,sum[k]/count[k]}", keyCol, valCol, keyCol)
			default: // sum
				body = fmt.Sprintf("{sum[$%s]+=$%s} END{for(k in sum) print k,sum[k]}", keyCol, valCol)
			}
			delim := getString(c, "delimiter")
			pieces := []string{"awk"}
			if delim != "" {
				pieces = append(pieces, "-F"+ShellQuote(delim))
			}
			pieces = append(pieces, ShellQuote(body))
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

		case "cut":
			f := getString(c, "fields")
			if f == "" {
				f = "1"
			}
			pieces := []string{"cut"}
			delim := getString(c, "delimiter")
			if delim != "" {
				pieces = append(pieces, "-d"+ShellQuote(delim))
			}
			pieces = append(pieces, "-f"+f)
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

		case "sort":
			var flags []string
			delim := getString(c, "delimiter")
			if delim != "" {
				flags = append(flags, "-t"+ShellQuote(delim))
			}
			k := getString(c, "key")
			if k != "" {
				flags = append(flags, "-k"+k)
			}
			if getBool(c, "numeric") {
				flags = append(flags, "-n")
			}
			if getBool(c, "reverse") {
				flags = append(flags, "-r")
			}
			pieces := []string{"sort"}
			pieces = append(pieces, flags...)
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

		case "uniq":
			var flags []string
			if getBool(c, "count") {
				flags = append(flags, "-c")
			}
			if getBool(c, "duplicatesOnly") {
				flags = append(flags, "-d")
			}
			pieces := []string{"uniq"}
			pieces = append(pieces, flags...)
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

		case "head":
			n := getInt(c, "lines", 5)
			pieces := []string{"head", "-n", fmt.Sprintf("%d", n)}
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

		case "tail":
			n := getInt(c, "lines", 5)
			pieces := []string{"tail", "-n", fmt.Sprintf("%d", n)}
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

		case "sed":
			pat := getString(c, "pattern")
			if pat == "" {
				pat = "pattern"
			}
			rep := getString(c, "replacement")
			g := getBool(c, "global")
			pieces := []string{"sed", ShellQuote(BuildSedExpression(pat, rep, g))}
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

		case "tr":
			var flags []string
			if getBool(c, "delete") {
				flags = append(flags, "-d")
			}
			if getBool(c, "squeeze") {
				flags = append(flags, "-s")
			}
			from := getString(c, "from")
			if from == "" {
				from = "a-z"
			}
			pieces := []string{"tr"}
			pieces = append(pieces, flags...)
			pieces = append(pieces, ShellQuote(from))
			if !getBool(c, "delete") {
				to := getString(c, "to")
				if to == "" {
					to = "A-Z"
				}
				pieces = append(pieces, ShellQuote(to))
			}
			cmd = strings.Join(pieces, " ")
			if isFirst {
				cmd = "cat " + file + " | " + cmd
			}

		case "tee":
			fn := getString(c, "filename")
			if fn == "" {
				fn = "output.csv"
			}
			cmd = "tee " + ShellQuote(fn)
			if isFirst {
				cmd = "cat " + file + " | " + cmd
			}

		case "xargs":
			var flags []string
			if getBool(c, "placeholder") {
				rs := getString(c, "replaceStr")
				if rs == "" {
					rs = "{}"
				}
				flags = append(flags, "-I"+ShellQuote(rs))
			}
			ma := getString(c, "maxArgs")
			if ma != "" {
				flags = append(flags, "-n"+ma)
			}
			target := getString(c, "command")
			if target == "" {
				target = "echo"
			}
			pieces := []string{"xargs"}
			pieces = append(pieces, flags...)
			pieces = append(pieces, target)
			cmd = strings.Join(pieces, " ")
			if isFirst {
				cmd = "cat " + file + " | " + cmd
			}

		case "join":
			rightFile := getString(c, "file")
			if rightFile == "" {
				rightFile = "right.csv"
			}
			leftCol := getString(c, "leftCol")
			if leftCol == "" {
				leftCol = "1"
			}
			rightCol := getString(c, "rightCol")
			if rightCol == "" {
				rightCol = "1"
			}
			mode := getString(c, "mode")
			delim := getString(c, "delimiter")
			if delim == "" {
				delim = ","
			}

			var joinFlags []string
			joinFlags = append(joinFlags, "-t"+ShellQuote(delim))
			joinFlags = append(joinFlags, "-1", leftCol, "-2", rightCol)
			switch mode {
			case "left":
				joinFlags = append(joinFlags, "-a", "1")
			case "right":
				joinFlags = append(joinFlags, "-a", "2")
			case "full":
				joinFlags = append(joinFlags, "-a", "1", "-a", "2")
			}
			joinFlags = append(joinFlags, "-e", "''")
			joinFlags = append(joinFlags, "-o", "auto")

			// build right-side command
			subBlocks := parseSubPipeline(c)
			var rightCmd string
			if len(subBlocks) > 0 {
				rightCmd = GenerateCommand(subBlocks, rightFile) + " | sort -t" + ShellQuote(delim) + " -k" + rightCol
			} else {
				rightCmd = "sort -t" + ShellQuote(delim) + " -k" + rightCol + " " + rightFile
			}

			if isFirst {
				// both sides via process substitution
				leftCmd := "sort -t" + ShellQuote(delim) + " -k" + leftCol + " " + file
				cmd = "join " + strings.Join(joinFlags, " ") + " <(" + leftCmd + ") <(" + rightCmd + ")"
			} else {
				// left from stdin (previous pipe), right via process substitution
				cmd = "sort -t" + ShellQuote(delim) + " -k" + leftCol + " | join " + strings.Join(joinFlags, " ") + " - <(" + rightCmd + ")"
			}

		case "table":
			target := getInt(c, "index", 1)
			delim := getString(c, "delimiter")
			if delim == "" {
				delim = ","
			}
			body := fmt.Sprintf(`BEGIN{tbl=0;bd=1;pn=0} {blank=1;for(i=1;i<=NF;i++)if($i!="")blank=0; if(blank){in_t=0;bd=1;pn=0;next} ne=0;for(i=1;i<=NF;i++)if($i!="")ne++; if(pn){if(ne>1){tbl++;in_t=1;pn=0;bd=0}else{tbl++;in_t=1;pn=0;bd=0;if(tbl==%d)print pl}} else if(bd){if(ne<=1){pl=$0;pn=1;bd=0;next}tbl++;in_t=1;bd=0} if(in_t&&tbl==%d)print}`, target, target)
			pieces := []string{"awk", "-F" + ShellQuote(delim), ShellQuote(body)}
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")

		case "wc":
			var flags []string
			if getBool(c, "lines") {
				flags = append(flags, "-l")
			}
			if getBool(c, "words") {
				flags = append(flags, "-w")
			}
			if getBool(c, "chars") {
				flags = append(flags, "-c")
			}
			if len(flags) == 0 {
				flags = append(flags, "-l")
			}
			pieces := []string{"wc"}
			pieces = append(pieces, flags...)
			if file != "" {
				pieces = append(pieces, file)
			}
			cmd = strings.Join(pieces, " ")
		}
		parts = append(parts, cmd)
	}
	return strings.Join(parts, " | ")
}

func getString(c map[string]any, key string) string {
	if v, ok := c[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(c map[string]any, key string) bool {
	if v, ok := c[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getInt(c map[string]any, key string, fallback int) int {
	if v, ok := c[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return fallback
}
