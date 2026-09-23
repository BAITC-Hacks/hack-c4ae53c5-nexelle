package store_test

import (
	"context"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/domain"
	"github.com/BAITC-Hacks/hack-c4ae53c5-nexelle/internal/store"
	"sync"
	"testing"
	"time"
)

func TestSnapshotsAreIsolatedDuringConcurrentCompletion(t *testing.T) {
	ctx := context.Background()
	data := domain.NewDataset()
	data.Employees["E1"] = domain.Employee{EmployeeID: "E1", Skills: map[string]int{"S1": 1}}
	data.Events["EV_036"] = domain.Event{EventID: "EV_036"}
	memory := store.NewMemoryStore()
	if err := memory.ReplaceDataset(ctx, data); err != nil {
		t.Fatal(err)
	}
	first, err := memory.GetDataset(ctx)
	if err != nil {
		t.Fatal(err)
	}
	first.Employees["E1"].Skills["S1"] = 5
	data.Employees["E1"].Skills["S1"] = 4
	group := sync.WaitGroup{}
	cutoff := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for range 20 {
		group.Add(2)
		go func() {
			defer group.Done()
			if _, err := memory.AddCompletedActivity(ctx, "E1", "EV_036", cutoff); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer group.Done()
			snapshot, err := memory.GetDataset(ctx)
			if err != nil {
				t.Error(err)
				return
			}
			for range snapshot.ActivityRecords {
			}
		}()
	}
	group.Wait()
	final, err := memory.GetDataset(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.ActivityRecords) != 0 || len(final.ActivityRecords) != 20 || final.Employees["E1"].Skills["S1"] != 1 {
		t.Fatal("snapshot or imported data shared mutable maps")
	}
}
