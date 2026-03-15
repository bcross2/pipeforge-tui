package pipeline

import (
	"fmt"
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

func evalAwkAction(action string, fields []string, delim string) string {
	re := regexp.MustCompile(`print\s+(.*)`)
	m := re.FindStringSubmatch(action)
	if m == nil {
		return strings.Join(fields, delim)
	}
	parts := strings.Split(m[1], ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		fieldRe := regexp.MustCompile(`^\$(\d+)$`)
		if fm := fieldRe.FindStringSubmatch(p); fm != nil {
			idx, _ := strconv.Atoi(fm[1])
			if idx-1 < len(fields) {
				result = append(result, fields[idx-1])
			} else {
				result = append(result, "")
			}
		} else if len(p) >= 2 && p[0] == '"' && p[len(p)-1] == '"' {
			result = append(result, p[1:len(p)-1])
		} else {
			result = append(result, p)
		}
	}
	return strings.Join(result, " ")
}
