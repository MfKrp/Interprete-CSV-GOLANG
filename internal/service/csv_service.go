package service

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const PreviewLimit = 5

var ErrEmptyCSV = errors.New("CSV is empty")

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Dataset struct {
	Rows    int                 `json:"row_count"`
	Columns []Column            `json:"columns"`
	Preview []map[string]string `json:"preview"`
	Records [][]string          `json:"-"`
}

// ReadCSV validates and reads a CSV with a header row. Records excludes the header.
func ReadCSV(input io.Reader) (Dataset, error) {
	r := csv.NewReader(input)
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	header, err := r.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return Dataset{}, ErrEmptyCSV
		}
		return Dataset{}, fmt.Errorf("read CSV header: %w", err)
	}
	if len(header) == 0 {
		return Dataset{}, fmt.Errorf("CSV header is empty")
	}
	seen := make(map[string]struct{}, len(header))
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
		if header[i] == "" {
			return Dataset{}, fmt.Errorf("CSV header %d is empty", i+1)
		}
		if _, ok := seen[header[i]]; ok {
			return Dataset{}, fmt.Errorf("duplicate CSV header %q", header[i])
		}
		seen[header[i]] = struct{}{}
	}

	data := Dataset{Columns: make([]Column, len(header)), Preview: make([]map[string]string, 0, PreviewLimit)}
	for i, name := range header {
		data.Columns[i] = Column{Name: name, Type: "number"}
	}
	for rowNum := 2; ; rowNum++ {
		row, readErr := r.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return Dataset{}, fmt.Errorf("read CSV row %d: %w", rowNum, readErr)
		}
		if len(row) != len(header) {
			return Dataset{}, fmt.Errorf("CSV row %d has %d fields; expected %d", rowNum, len(row), len(header))
		}
		data.Rows++
		data.Records = append(data.Records, row)
		previewRow := make(map[string]string, len(header))
		for i, value := range row {
			previewRow[header[i]] = value
			if value != "" {
				if _, parseErr := strconv.ParseFloat(strings.TrimSpace(value), 64); parseErr != nil {
					data.Columns[i].Type = "string"
				}
			}
		}
		if len(data.Preview) < PreviewLimit {
			data.Preview = append(data.Preview, previewRow)
		}
	}
	return data, nil
}
