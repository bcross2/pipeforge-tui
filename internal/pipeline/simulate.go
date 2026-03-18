package pipeline

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func SimulateUpTo(blocks []Block, index int, sampleData string) []string {
	lines := strings.Split(sampleData, "\n")
	for i := 0; i <= index && i < len(blocks); i++ {
		lines = SimulateStep(lines, blocks[i])
	}
	return lines
}

func SimulateStep(lines []string, block Block) []string {
	c := block.Config
	switch block.Type {
	case "grep":
		pat := getString(c, "pattern")
		if pat == "" {
			return lines
		}
		flags := ""
		if getBool(c, "ignoreCase") {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + pat)
		if err != nil {
			return lines
		}
		invert := getBool(c, "invert")
		var result []string
		for _, l := range lines {
			match := re.MatchString(l)
			if (match && !invert) || (!match && invert) {
				result = append(result, l)
			}
		}
		return result

	case "awk":
		cond := getString(c, "condition")
		act := getString(c, "action")
		if cond == "" && act == "print $0" {
			return lines
		}
		delim := getString(c, "delimiter")
		if delim == "" {
			delim = ","
		}
		var result []string
		for _, line := range lines {
			fields := strings.Split(line, delim)
			if cond != "" && !evalAwkCondition(cond, fields) {
				continue
			}
			if act != "" && act != "print $0" {
				result = append(result, evalAwkAction(act, fields, delim))
			} else {
				result = append(result, line)
			}
		}
		return result

	case "group":
		keyIdx := 2 // default: column 3 (0-indexed)
		if k := getString(c, "keyCol"); k != "" {
			if n, err := strconv.Atoi(k); err == nil {
				keyIdx = n - 1
			}
		}
		valIdx := 3 // default: column 4 (0-indexed)
		if v := getString(c, "valCol"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				valIdx = n - 1
			}
		}
		agg := getString(c, "agg")
		if agg == "" {
			agg = "sum"
		}
		delim := getString(c, "delimiter")
		if delim == "" {
			delim = ","
		}
		type groupAcc struct {
			sum   float64
			count int
		}
		groups := make(map[string]*groupAcc)
		var order []string
		for i, line := range lines {
			fields := strings.Split(line, delim)
			if keyIdx >= len(fields) {
				continue
			}
			// skip header row: if the first line's value column is not numeric, skip it
			if i == 0 && valIdx < len(fields) {
				if _, err := strconv.ParseFloat(strings.TrimSpace(fields[valIdx]), 64); err != nil {
					continue
				}
			}
			key := fields[keyIdx]
			if _, ok := groups[key]; !ok {
				groups[key] = &groupAcc{}
				order = append(order, key)
			}
			// treat empty or non-numeric values as null — skip in aggregation
			if valIdx < len(fields) {
				raw := strings.TrimSpace(fields[valIdx])
				if raw != "" {
					if val, err := strconv.ParseFloat(raw, 64); err == nil {
						groups[key].sum += val
						groups[key].count++
					}
				}
			}
		}
		var result []string
		for _, key := range order {
			g := groups[key]
			switch agg {
			case "sum":
				if g.sum == float64(int(g.sum)) {
					result = append(result, fmt.Sprintf("%s%s%.0f", key, delim, g.sum))
				} else {
					result = append(result, fmt.Sprintf("%s%s%.2f", key, delim, g.sum))
				}
			case "count":
				result = append(result, fmt.Sprintf("%s%s%d", key, delim, g.count))
			case "avg":
				avg := g.sum / float64(g.count)
				result = append(result, fmt.Sprintf("%s%s%.2f", key, delim, avg))
			default:
				result = append(result, fmt.Sprintf("%s%s%.0f", key, delim, g.sum))
			}
		}
		return result

	case "cut":
		f := getString(c, "fields")
		if f == "" {
			return lines
		}
		delim := getString(c, "delimiter")
		if delim == "" {
			delim = ","
		}
		fieldList := ParseFields(f)
		var result []string
		for _, line := range lines {
			parts := strings.Split(line, delim)
			var selected []string
			for _, fi := range fieldList {
				if fi-1 < len(parts) {
					selected = append(selected, parts[fi-1])
				} else {
					selected = append(selected, "")
				}
			}
			result = append(result, strings.Join(selected, delim))
		}
		return result

	case "sort":
		sorted := make([]string, len(lines))
		copy(sorted, lines)
		ki := 0
		if k := getString(c, "key"); k != "" {
			if n, err := strconv.Atoi(k); err == nil {
				ki = n - 1
			}
		}
		delim := getString(c, "delimiter")
		if delim == "" {
			delim = ","
		}
		numeric := getBool(c, "numeric")
		sort.SliceStable(sorted, func(i, j int) bool {
			pa := strings.Split(sorted[i], delim)
			pb := strings.Split(sorted[j], delim)
			va, vb := "", ""
			if ki < len(pa) {
				va = pa[ki]
			}
			if ki < len(pb) {
				vb = pb[ki]
			}
			if numeric {
				na, _ := strconv.ParseFloat(va, 64)
				nb, _ := strconv.ParseFloat(vb, 64)
				return na < nb
			}
			return va < vb
		})
		if getBool(c, "reverse") {
			for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
		return sorted

	case "uniq":
		count := getBool(c, "count")
		dupsOnly := getBool(c, "duplicatesOnly")
		var result []string
		prev := ""
		cnt := 0
		for _, line := range lines {
			if line == prev {
				cnt++
			} else {
				if cnt > 0 {
					if !dupsOnly || cnt > 1 {
						if count {
							result = append(result, fmt.Sprintf("%7d %s", cnt, prev))
						} else {
							result = append(result, prev)
						}
					}
				}
				prev = line
				cnt = 1
			}
		}
		if cnt > 0 {
			if !dupsOnly || cnt > 1 {
				if count {
					result = append(result, fmt.Sprintf("%7d %s", cnt, prev))
				} else {
					result = append(result, prev)
				}
			}
		}
		return result

	case "join":
		rightFile := getString(c, "file")
		if rightFile == "" {
			return lines
		}
		raw, err := os.ReadFile(rightFile)
		if err != nil {
			return lines
		}
		rightLines := strings.Split(strings.TrimRight(string(raw), "\n\r"), "\n")

		// skip header rows before any processing
		lines = skipHeaderRow(lines)
		rightLines = skipHeaderRow(rightLines)

		// run sub-pipeline on right side if provided (after header removal)
		subBlocks := parseSubPipeline(c)
		for _, sb := range subBlocks {
			rightLines = SimulateStep(rightLines, sb)
		}

		leftCol := 0
		if k := getString(c, "leftCol"); k != "" {
			if n, err := strconv.Atoi(k); err == nil {
				leftCol = n - 1
			}
		}
		rightCol := 0
		if k := getString(c, "rightCol"); k != "" {
			if n, err := strconv.Atoi(k); err == nil {
				rightCol = n - 1
			}
		}
		mode := getString(c, "mode")
		if mode == "" {
			mode = "inner"
		}
		delim := getString(c, "delimiter")
		if delim == "" {
			delim = ","
		}

		// build right-side lookup: joinKey → []fields
		type rightEntry struct {
			fields []string
		}
		rightMap := make(map[string][]rightEntry)
		var rightKeys []string
		for _, line := range rightLines {
			fields := strings.Split(line, delim)
			if rightCol >= len(fields) {
				continue
			}
			key := strings.TrimSpace(fields[rightCol])
			if _, exists := rightMap[key]; !exists {
				rightKeys = append(rightKeys, key)
			}
			// collect fields excluding the join column
			var kept []string
			for j, f := range fields {
				if j != rightCol {
					kept = append(kept, f)
				}
			}
			rightMap[key] = append(rightMap[key], rightEntry{fields: kept})
		}

		// join
		rightMatched := make(map[string]bool)
		var joinResult []string
		for _, line := range lines {
			fields := strings.Split(line, delim)
			if leftCol >= len(fields) {
				continue
			}
			key := strings.TrimSpace(fields[leftCol])
			if entries, ok := rightMap[key]; ok {
				rightMatched[key] = true
				for _, e := range entries {
					combined := append(fields, e.fields...)
					joinResult = append(joinResult, strings.Join(combined, delim))
				}
			} else if mode == "left" || mode == "full" {
				// pad with empty fields for unmatched left row
				padCount := 0
				if len(rightMap) > 0 {
					for _, entries := range rightMap {
						padCount = len(entries[0].fields)
						break
					}
				}
				padded := make([]string, padCount)
				combined := append(fields, padded...)
				joinResult = append(joinResult, strings.Join(combined, delim))
			}
		}

		// add unmatched right rows for right/full mode
		if mode == "right" || mode == "full" {
			leftWidth := 0
			if len(lines) > 0 {
				leftWidth = len(strings.Split(lines[0], delim))
			}
			for _, key := range rightKeys {
				if !rightMatched[key] {
					for _, e := range rightMap[key] {
						padded := make([]string, leftWidth)
						// put the join key in the left join column
						if leftCol < leftWidth {
							padded[leftCol] = key
						}
						combined := append(padded, e.fields...)
						joinResult = append(joinResult, strings.Join(combined, delim))
					}
				}
			}
		}
		return joinResult

	case "comm":
		rightFile := getString(c, "file")
		if rightFile == "" {
			return lines
		}
		raw, err := os.ReadFile(rightFile)
		if err != nil {
			return lines
		}
		rightLines := strings.Split(strings.TrimRight(string(raw), "\n\r"), "\n")

		leftLines := make([]string, len(lines))
		copy(leftLines, lines)
		if getBool(c, "autoSort") {
			sort.Strings(leftLines)
			sort.Strings(rightLines)
		}

		mode := getString(c, "mode")
		if mode == "" {
			mode = "common"
		}

		leftSet := make(map[string]bool)
		for _, l := range leftLines {
			leftSet[l] = true
		}
		rightSet := make(map[string]bool)
		for _, l := range rightLines {
			rightSet[l] = true
		}

		var commResult []string
		switch mode {
		case "common":
			for _, l := range leftLines {
				if rightSet[l] {
					commResult = append(commResult, l)
				}
			}
		case "left-only":
			for _, l := range leftLines {
				if !rightSet[l] {
					commResult = append(commResult, l)
				}
			}
		case "right-only":
			for _, l := range rightLines {
				if !leftSet[l] {
					commResult = append(commResult, l)
				}
			}
		default: // "all"
			i, j := 0, 0
			for i < len(leftLines) && j < len(rightLines) {
				if leftLines[i] < rightLines[j] {
					commResult = append(commResult, leftLines[i])
					i++
				} else if leftLines[i] > rightLines[j] {
					commResult = append(commResult, "\t"+rightLines[j])
					j++
				} else {
					commResult = append(commResult, "\t\t"+leftLines[i])
					i++
					j++
				}
			}
			for ; i < len(leftLines); i++ {
				commResult = append(commResult, leftLines[i])
			}
			for ; j < len(rightLines); j++ {
				commResult = append(commResult, "\t"+rightLines[j])
			}
		}
		return commResult

	case "table":
		target := getInt(c, "index", 1)
		delim := getString(c, "delimiter")
		if delim == "" {
			delim = ","
		}
		tableNum := 0
		inTable := false
		atBoundary := true // start of input counts as a boundary
		hasPending := false
		pendingLine := ""
		var result []string
		for _, line := range lines {
			// check if line is blank (all fields empty)
			fields := strings.Split(line, delim)
			blank := true
			for _, f := range fields {
				if strings.TrimSpace(f) != "" {
					blank = false
					break
				}
			}
			if blank {
				inTable = false
				atBoundary = true
				hasPending = false
				continue
			}
			nonEmpty := 0
			for _, f := range fields {
				if strings.TrimSpace(f) != "" {
					nonEmpty++
				}
			}
			if hasPending {
				if nonEmpty > 1 {
					// pending was a title, this row is a header — new table
					tableNum++
					inTable = true
					hasPending = false
					atBoundary = false
				} else {
					// pending was a header for a single-column table
					tableNum++
					inTable = true
					hasPending = false
					atBoundary = false
					if tableNum == target {
						result = append(result, pendingLine)
					}
				}
			} else if atBoundary {
				if nonEmpty <= 1 {
					// might be a title — buffer it and wait for next row
					pendingLine = line
					hasPending = true
					atBoundary = false
					continue
				}
				// multi-cell header — new table starts
				tableNum++
				inTable = true
				atBoundary = false
			}
			if inTable && tableNum == target {
				result = append(result, line)
			}
		}
		return result

	case "head":
		n := getInt(c, "lines", 5)
		if n >= len(lines) {
			return lines
		}
		return lines[:n]

	case "tail":
		n := getInt(c, "lines", 5)
		if n >= len(lines) {
			return lines
		}
		return lines[len(lines)-n:]

	case "sed":
		pat := getString(c, "pattern")
		if pat == "" {
			return lines
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return lines
		}
		rep := getString(c, "replacement")
		global := getBool(c, "global")
		var result []string
		for _, l := range lines {
			if global {
				result = append(result, re.ReplaceAllString(l, rep))
			} else {
				loc := re.FindStringIndex(l)
				if loc != nil {
					result = append(result, l[:loc[0]]+rep+l[loc[1]:])
				} else {
					result = append(result, l)
				}
			}
		}
		return result

	case "tr":
		from := getString(c, "from")
		if from == "" {
			return lines
		}
		del := getBool(c, "delete")
		squeeze := getBool(c, "squeeze")
		fromChars := ExpandRange(from)
		toChars := ExpandRange(getString(c, "to"))
		var result []string
		for _, l := range lines {
			var buf []byte
			for _, ch := range []byte(l) {
				if del {
					found := false
					for _, fc := range fromChars {
						if ch == fc {
							found = true
							break
						}
					}
					if !found {
						buf = append(buf, ch)
					}
				} else {
					replaced := ch
					for idx, fc := range fromChars {
						if ch == fc {
							ti := idx
							if ti >= len(toChars) {
								ti = len(toChars) - 1
							}
							if ti >= 0 {
								replaced = toChars[ti]
							}
							break
						}
					}
					buf = append(buf, replaced)
				}
			}
			s := string(buf)
			if squeeze {
				var squeezed []byte
				for i := 0; i < len(s); i++ {
					if i == 0 || s[i] != s[i-1] {
						squeezed = append(squeezed, s[i])
					}
				}
				s = string(squeezed)
			}
			result = append(result, s)
		}
		return result

	case "tee":
		return lines

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
			var result []string
			for _, line := range lines {
				if strings.Contains(target, rs) {
					result = append(result, strings.ReplaceAll(target, rs, line))
				} else {
					result = append(result, target+" "+line)
				}
			}
			return result
		}
		ma := getString(c, "maxArgs")
		n := len(lines)
		if ma != "" {
			if parsed, err := strconv.Atoi(ma); err == nil && parsed > 0 {
				n = parsed
			}
		}
		var result []string
		for i := 0; i < len(lines); i += n {
			end := i + n
			if end > len(lines) {
				end = len(lines)
			}
			result = append(result, target+" "+strings.Join(lines[i:end], " "))
		}
		return result

	case "datamash":
		delim := getString(c, "delimiter")
		if delim == "" {
			delim = ","
		}
		groupByStr := getString(c, "groupBy")
		op := getString(c, "op")
		if op == "" {
			op = "sum"
		}
		colStr := getString(c, "col")
		colIdx := 0
		if colStr != "" {
			if n, err := strconv.Atoi(colStr); err == nil {
				colIdx = n - 1
			}
		}

		// Parse group-by columns (e.g. "1,3")
		var groupCols []int
		if groupByStr != "" {
			for _, part := range strings.Split(groupByStr, ",") {
				if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
					groupCols = append(groupCols, n-1)
				}
			}
		}

		dataLines := lines
		if getBool(c, "headerIn") && len(dataLines) > 0 {
			dataLines = dataLines[1:]
		}

		type groupAcc struct {
			values []float64
			strs   []string
		}
		groups := make(map[string]*groupAcc)
		var order []string

		for _, line := range dataLines {
			fields := strings.Split(line, delim)
			var keyParts []string
			for _, gi := range groupCols {
				if gi < len(fields) {
					keyParts = append(keyParts, strings.TrimSpace(fields[gi]))
				}
			}
			key := strings.Join(keyParts, "\t")
			if _, ok := groups[key]; !ok {
				groups[key] = &groupAcc{}
				order = append(order, key)
			}
			g := groups[key]
			if colIdx < len(fields) {
				raw := strings.TrimSpace(fields[colIdx])
				g.strs = append(g.strs, raw)
				if val, err := strconv.ParseFloat(raw, 64); err == nil {
					g.values = append(g.values, val)
				}
			}
		}

		var result []string
		for _, key := range order {
			g := groups[key]
			var agg string
			switch op {
			case "sum":
				s := 0.0
				for _, v := range g.values {
					s += v
				}
				agg = simFormatNum(s)
			case "mean":
				if len(g.values) == 0 {
					agg = "0"
				} else {
					s := 0.0
					for _, v := range g.values {
						s += v
					}
					agg = fmt.Sprintf("%.2f", s/float64(len(g.values)))
				}
			case "median":
				if len(g.values) == 0 {
					agg = "0"
				} else {
					sorted := make([]float64, len(g.values))
					copy(sorted, g.values)
					sort.Float64s(sorted)
					mid := len(sorted) / 2
					if len(sorted)%2 == 0 {
						agg = fmt.Sprintf("%.2f", (sorted[mid-1]+sorted[mid])/2)
					} else {
						agg = simFormatNum(sorted[mid])
					}
				}
			case "min":
				if len(g.values) == 0 {
					agg = "0"
				} else {
					m := g.values[0]
					for _, v := range g.values[1:] {
						if v < m {
							m = v
						}
					}
					agg = simFormatNum(m)
				}
			case "max":
				if len(g.values) == 0 {
					agg = "0"
				} else {
					m := g.values[0]
					for _, v := range g.values[1:] {
						if v > m {
							m = v
						}
					}
					agg = simFormatNum(m)
				}
			case "count":
				agg = strconv.Itoa(len(g.strs))
			case "countunique":
				seen := make(map[string]bool)
				for _, s := range g.strs {
					seen[s] = true
				}
				agg = strconv.Itoa(len(seen))
			case "stdev":
				if len(g.values) == 0 {
					agg = "0"
				} else {
					mean := 0.0
					for _, v := range g.values {
						mean += v
					}
					mean /= float64(len(g.values))
					variance := 0.0
					for _, v := range g.values {
						d := v - mean
						variance += d * d
					}
					variance /= float64(len(g.values))
					agg = fmt.Sprintf("%.2f", math.Sqrt(variance))
				}
			case "mode":
				freq := make(map[string]int)
				for _, s := range g.strs {
					freq[s]++
				}
				maxFreq := 0
				modeVal := ""
				for s, f := range freq {
					if f > maxFreq {
						maxFreq = f
						modeVal = s
					}
				}
				agg = modeVal
			default:
				agg = "?"
			}
			if key != "" {
				result = append(result, strings.ReplaceAll(key, "\t", delim)+delim+agg)
			} else {
				result = append(result, agg)
			}
		}
		return result

	case "wc":
		var parts []string
		l := getBool(c, "lines")
		w := getBool(c, "words")
		ch := getBool(c, "chars")
		if !l && !w && !ch {
			l = true
		}
		if l {
			parts = append(parts, strconv.Itoa(len(lines)))
		}
		if w {
			count := 0
			for _, line := range lines {
				count += len(strings.Fields(line))
			}
			parts = append(parts, strconv.Itoa(count))
		}
		if ch {
			count := len(strings.Join(lines, "\n"))
			parts = append(parts, strconv.Itoa(count))
		}
		return []string{strings.Join(parts, "\t")}
	}
	return lines
}

