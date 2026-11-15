package main

import (
	"fmt"
	"log"
	"os"

	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/repository"
	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/service"
)

func main() {
	file, err := os.Open("ventas_demo.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	dataset, err := service.ReadCSV(file)
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewDatasetRepository()
	stored, err := repo.Save(repository.Dataset{
		Filename: "ventas_demo.csv",
		Dataset:  dataset,
	})
	if err != nil {
		log.Fatal(err)
	}

	analysisService := service.NewAnalysisService(repo)
	request := service.AnalysisRequest{
		Operation:   "group_by",
		GroupBy:     "region",
		Aggregation: "sum",
		SumColumn:   "total_venta",
	}
	result, err := analysisService.Execute(stored.ID, request)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Dataset: %s\nOperación: %s (%s de %s agrupada por %s)\n", result.DatasetID, result.Operation, request.Aggregation, request.SumColumn, request.GroupBy)
	for _, group := range result.Groups {
		fmt.Printf("%s: %.2f\n", group.Key, group.Sum)
	}
}
