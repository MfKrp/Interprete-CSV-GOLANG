package repository

import (
	"fmt"
	"sync"
	"testing"

	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/service"
)

func TestSaveAndGet(t *testing.T) {
	repo := NewDatasetRepository()
	want, err := repo.Save(Dataset{Filename: "data.csv", Dataset: service.Dataset{Rows: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if want.ID == "" {
		t.Fatal("Save returned empty ID")
	}
	got, ok := repo.Get(want.ID)
	if !ok || got.Filename != want.Filename || got.Rows != want.Rows {
		t.Fatalf("Get() = %#v, %v", got, ok)
	}
	if _, ok := repo.Get("missing"); ok {
		t.Fatal("Get(missing) found a dataset")
	}
}

func TestConcurrentSaveAndGet(t *testing.T) {
	repo := NewDatasetRepository()
	const workers = 64
	var group sync.WaitGroup
	group.Add(workers)
	for i := 0; i < workers; i++ {
		go func(index int) {
			defer group.Done()
			saved, err := repo.Save(Dataset{Filename: fmt.Sprintf("%d.csv", index)})
			if err != nil {
				t.Errorf("Save() error = %v", err)
				return
			}
			if got, ok := repo.Get(saved.ID); !ok || got.Filename != saved.Filename {
				t.Errorf("Get(%s) = %#v, %v", saved.ID, got, ok)
			}
		}(i)
	}
	group.Wait()
}
