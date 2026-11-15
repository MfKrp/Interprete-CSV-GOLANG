// Package analysis provides deterministic calculations over CSV records.
package analysis

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type Column struct {
	Name string
	Type string
}

type GroupResult struct {
	Key   string  `json:"key"`
	Count int     `json:"count,omitempty"`
	Sum   float64 `json:"sum,omitempty"`
}

// CountRows returns the number of data records, excluding the CSV header.
func CountRows(rows [][]string) int { return len(rows) }

// CountDistinct counts non-empty, trimmed values without case sensitivity.
func CountDistinct(columns []Column, rows [][]string, columnName string) (int, error) {
	index, err := columnIndex(columns, columnName)
	if err != nil {
		return 0, err
	}
	values := make(map[string]struct{})
	for rowIndex, row := range rows {
		if index >= len(row) {
			return 0, fmt.Errorf("row %d has no value for column %q", rowIndex+1, columnName)
		}
		value := strings.ToLower(strings.TrimSpace(row[index]))
		if value != "" {
			values[value] = struct{}{}
		}
	}
	return len(values), nil
}

// CountFiltered counts rows matching one comparison against a column.
func CountFiltered(columns []Column, rows [][]string, columnName, operator string, filterValue any) (int, error) {
	index, err := columnIndex(columns, columnName)
	if err != nil {
		return 0, err
	}
	column := columns[index]
	if filterValue == nil {
		return 0, fmt.Errorf("filter value is required")
	}
	textValue, isText := filterValue.(string)
	var numericTarget float64
	if column.Type == "number" {
		switch value := filterValue.(type) {
		case float64:
			numericTarget = value
		case float32:
			numericTarget = float64(value)
		case int:
			numericTarget = float64(value)
		case int64:
			numericTarget = float64(value)
		default:
			err = fmt.Errorf("filter value for numeric column %q must be a number", columnName)
		}
		if err != nil || math.IsNaN(numericTarget) || math.IsInf(numericTarget, 0) {
			return 0, fmt.Errorf("filter value for numeric column %q must be a valid number", columnName)
		}
	} else if !isText {
		return 0, fmt.Errorf("filter value for text column %q must be text", columnName)
	}
	if column.Type != "number" && column.Type != "string" {
		return 0, fmt.Errorf("unsupported column type %q", column.Type)
	}
	switch operator {
	case "eq", "neq":
	case "contains":
		if column.Type != "string" {
			return 0, fmt.Errorf("operator contains requires a text column")
		}
	case "gt", "gte", "lt", "lte":
		if column.Type != "number" {
			return 0, fmt.Errorf("operator %s requires a numeric column", operator)
		}
	default:
		return 0, fmt.Errorf("unsupported filter operator %q", operator)
	}
	textTarget := strings.ToLower(strings.TrimSpace(textValue))
	count := 0
	for rowIndex, row := range rows {
		if index >= len(row) {
			return 0, fmt.Errorf("row %d has no value for column %q", rowIndex+1, columnName)
		}
		cell := strings.TrimSpace(row[index])
		matched := false
		if column.Type == "number" {
			if cell == "" {
				continue
			}
			numericCell, parseErr := strconv.ParseFloat(cell, 64)
			if parseErr != nil || math.IsNaN(numericCell) || math.IsInf(numericCell, 0) {
				return 0, fmt.Errorf("invalid numeric value %q in column %q at row %d", cell, columnName, rowIndex+1)
			}
			switch operator {
			case "eq":
				matched = numericCell == numericTarget
			case "neq":
				matched = numericCell != numericTarget
			case "gt":
				matched = numericCell > numericTarget
			case "gte":
				matched = numericCell >= numericTarget
			case "lt":
				matched = numericCell < numericTarget
			case "lte":
				matched = numericCell <= numericTarget
			}
		} else {
			cell = strings.ToLower(cell)
			switch operator {
			case "eq":
				matched = cell == textTarget
			case "neq":
				matched = cell != textTarget
			case "contains":
				matched = strings.Contains(cell, textTarget)
			}
		}
		if matched {
			count++
		}
	}
	return count, nil
}

// Sum totals the non-empty values in a numeric column.
func Sum(columns []Column, rows [][]string, columnName string) (float64, error) {
	columnIndex, err := numericColumnIndex(columns, columnName)
	if err != nil {
		return 0, err
	}
	var total float64
	for rowIndex, row := range rows {
		value, present, err := numericValue(row, rowIndex, columnIndex, columnName)
		if err != nil {
			return 0, err
		}
		if present {
			total += value
		}
	}
	return total, nil
}

