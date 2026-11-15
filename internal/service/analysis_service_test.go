package service

import "testing"

type fakeDatasetProvider map[string]Dataset

func (f fakeDatasetProvider) GetDataset(id string) (Dataset, bool) {
	dataset, ok := f[id]
	return dataset, ok
}

func TestAnalysisServiceUsesDatasetID(t *testing.T) {
	datasets := fakeDatasetProvider{"dataset-1": {
		Columns: []Column{{Name: "item", Type: "string"}, {Name: "amount", Type: "number"}},
		Records: [][]string{{" a ", "2"}, {"b", "3"}, {"A", "4"}},
	}}
	analysisService := NewAnalysisService(datasets)
	result, err := analysisService.Execute("dataset-1", AnalysisRequest{Operation: "sum", Column: "amount"})
	if err != nil {
		t.Fatal(err)
	}
	if result.DatasetID != "dataset-1" || result.Value != float64(9) {
		t.Fatalf("result = %#v", result)
	}
	for _, test := range []struct {
		request AnalysisRequest
		want    int
	}{
		{AnalysisRequest{Operation: "count_distinct", Column: "item"}, 2},
		{AnalysisRequest{Operation: "count_filtered", Column: "item", Operator: "eq", Value: "a"}, 2},
		{AnalysisRequest{Operation: "count_filtered", Column: "amount", Operator: "gt", Value: float64(2)}, 2},
	} {
		result, err := analysisService.Execute("dataset-1", test.request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Value != test.want {
			t.Errorf("%s result = %#v, want %d", test.request.Operation, result.Value, test.want)
		}
	}
	if _, err := analysisService.Execute("unknown", AnalysisRequest{Operation: "count_rows"}); err == nil {
		t.Fatal("expected missing dataset error")
	}
	if _, err := analysisService.Execute("dataset-1", AnalysisRequest{Operation: "mystery"}); err == nil {
		t.Fatal("expected unsupported operation error")
	}
}
