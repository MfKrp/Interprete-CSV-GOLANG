package repository

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/service"
)

type Dataset struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	service.Dataset
}

type DatasetRepository struct {
	mu       sync.RWMutex
	datasets map[string]Dataset
}

func NewDatasetRepository() *DatasetRepository {
	return &DatasetRepository{datasets: make(map[string]Dataset)}
}

func (r *DatasetRepository) Save(dataset Dataset) (Dataset, error) {
	if dataset.ID == "" {
		idBytes := make([]byte, 16)
		if _, err := rand.Read(idBytes); err != nil {
			return Dataset{}, err
		}
		dataset.ID = hex.EncodeToString(idBytes)
	}
	r.mu.Lock()
	r.datasets[dataset.ID] = dataset
	r.mu.Unlock()
	return dataset, nil
}

func (r *DatasetRepository) Get(id string) (Dataset, bool) {
	r.mu.RLock()
	dataset, ok := r.datasets[id]
	r.mu.RUnlock()
	return dataset, ok
}

// GetDataset returns the CSV data needed by services, without repository metadata.
func (r *DatasetRepository) GetDataset(id string) (service.Dataset, bool) {
	dataset, ok := r.Get(id)
	if !ok {
		return service.Dataset{}, false
	}
	return dataset.Dataset, true
}
