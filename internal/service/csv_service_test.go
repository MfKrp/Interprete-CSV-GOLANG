package service

import (
	"strings"
	"testing"
)

func TestReadCSV(t *testing.T) {
	got, err := ReadCSV(strings.NewReader("name,age\nAna,30\nLuis,24\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Rows != 2 || len(got.Columns) != 2 || got.Columns[0].Type != "string" || got.Columns[1].Type != "number" {
		t.Fatalf("unexpected dataset: %#v", got)
	}
	if len(got.Preview) != 2 || got.Preview[0]["name"] != "Ana" {
		t.Fatalf("unexpected preview: %#v", got.Preview)
	}
}

func TestReadCSVPreviewLimit(t *testing.T) {
	got, err := ReadCSV(strings.NewReader("n\n1\n2\n3\n4\n5\n6\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Rows != 6 || len(got.Preview) != PreviewLimit {
		t.Fatalf("rows=%d preview=%d", got.Rows, len(got.Preview))
	}
}

func TestReadCSVRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{"", ",x\na,b\n", "a,a\n1,2\n", "a,b\n1\n"} {
		if _, err := ReadCSV(strings.NewReader(input)); err == nil {
			t.Errorf("ReadCSV(%q) error = nil", input)
		}
	}
}
