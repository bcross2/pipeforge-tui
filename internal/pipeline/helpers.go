package pipeline

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

// CSVSplit parses a single CSV line respecting quoted fields.
// Falls back to strings.Split if the delimiter is not a comma
// or if csv parsing fails.
func CSVSplit(line string, delim string) []string {
	if delim != "," {
		return strings.Split(line, delim)
	}
	r := csv.NewReader(strings.NewReader(line))
	r.LazyQuotes = true
	fields, err := r.Read()
	if err != nil {
		return strings.Split(line, delim)
	}
	return fields
}

// CSVJoin produces a CSV line, quoting fields that contain the delimiter.
func CSVJoin(fields []string, delim string) string {
	if delim != "," {
		return strings.Join(fields, delim)
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	w.Write(fields)
	w.Flush()
	return strings.TrimRight(b.String(), "\n")
}

func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func BuildSedExpression(pattern, replacement string, global bool) string {
	delimiters := []byte{'/', '|', '#', '~', '%', '@', ':'}
	delim := byte('/')
	for _, ch := range delimiters {
		if !strings.ContainsRune(pattern, rune(ch)) && !strings.ContainsRune(replacement, rune(ch)) {
			delim = ch
			break
		}
	}
	d := string(delim)
	safePat := strings.ReplaceAll(pattern, d, `\`+d)
	safeRep := strings.ReplaceAll(replacement, "&", `\&`)
	safeRep = strings.ReplaceAll(safeRep, d, `\`+d)
	g := ""
	if global {
		g = "g"
	}
	return fmt.Sprintf("s%s%s%s%s%s%s", d, safePat, d, safeRep, d, g)
}

func ParseFields(fieldStr string) []int {
	var result []int
	for _, part := range strings.Split(fieldStr, ",") {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, "-"); idx > 0 {
			start, err1 := strconv.Atoi(part[:idx])
			end, err2 := strconv.Atoi(part[idx+1:])
			if err1 == nil && err2 == nil {
				for i := start; i <= end; i++ {
					result = append(result, i)
				}
			}
		} else {
			n, err := strconv.Atoi(part)
			if err == nil {
				result = append(result, n)
			}
		}
	}
	return result
}

func ExpandRange(str string) []byte {
	if len(str) >= 3 && str[1] == '-' {
		start := str[0]
		end := str[2]
		var chars []byte
		for i := start; i <= end; i++ {
			chars = append(chars, i)
		}
		return chars
	}
	return []byte(str)
}

func GetConfigPreview(b Block) string {
	switch b.Type {
	case "grep":
		pat := b.GetString("pattern")
		if pat == "" {
			return ""
		}
		s := `"` + pat + `"`
		if b.GetBool("ignoreCase") {
			s += " -i"
		}
		if b.GetBool("invert") {
			s += " -v"
		}
		return s
	case "awk":
		cond := b.GetString("condition")
		act := b.GetString("action")
		if cond == "" && act == "print $0" {
			return ""
		}
		return fmt.Sprintf("%s {%s}", cond, act)
	case "group":
		keyCol := b.GetString("keyCol")
		agg := b.GetString("agg")
		valCol := b.GetString("valCol")
		if keyCol == "" {
			return ""
		}
		return fmt.Sprintf("col %s [%s] col %s", keyCol, agg, valCol)
	case "cut":
		f := b.GetString("fields")
		if f == "" {
			return ""
		}
		return "-f" + f
	case "sort":
		k := b.GetString("key")
		s := ""
		if k != "" {
			s = "-k" + k
		}
		if b.GetBool("numeric") {
			s += " -n"
		}
		if b.GetBool("reverse") {
			s += " -r"
		}
		return strings.TrimSpace(s)
	case "uniq":
		if b.GetBool("count") {
			return "-c"
		}
		return ""
	case "head", "tail":
		n := b.GetInt("lines")
		if n > 0 {
			return fmt.Sprintf("-n %d", n)
		}
		return ""
	case "sed":
		pat := b.GetString("pattern")
		if pat == "" {
			return ""
		}
		return fmt.Sprintf("s/%s/%s/", pat, b.GetString("replacement"))
	case "tr":
		from := b.GetString("from")
		if from == "" {
			return ""
		}
		return fmt.Sprintf("'%s' '%s'", from, b.GetString("to"))
	case "tee":
		return b.GetString("filename")
	case "xargs":
		cmd := b.GetString("command")
		if cmd == "" {
			return ""
		}
		s := ""
		if b.GetBool("placeholder") {
			rs := b.GetString("replaceStr")
			if rs == "" {
				rs = "{}"
			}
			s = "-I" + rs + " "
		}
		return s + cmd
	case "join":
		f := b.GetString("file")
		if f == "" {
			return ""
		}
		return fmt.Sprintf("%s on %s=%s", b.GetString("mode"), b.GetString("leftCol"), b.GetString("rightCol"))
	case "table":
		n := b.GetInt("index")
		if n > 0 {
			return fmt.Sprintf("table %d", n)
		}
		return ""
	case "wc":
		var flags []string
		if b.GetBool("lines") {
			flags = append(flags, "-l")
		}
		if b.GetBool("words") {
			flags = append(flags, "-w")
		}
		if b.GetBool("chars") {
			flags = append(flags, "-c")
		}
		if len(flags) == 0 {
			return "-l"
		}
		return strings.Join(flags, " ")
	}
	return ""
}
