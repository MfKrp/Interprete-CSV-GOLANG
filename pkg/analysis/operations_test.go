package analysis

import (
	"strings"
	"testing"
)

var testColumns = []Column{{Name: "region", Type: "string"}, {Name: "sales", Type: "number"}, {Name: "units", Type: "number"}}
var testRows = [][]string{{"Norte", "10", "1"}, {"Sur", "20", "2"}, {"Norte", "30", "3"}, {"Centro", "", "4"}}

func TestCountRows(t *testing.T) {
	if got := CountRows(testRows); got != 4 {
		t.Fatalf("CountRows() = %d, want 4", got)
	}
	if got := CountRows(nil); got != 0 {
		t.Fatalf("CountRows(empty) = %d, want 0", got)
	}
}

func TestCountDistinct(t *testing.T) {
	columns := []Column{{Name: "seller", Type: "string"}, {Name: "amount", Type: "number"}}
	tests := []struct {
		name    string
		rows    [][]string
		column  string
		want    int
		wantErr bool
	}{
		{"duplicates normalized", [][]string{{" Ana ", "1"}, {"ana", "2"}, {"BOB", "3"}}, "seller", 2, false},
		{"all unique", [][]string{{"a", "1"}, {"b", "2"}}, "seller", 2, false},
		{"ignore blank", [][]string{{"", "1"}, {"  ", "2"}, {"a", "3"}}, "seller", 1, false},
		{"numeric column", [][]string{{"a", "1"}, {"b", "2"}, {"c", "1"}}, "amount", 2, false},
		{"all blank", [][]string{{" ", ""}, {"", ""}}, "seller", 0, false},
		{"empty dataset", nil, "seller", 0, false},
		{"missing column", nil, "absent", 0, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CountDistinct(columns, test.rows, test.column)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestCountFiltered(t *testing.T) {
	columns := []Column{{Name: "region", Type: "string"}, {Name: "seller", Type: "string"}, {Name: "sales", Type: "number"}}
	rows := [][]string{{" Norte ", "Ana", "10000"}, {"sur", "Bob", "10001"}, {"NORTE", "Ana", "20000"}, {"Centro", "Cara", ""}}
	tests := []struct {
		name, column, operator string
		value                  any
		rows                   [][]string
		want                   int
		wantErr                bool
	}{
		{"text equal case insensitive", "region", "eq", "norte", rows, 2, false},
		{"text not equal", "region", "neq", "Norte", rows, 2, false},
		{"contains case insensitive", "seller", "contains", "AN", rows, 2, false},
		{"numeric greater", "sales", "gt", 10000.0, rows, 2, false},
		{"numeric greater equal", "sales", "gte", 10000.0, rows, 3, false},
		{"numeric less", "sales", "lt", 10001.0, rows, 1, false},
		{"numeric less equal", "sales", "lte", 10000.0, rows, 1, false},
		{"numeric equal", "sales", "eq", 10000.0, rows, 1, false},
		{"no match", "region", "eq", "Oeste", rows, 0, false},
		{"empty rows", "region", "eq", "Norte", nil, 0, false},
		{"missing column", "absent", "eq", "x", rows, 0, true},
		{"invalid operator", "region", "regex", "Norte", rows, 0, true},
		{"contains numeric", "sales", "contains", "1", rows, 0, true},
		{"numeric compare text", "region", "gt", 1.0, rows, 0, true},
		{"text value for number", "sales", "gt", "10000", rows, 0, true},
		{"number value for text", "region", "eq", 1.0, rows, 0, true},
		{"invalid numeric cell", "sales", "eq", 1.0, [][]string{{"Norte", "Ana", "bad"}}, 0, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CountFiltered(columns, test.rows, test.column, test.operator, test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestNumericOperations(t *testing.T) {
	tests := []struct {
		name      string
		operation func([]Column, [][]string, string) (float64, error)
		want      float64
	}{
		{"sum", Sum, 60}, {"average", Average, 20}, {"min", Min, 10}, {"max", Max, 30},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.operation(testColumns, testRows, "sales")
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestNumericOperationsValidation(t *testing.T) {
	operations := []func([]Column, [][]string, string) (float64, error){Sum, Average, Min, Max}
	for _, operation := range operations {
		if _, err := operation(testColumns, testRows, "missing"); err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("missing column error = %v", err)
		}
		if _, err := operation(testColumns, testRows, "region"); err == nil || !strings.Contains(err.Error(), "not numeric") {
			t.Errorf("string column error = %v", err)
		}
	}
}

func TestNumericOperationsEmptyAndInvalidValues(t *testing.T) {
	columns := []Column{{Name: "n", Type: "number"}}
	if got, err := Sum(columns, [][]string{{""}}, "n"); err != nil || got != 0 {
		t.Fatalf("sum of empty values = %v, %v", got, err)
	}
	for _, operation := range []func([]Column, [][]string, string) (float64, error){Average, Min, Max} {
		if _, err := operation(columns, nil, "n"); err == nil {
			t.Error("expected no-values error for empty dataset")
		}
		if _, err := operation(columns, [][]string{{""}}, "n"); err == nil {
			t.Error("expected no-values error for blank numeric values")
		}
	}
	if _, err := Sum(columns, [][]string{{"oops"}}, "n"); err == nil || !strings.Contains(err.Error(), "invalid numeric value") {
		t.Fatalf("invalid value error = %v", err)
	}
	if _, err := Sum(columns, [][]string{{"NaN"}}, "n"); err == nil {
		t.Fatal("NaN should be rejected")
	}
}

func TestGroupByCountAndSum(t *testing.T) {
	counts, err := GroupBy(testColumns, testRows, "region", "count", "")
	if err != nil {
		t.Fatal(err)
	}
	wantCounts := []GroupResult{{Key: "Centro", Count: 1}, {Key: "Norte", Count: 2}, {Key: "Sur", Count: 1}}
	if !equalGroups(counts, wantCounts) {
		t.Fatalf("count groups = %#v, want %#v", counts, wantCounts)
	}

	sums, err := GroupBy(testColumns, testRows, "region", "sum", "sales")
	if err != nil {
		t.Fatal(err)
	}
	wantSums := []GroupResult{{Key: "Centro"}, {Key: "Norte", Sum: 40}, {Key: "Sur", Sum: 20}}
	if !equalGroups(sums, wantSums) {
		t.Fatalf("sum groups = %#v, want %#v", sums, wantSums)
	}
}

func TestGroupByValidationAndEmptyRows(t *testing.T) {
	if got, err := GroupBy(testColumns, nil, "region", "count", ""); err != nil || len(got) != 0 {
		t.Fatalf("empty group count = %#v, %v", got, err)
	}
	for _, tc := range []struct{ group, agg, sum string }{{"missing", "count", ""}, {"region", "avg", ""}, {"region", "sum", "region"}} {
		if _, err := GroupBy(testColumns, testRows, tc.group, tc.agg, tc.sum); err == nil {
			t.Errorf("GroupBy(%q,%q,%q) error = nil", tc.group, tc.agg, tc.sum)
		}
	}
	if _, err := GroupBy(testColumns, [][]string{{"Norte", "bad", "1"}}, "region", "sum", "sales"); err == nil {
		t.Fatal("expected invalid sum error")
	}
}

func TestMalformedRows(t *testing.T) {
	columns := []Column{{Name: "n", Type: "number"}}
	if _, err := Sum(columns, [][]string{{}}, "n"); err == nil {
		t.Fatal("expected short row error")
	}
}

func equalGroups(left, right []GroupResult) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