// Average returns the mean of the non-empty values in a numeric column.
func Average(columns []Column, rows [][]string, columnName string) (float64, error) {
	columnIndex, err := numericColumnIndex(columns, columnName)
	if err != nil {
		return 0, err
	}
	var total float64
	count := 0
	for rowIndex, row := range rows {
		value, present, err := numericValue(row, rowIndex, columnIndex, columnName)
		if err != nil {
			return 0, err
		}
		if present {
			total += value
			count++
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("cannot calculate average of column %q: no numeric values", columnName)
	}
	return total / float64(count), nil
}

// Min returns the smallest non-empty value in a numeric column.
func Min(columns []Column, rows [][]string, columnName string) (float64, error) {
	return extreme(columns, rows, columnName, true)
}

// Max returns the largest non-empty value in a numeric column.
func Max(columns []Column, rows [][]string, columnName string) (float64, error) {
	return extreme(columns, rows, columnName, false)
}

// GroupBy groups records by a column and aggregates each group with count or sum.
// For sum, sumColumn must name a numeric column. Results are sorted by group key.
func GroupBy(columns []Column, rows [][]string, groupColumn, aggregation, sumColumn string) ([]GroupResult, error) {
	groupIndex, err := columnIndex(columns, groupColumn)
	if err != nil {
		return nil, err
	}
	if aggregation != "count" && aggregation != "sum" {
		return nil, fmt.Errorf("unsupported group_by aggregation %q: use count or sum", aggregation)
	}
	sumIndex := -1
	if aggregation == "sum" {
		sumIndex, err = numericColumnIndex(columns, sumColumn)
		if err != nil {
			return nil, err
		}
	}

	groups := make(map[string]*GroupResult)
	for rowIndex, row := range rows {
		if groupIndex >= len(row) {
			return nil, fmt.Errorf("row %d has no value for column %q", rowIndex+1, groupColumn)
		}
		key := row[groupIndex]
		group := groups[key]
		if group == nil {
			group = &GroupResult{Key: key}
			groups[key] = group
		}
		if aggregation == "count" {
			group.Count++
			continue
		}
		value, present, err := numericValue(row, rowIndex, sumIndex, sumColumn)
		if err != nil {
			return nil, err
		}
		if present {
			group.Sum += value
		}
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]GroupResult, 0, len(keys))
	for _, key := range keys {
		result = append(result, *groups[key])
	}
	return result, nil
}

func extreme(columns []Column, rows [][]string, columnName string, minimum bool) (float64, error) {
	columnIndex, err := numericColumnIndex(columns, columnName)
	if err != nil {
		return 0, err
	}
	var result float64
	found := false
	for rowIndex, row := range rows {
		value, present, err := numericValue(row, rowIndex, columnIndex, columnName)
		if err != nil {
			return 0, err
		}
		if !present {
			continue
		}
		if !found || (minimum && value < result) || (!minimum && value > result) {
			result, found = value, true
		}
	}
	if !found {
		return 0, fmt.Errorf("cannot calculate %s of column %q: no numeric values", map[bool]string{true: "min", false: "max"}[minimum], columnName)
	}
	return result, nil
}

func numericColumnIndex(columns []Column, name string) (int, error) {
	index, err := columnIndex(columns, name)
	if err != nil {
		return 0, err
	}
	if columns[index].Type != "number" {
		return 0, fmt.Errorf("column %q is not numeric", name)
	}
	return index, nil
}

func columnIndex(columns []Column, name string) (int, error) {
	if name == "" {
		return 0, fmt.Errorf("column name is required")
	}
	for index, column := range columns {
		if column.Name == name {
			return index, nil
		}
	}
	return 0, fmt.Errorf("column %q does not exist", name)
}

func numericValue(row []string, rowIndex, columnIndex int, columnName string) (float64, bool, error) {
	if columnIndex >= len(row) {
		return 0, false, fmt.Errorf("row %d has no value for column %q", rowIndex+1, columnName)
	}
	text := strings.TrimSpace(row[columnIndex])
	if text == "" {
		return 0, false, nil
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false, fmt.Errorf("invalid numeric value %q in column %q at row %d", text, columnName, rowIndex+1)
	}
	return value, true, nil
}
