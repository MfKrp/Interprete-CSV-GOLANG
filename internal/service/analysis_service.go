package service

import (
	"fmt"

	"github.com/MfKrp/Interprete-CSV-GOLANG/pkg/analysis"
)

// DatasetProvider supplies a stored dataset without coupling the analysis service to storage.
type DatasetProvider interface {
	GetDataset(id string) (Dataset, bool)
}

type AnalysisRequest struct {
	Operation   string `json:"operation"`
	Column      string `json:"column,omitempty"`
	GroupBy     string `json:"group_column,omitempty"`
	Aggregation string `json:"aggregation,omitempty"`
	SumColumn   string `json:"target_column,omitempty"`
	Operator    string `json:"operator,omitempty"`
	Value       any    `json:"value,omitempty"`
}

type AnalysisResult struct {
	DatasetID string                 `json:"dataset_id"`
	Operation string                 `json:"operation"`
	Value     any                    `json:"value,omitempty"`
	Groups    []analysis.GroupResult `json:"groups,omitempty"`
}

type AnalysisService struct {
	datasets DatasetProvider
}

func NewAnalysisService(datasets DatasetProvider) *AnalysisService {
	return &AnalysisService{datasets: datasets}
}

// Execute retrieves a stored dataset by ID and runs one deterministic analysis operation.
func (s *AnalysisService) Execute(datasetID string, request AnalysisRequest) (AnalysisResult, error) {
	dataset, ok := s.datasets.GetDataset(datasetID)
	if !ok {
		return AnalysisResult{}, fmt.Errorf("dataset %q not found", datasetID)
	}
	columns := make([]analysis.Column, len(dataset.Columns))
	for i, column := range dataset.Columns {
		columns[i] = analysis.Column{Name: column.Name, Type: column.Type}
	}
	result := AnalysisResult{DatasetID: datasetID, Operation: request.Operation}
	switch request.Operation {
	case "count_rows":
		result.Value = analysis.CountRows(dataset.Records)
	case "count_distinct":
		value, err := analysis.CountDistinct(columns, dataset.Records, request.Column)
		if err != nil {
			return AnalysisResult{}, err
		}
		result.Value = value
	case "count_filtered":
		value, err := analysis.CountFiltered(columns, dataset.Records, request.Column, request.Operator, request.Value)
		if err != nil {
			return AnalysisResult{}, err
		}
		result.Value = value
	case "sum":
		value, err := analysis.Sum(columns, dataset.Records, request.Column)
		if err != nil {
			return AnalysisResult{}, err
		}
		result.Value = value
	case "average":
		value, err := analysis.Average(columns, dataset.Records, request.Column)
		if err != nil {
			return AnalysisResult{}, err
		}
		result.Value = value
	case "min":
		value, err := analysis.Min(columns, dataset.Records, request.Column)
		if err != nil {
			return AnalysisResult{}, err
		}
		result.Value = value
	case "max":
		value, err := analysis.Max(columns, dataset.Records, request.Column)
		if err != nil {
			return AnalysisResult{}, err
		}
		result.Value = value
	case "group_by":
		groups, err := analysis.GroupBy(columns, dataset.Records, request.GroupBy, request.Aggregation, request.SumColumn)
		if err != nil {
			return AnalysisResult{}, err
		}
		result.Groups = groups
	default:
		return AnalysisResult{}, fmt.Errorf("unsupported operation %q", request.Operation)
	}
	return result, nil
}
