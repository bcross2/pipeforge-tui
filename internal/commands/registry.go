package commands

var Groups = []Group{
	{ID: "filter", Label: "Filter"},
	{ID: "transform", Label: "Transform"},
	{ID: "aggregate", Label: "Aggregate"},
	{ID: "limit", Label: "Limit"},
	{ID: "output", Label: "Output"},
}

var Registry = map[string]CommandDef{
	"grep": {
		Label: "grep", Excel: "Ctrl+F / Filter", Group: "filter", Icon: "Gp",
		Defaults: map[string]any{"pattern": "", "ignoreCase": false, "invert": false},
		Config: []ConfigField{
			{Key: "pattern", Type: FieldText, Label: "Search Pattern", Placeholder: "e.g. North"},
			{Key: "ignoreCase", Type: FieldCheck, Label: "Case insensitive (-i)"},
			{Key: "invert", Type: FieldCheck, Label: "Invert match (-v)"},
		},
	},
	"awk": {
		Label: "awk", Excel: "Column filter + math", Group: "filter", Icon: "Ak",
		Defaults: map[string]any{"conditionPreset": "", "condition": "", "action": "print $0", "delimiter": ","},
		Config: []ConfigField{
			{
				Key: "conditionPreset", Type: FieldSelect, Label: "Common conditions",
				Hint: "CSV columns: $1=date, $2=product, $3=region, $4=amount",
				Options: []SelectOption{
					{Value: "", Label: "Custom condition"},
					{Value: `$3 == "North"`, Label: "Region is North"},
					{Value: `$3 != "North"`, Label: "Region is not North"},
					{Value: "$4 >= 200", Label: "Amount is at least 200"},
					{Value: "$4 < 200", Label: "Amount is below 200"},
					{Value: `$2 == "Widget"`, Label: "Product is Widget"},
					{Value: `$4 != ""`, Label: "Amount is not empty"},
				},
			},
			{Key: "condition", Type: FieldText, Label: "Condition", Placeholder: `e.g. $3 == "North"`},
			{Key: "action", Type: FieldText, Label: "Action", Placeholder: "e.g. print $2, $4"},
			{Key: "delimiter", Type: FieldText, Label: "Delimiter (-F)", Placeholder: ","},
		},
	},
	"cut": {
		Label: "cut", Excel: "Select columns", Group: "transform", Icon: "Ct",
		Defaults: map[string]any{"delimiter": ",", "fields": ""},
		Config: []ConfigField{
			{Key: "delimiter", Type: FieldText, Label: "Delimiter (-d)", Placeholder: ","},
			{Key: "fields", Type: FieldText, Label: "Fields (-f)", Placeholder: "e.g. 1,3 or 2-4"},
		},
	},
	"sed": {
		Label: "sed", Excel: "Find & Replace", Group: "transform", Icon: "Sd",
		Defaults: map[string]any{"pattern": "", "replacement": "", "global": true},
		Config: []ConfigField{
			{Key: "pattern", Type: FieldText, Label: "Find Pattern", Placeholder: "e.g. North"},
			{Key: "replacement", Type: FieldText, Label: "Replace With", Placeholder: "e.g. NORTH"},
			{Key: "global", Type: FieldCheck, Label: "Replace all occurrences (g)"},
		},
	},
	"tr": {
		Label: "tr", Excel: "SUBSTITUTE (chars)", Group: "transform", Icon: "Tr",
		Defaults: map[string]any{"from": "", "to": "", "squeeze": false, "delete": false},
		Config: []ConfigField{
			{Key: "from", Type: FieldText, Label: "From chars", Placeholder: "e.g. a-z"},
			{Key: "to", Type: FieldText, Label: "To chars", Placeholder: "e.g. A-Z"},
			{Key: "squeeze", Type: FieldCheck, Label: "Squeeze repeats (-s)"},
			{Key: "delete", Type: FieldCheck, Label: "Delete chars (-d)"},
		},
	},
	"group": {
		Label: "group", Excel: "Pivot / GROUP BY", Group: "aggregate", Icon: "Gr",
		Defaults: map[string]any{"keyCol": "3", "aggPreset": "", "agg": "sum", "valCol": "4", "delimiter": ","},
		Config: []ConfigField{
			{
				Key: "aggPreset", Type: FieldSelect, Label: "Quick setup",
				Hint: "CSV columns: $1=date, $2=product, $3=region, $4=amount",
				Options: []SelectOption{
					{Value: "", Label: "Custom"},
					{Value: "sum_by_region", Label: "Sum amount by region"},
					{Value: "sum_by_product", Label: "Sum amount by product"},
					{Value: "count_by_region", Label: "Count rows by region"},
					{Value: "count_by_product", Label: "Count rows by product"},
				},
			},
			{Key: "keyCol", Type: FieldText, Label: "Group by column", Placeholder: "e.g. 3 (region)"},
			{Key: "agg", Type: FieldText, Label: "Aggregation (sum/count/avg)", Placeholder: "sum"},
			{Key: "valCol", Type: FieldText, Label: "Value column to aggregate", Placeholder: "e.g. 4 (amount)"},
			{Key: "delimiter", Type: FieldText, Label: "Delimiter (-F)", Placeholder: ","},
		},
	},
	"sort": {
		Label: "sort", Excel: "Sort A-Z / Z-A", Group: "aggregate", Icon: "St",
		Defaults: map[string]any{"key": "", "numeric": false, "reverse": false, "delimiter": ","},
		Config: []ConfigField{
			{Key: "key", Type: FieldText, Label: "Sort key column (-k)", Placeholder: "e.g. 4"},
			{Key: "delimiter", Type: FieldText, Label: "Delimiter (-t)", Placeholder: ","},
			{Key: "numeric", Type: FieldCheck, Label: "Numeric sort (-n)"},
			{Key: "reverse", Type: FieldCheck, Label: "Reverse order (-r)"},
		},
	},
	"uniq": {
		Label: "uniq", Excel: "Remove Duplicates", Group: "aggregate", Icon: "Uq",
		Defaults: map[string]any{"count": false, "duplicatesOnly": false},
		Config: []ConfigField{
			{Key: "count", Type: FieldCheck, Label: "Show counts (-c)"},
			{Key: "duplicatesOnly", Type: FieldCheck, Label: "Only duplicates (-d)"},
		},
	},
	"wc": {
		Label: "wc", Excel: "COUNTA / Count", Group: "aggregate", Icon: "Wc",
		Defaults: map[string]any{"lines": true, "words": false, "chars": false},
		Config: []ConfigField{
			{Key: "lines", Type: FieldCheck, Label: "Count lines (-l)"},
			{Key: "words", Type: FieldCheck, Label: "Count words (-w)"},
			{Key: "chars", Type: FieldCheck, Label: "Count characters (-c)"},
		},
	},
	"join": {
		Label: "join", Excel: "VLOOKUP / merge", Group: "filter", Icon: "Jn",
		Defaults: map[string]any{"file": "", "leftCol": "1", "rightCol": "1", "mode": "inner", "delimiter": ","},
		Config: []ConfigField{
			{Key: "file", Type: FieldText, Label: "Right-side file", Placeholder: "e.g. vendors.csv"},
			{Key: "leftCol", Type: FieldText, Label: "Left join column", Placeholder: "e.g. 2"},
			{Key: "rightCol", Type: FieldText, Label: "Right join column", Placeholder: "e.g. 1"},
			{
				Key: "mode", Type: FieldSelect, Label: "Join mode",
				Options: []SelectOption{
					{Value: "inner", Label: "Inner — only matching rows"},
					{Value: "left", Label: "Left — all left rows, match or not"},
					{Value: "right", Label: "Right — all right rows, match or not"},
					{Value: "full", Label: "Full — all rows from both sides"},
				},
			},
			{Key: "delimiter", Type: FieldText, Label: "Delimiter", Placeholder: ","},
		},
	},
	"table": {
		Label: "table", Excel: "Extract table from sheet", Group: "filter", Icon: "Tb",
		Defaults: map[string]any{"index": 1, "delimiter": ","},
		Config: []ConfigField{
			{Key: "index", Type: FieldNumber, Label: "Table number", Placeholder: "e.g. 1, 2, 3"},
			{Key: "delimiter", Type: FieldText, Label: "Delimiter", Placeholder: ","},
		},
	},
	"head": {
		Label: "head", Excel: "Top N rows", Group: "limit", Icon: "Hd",
		Defaults: map[string]any{"lines": 5},
		Config: []ConfigField{
			{Key: "lines", Type: FieldNumber, Label: "Number of lines (-n)", Placeholder: "5"},
		},
	},
	"tail": {
		Label: "tail", Excel: "Bottom N rows", Group: "limit", Icon: "Tl",
		Defaults: map[string]any{"lines": 5},
		Config: []ConfigField{
			{Key: "lines", Type: FieldNumber, Label: "Number of lines (-n)", Placeholder: "5"},
		},
	},
	"tee": {
		Label: "tee", Excel: "Save + pass through", Group: "output", Icon: "Te",
		Defaults: map[string]any{"filename": "output.csv"},
		Config: []ConfigField{
			{Key: "filename", Type: FieldText, Label: "Save to file", Placeholder: "output.csv"},
		},
	},
	"xargs": {
		Label: "xargs", Excel: "Apply to each item", Group: "output", Icon: "Xa",
		Defaults: map[string]any{"command": "echo", "placeholder": false, "replaceStr": "{}", "maxArgs": ""},
		Config: []ConfigField{
			{Key: "command", Type: FieldText, Label: "Command to run", Placeholder: "e.g. echo, wc -l, rm"},
			{Key: "placeholder", Type: FieldCheck, Label: "Use replacement string (-I)"},
			{Key: "replaceStr", Type: FieldText, Label: "Replace string", Placeholder: "{}"},
			{Key: "maxArgs", Type: FieldText, Label: "Max args per call (-n)", Placeholder: "e.g. 1"},
		},
	},
}

// OrderedCommands returns command keys in display order (by group).
func OrderedCommands() []string {
	var result []string
	for _, g := range Groups {
		for _, key := range commandKeysForGroup(g.ID) {
			result = append(result, key)
		}
	}
	return result
}

func commandKeysForGroup(group string) []string {
	order := []string{"grep", "awk", "join", "table", "cut", "sed", "tr", "group", "sort", "uniq", "wc", "head", "tail", "tee", "xargs"}
	var result []string
	for _, k := range order {
		if def, ok := Registry[k]; ok && def.Group == group {
			result = append(result, k)
		}
	}
	return result
}