func simFormatNum(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func evalAwkCondition(cond string, fields []string) bool {
	re := regexp.MustCompile(`\$(\d+)\s*(==|!=|>=|<=|>|<)\s*"?([^"]*)"?\s*$`)
	m := re.FindStringSubmatch(cond)
	if m == nil {
		return true
	}
	idx, _ := strconv.Atoi(m[1])
	val := ""
	if idx-1 < len(fields) {
		val = fields[idx-1]
	}
	op := m[2]
	cmp := m[3]
	return compareAwkValues(val, cmp, op)
}

func compareAwkValues(left, right, op string) bool {
	lNum, lErr := strconv.ParseFloat(strings.TrimSpace(left), 64)
	rNum, rErr := strconv.ParseFloat(strings.TrimSpace(right), 64)
	if lErr == nil && rErr == nil {
		switch op {
		case "==":
			return lNum == rNum
		case "!=":
			return lNum != rNum
		case ">":
			return lNum > rNum
		case "<":
			return lNum < rNum
		case ">=":
			return lNum >= rNum
		case "<=":
			return lNum <= rNum
		}
	}
	switch op {
	case "==":
		return left == right
	case "!=":
		return left != right
	case ">":
		return left > right
	case "<":
		return left < right
	case ">=":
		return left >= right
	case "<=":
		return left <= right
	}
	return true
}

func skipHeaderRow(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	fields := strings.Split(lines[0], ",")
	for _, f := range fields {
		if _, err := strconv.ParseFloat(strings.TrimSpace(f), 64); err == nil {
			return lines // at least one numeric field — not a header
		}
	}
	return lines[1:] // all non-numeric — skip it
}

func parseSubPipeline(config map[string]any) []Block {
	raw, ok := config["pipeline"]
	if !ok {
		return nil
	}
	// re-marshal and unmarshal to get proper []Block
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var blocks []Block
	if err := json.Unmarshal(data, &blocks); err != nil {
		return nil
	}
	return blocks
}

func tokenizeAwkExpr(expr string) []string {
	var tokens []string
	i := 0
	for i < len(expr) {
		// skip whitespace
		if expr[i] == ' ' || expr[i] == '\t' {
			i++
			continue
		}
		// quoted string: collect until closing quote
		if expr[i] == '"' {
			j := i + 1
			for j < len(expr) && expr[j] != '"' {
				j++
			}
			if j < len(expr) {
				j++ // include closing quote
			}
			tokens = append(tokens, expr[i:j])
			i = j
			continue
		}
		// unquoted token: collect until whitespace or quote
		j := i
		for j < len(expr) && expr[j] != ' ' && expr[j] != '\t' && expr[j] != '"' {
			j++
		}
		tokens = append(tokens, expr[i:j])
		i = j
	}
	return tokens
}

func evalAwkAction(action string, fields []string, delim string) string {
	re := regexp.MustCompile(`print\s+(.*)`)
	m := re.FindStringSubmatch(action)
	if m == nil {
		return strings.Join(fields, delim)
	}
	tokens := tokenizeAwkExpr(m[1])
	fieldRe := regexp.MustCompile(`^\$(\d+)$`)
	var result []string
	for _, tok := range tokens {
		if fm := fieldRe.FindStringSubmatch(tok); fm != nil {
			idx, _ := strconv.Atoi(fm[1])
			if idx == 0 {
				result = append(result, strings.Join(fields, delim))
			} else if idx-1 < len(fields) {
				result = append(result, fields[idx-1])
			} else {
				result = append(result, "")
			}
		} else if len(tok) >= 2 && tok[0] == '"' && tok[len(tok)-1] == '"' {
			result = append(result, tok[1:len(tok)-1])
		} else {
			result = append(result, tok)
		}
	}
	return strings.Join(result, "")
}
