package pipeline

import (
	"reflect"
	"testing"
)

func TestTokenizeAwkExpr(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:   "simple field refs",
			input:  "$2 $3",
			expect: []string{"$2", "$3"},
		},
		{
			name:   "field refs with quoted separator",
			input:  `$2 "-" $3 "," $5`,
			expect: []string{"$2", `"-"`, "$3", `","`, "$5"},
		},
		{
			name:   "single field",
			input:  "$0",
			expect: []string{"$0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenizeAwkExpr(tt.input)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestEvalAwkAction(t *testing.T) {
	fields := []string{"2026-01-05", "Acme Corp", "South", "Widget", "500"}
	tests := []struct {
		name   string
		action string
		expect string
	}{
		{
			name:   "print all",
			action: "print $0",
			expect: "2026-01-05,Acme Corp,South,Widget,500",
		},
		{
			name:   "concatenate with separator",
			action: `print $2 "-" $3 "," $5`,
			expect: "Acme Corp-South,500",
		},
		{
			name:   "single field",
			action: "print $3",
			expect: "South",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalAwkAction(tt.action, fields, ",")
			if got != tt.expect {
				t.Errorf("got %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestGroupSkipsHeader(t *testing.T) {
	lines := []string{
		"date,region,amount",
		"2026-01-05,North,500",
		"2026-01-06,North,300",
		"2026-01-07,South,200",
	}
	block := Block{Type: "group", Config: map[string]any{
		"keyCol": "2", "valCol": "3", "agg": "sum", "delimiter": ",",
	}}
	result := SimulateStep(lines, block)

	for _, line := range result {
		if line == "region,0" || line == "date,0" {
			t.Errorf("header row leaked into output: %q", line)
		}
	}
}

func TestGroupNullHandling(t *testing.T) {
	lines := []string{
		"North,500",
		"North,",
		"North,300",
	}
	block := Block{Type: "group", Config: map[string]any{
		"keyCol": "1", "valCol": "2", "agg": "sum", "delimiter": ",",
	}}
	result := SimulateStep(lines, block)

	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0] != "North,800" {
		t.Errorf("got %q, want %q (empty value should be skipped, not 0)", result[0], "North,800")
	}
}

func TestGroupAvgSkipsNulls(t *testing.T) {
	lines := []string{
		"North,500",
		"North,",
		"North,300",
	}
	block := Block{Type: "group", Config: map[string]any{
		"keyCol": "1", "valCol": "2", "agg": "avg", "delimiter": ",",
	}}
	result := SimulateStep(lines, block)

	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	// avg of 500 and 300 = 400, not 266.67 (which would include null as 0)
	if result[0] != "North,400.00" {
		t.Errorf("got %q, want %q", result[0], "North,400.00")
	}
}

func TestTableExtraction(t *testing.T) {
	lines := []string{
		"Sales Q1,,,",
		"date,rep,region,amount",
		"2026-01-05,Alice,North,500",
		"2026-01-06,Bob,South,350",
		",,,",
		"Expenses Q1,,,",
		"date,category,dept,amount",
		"2026-01-05,Travel,Sales,1200",
	}

	tests := []struct {
		name  string
		index int
		want  int // expected row count
		first string
	}{
		{"table 1", 1, 3, "date,rep,region,amount"},
		{"table 2", 2, 2, "date,category,dept,amount"},
		{"out of range", 3, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := Block{Type: "table", Config: map[string]any{
				"index": tt.index, "delimiter": ",",
			}}
			result := SimulateStep(lines, block)
			if len(result) != tt.want {
				t.Errorf("got %d rows, want %d", len(result), tt.want)
			}
			if tt.want > 0 && result[0] != tt.first {
				t.Errorf("first row = %q, want %q", result[0], tt.first)
			}
		})
	}
}

func TestTableSingleColumn(t *testing.T) {
	lines := []string{
		"name",
		"Alice",
		"Bob",
		",,,",
		"date,amount",
		"2026-01-05,500",
	}
	block := Block{Type: "table", Config: map[string]any{
		"index": 1, "delimiter": ",",
	}}
	result := SimulateStep(lines, block)

	if len(result) != 3 {
		t.Fatalf("got %d rows, want 3 (header + 2 data)", len(result))
	}
	if result[0] != "name" {
		t.Errorf("first row = %q, want %q", result[0], "name")
	}
}

func TestSkipHeaderRow(t *testing.T) {
	tests := []struct {
		name   string
		lines  []string
		expect int // expected length after skip
	}{
		{"with header", []string{"date,name,amount", "2026-01-05,Alice,500"}, 1},
		{"no header", []string{"2026-01-05,Alice,500", "2026-01-06,Bob,300"}, 2},
		{"empty", []string{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipHeaderRow(tt.lines)
			if len(got) != tt.expect {
				t.Errorf("got %d lines, want %d", len(got), tt.expect)
			}
		})
	}
}
